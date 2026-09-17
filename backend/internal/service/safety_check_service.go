package service

import (
	"errors"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"groundclearance/internal/constants"
	"groundclearance/internal/model"
	"groundclearance/internal/repository"
	"groundclearance/internal/util"
)

type SafetyCheckService struct {
	db             *gorm.DB
	repo           *repository.SafetyCheckRepository
	turnaroundRepo *repository.TurnaroundRepository
	clearanceRepo  *repository.ClearanceDecisionRepository
	unitRepo       *repository.GroundUnitRepository
	logger         *slog.Logger
}

func NewSafetyCheckService(db *gorm.DB, repo *repository.SafetyCheckRepository, turnaroundRepo *repository.TurnaroundRepository,
	clearanceRepo *repository.ClearanceDecisionRepository, unitRepo *repository.GroundUnitRepository, logger *slog.Logger) *SafetyCheckService {
	return &SafetyCheckService{db: db, repo: repo, turnaroundRepo: turnaroundRepo, clearanceRepo: clearanceRepo, unitRepo: unitRepo, logger: logger}
}

func (s *SafetyCheckService) List(page, pageSize int, turnaroundID uint64, result string) ([]model.SafetyCheck, int64, error) {
	result = strings.TrimSpace(result)
	if result != "" && !constants.In(constants.CheckResultValues, result) {
		return nil, 0, util.NewAppError(constants.CodeValidationFailed, "invalid check result filter")
	}
	return s.repo.List(page, pageSize, turnaroundID, result)
}

func (s *SafetyCheckService) Summary() (map[string]any, error) {
	return s.repo.Summary()
}

func (s *SafetyCheckService) Create(check *model.SafetyCheck, actor AuditContext) (*model.SafetyCheck, error) {
	check.CheckCode = strings.ToUpper(strings.TrimSpace(check.CheckCode))
	check.ItemName = strings.TrimSpace(check.ItemName)
	if check.CheckCode == "" || check.ItemName == "" {
		return nil, util.NewAppError(constants.CodeValidationFailed, "check code and item name are required")
	}
	if !constants.IsValidRiskLevel(check.RiskLevel) {
		return nil, util.NewAppError(constants.CodeValidationFailed, "invalid risk level")
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		turnaround, err := s.turnaroundRepo.FindByIDTx(tx, check.TurnaroundID)
		if err != nil {
			return util.NewAppError(constants.CodeValidationFailed, "turnaround does not exist")
		}
		if turnaround.Status != constants.TurnaroundOpen && turnaround.Status != constants.TurnaroundChecking {
			return util.NewAppError(constants.CodeStateConflict, "checks cannot be added after a clearance decision")
		}
		decision, err := s.clearanceRepo.FindByTurnaroundTx(tx, check.TurnaroundID)
		if err != nil || decision.State != constants.ClearancePending {
			return util.NewAppError(constants.CodeStateConflict, "checks cannot be added after a clearance decision")
		}
		if check.GroundUnitID != nil {
			if !turnaroundHasUnit(turnaround, *check.GroundUnitID) {
				return util.NewAppError(constants.CodeValidationFailed, "safety check equipment must be assigned to the turnaround")
			}
			if _, err := s.unitRepo.FindByIDTx(tx, *check.GroundUnitID); err != nil {
				return util.NewAppError(constants.CodeValidationFailed, "ground unit does not exist")
			}
		}
		existing, err := s.repo.ListByTurnaroundTx(tx, check.TurnaroundID)
		if err != nil {
			return err
		}
		for _, item := range existing {
			if item.CheckCode == check.CheckCode {
				return util.NewAppError(constants.CodeConflict, "check code already exists in this turnaround")
			}
		}
		check.Sequence = len(existing) + 1
		check.Result = constants.CheckPending
		if err := s.repo.CreateTx(tx, check); err != nil {
			return err
		}
		return persistTransitionAudit(tx, actor, "SAFETY_CHECK_CREATED", "checks", check.ID, map[string]any{
			"turnaround_id": check.TurnaroundID, "check_code": check.CheckCode, "sequence": check.Sequence,
		})
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogSafetyCheckCreated, "check_id", check.ID, "turnaround_id", check.TurnaroundID)
	return check, nil
}

// BatchReviewItem pairs a selected check with the conclusion chosen for it on
// the inspection desk batch form.
type BatchReviewItem struct {
	CheckID uint64
	Result  string
}

func (s *SafetyCheckService) Review(id, operatorID uint64, result string, evidence []string, remark string, actor AuditContext) (*model.SafetyCheck, error) {
	if !constants.In(constants.CheckResultValues, result) || result == constants.CheckPending {
		return nil, util.NewAppError(constants.CodeValidationFailed, "invalid check result")
	}
	evidence, err := normalizeEvidence(evidence)
	if err != nil {
		return nil, err
	}
	initial, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeNotFound, constants.MsgNotFound)
		}
		return nil, err
	}
	remark = strings.TrimSpace(remark)
	var check *model.SafetyCheck
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := assertTurnaroundReviewableTx(tx, s, initial.TurnaroundID); err != nil {
			return err
		}
		locked, err := s.repo.FindByIDTx(tx, id)
		if err != nil {
			return err
		}
		if locked.Result != constants.CheckPending {
			return util.NewAppError(constants.CodeStateConflict, "check has already been reviewed")
		}
		applyReviewResult(locked, result, evidence, remark, operatorID)
		if err := s.repo.UpdateTx(tx, locked); err != nil {
			return err
		}
		if err := s.ensureTurnaroundCheckingTx(tx, locked.TurnaroundID); err != nil {
			return err
		}
		if err := writeReviewAuditTx(tx, actor, locked, result, evidence, remark, operatorID, nil); err != nil {
			return err
		}
		check = locked
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogSafetyCheckReviewed, "check_id", check.ID, "result", result)
	return check, nil
}

// BatchReview concludes several pending checks atomically. Every item shares a
// single evidence set and remark; if any item is missing, already reviewed or
// belongs to a turnaround that no longer accepts reviews, the whole batch is
// rejected and no conclusion is persisted.
func (s *SafetyCheckService) BatchReview(operatorID uint64, items []BatchReviewItem, evidence []string, remark string, actor AuditContext) ([]model.SafetyCheck, error) {
	results, err := validateBatchItems(items)
	if err != nil {
		return nil, err
	}
	evidence, err = normalizeEvidence(evidence)
	if err != nil {
		return nil, err
	}
	remark = strings.TrimSpace(remark)
	ids := make([]uint64, 0, len(results))
	for id := range results {
		ids = append(ids, id)
	}
	// Resolve turnaround IDs without locks first so locks are taken in the same
	// turnaround-before-check order as single reviews, avoiding cross-request deadlocks.
	initial, err := s.repo.FindByIDs(ids)
	if err != nil {
		return nil, err
	}
	if len(initial) != len(ids) {
		return nil, util.NewAppError(constants.CodeNotFound, constants.MsgNotFound)
	}
	turnaroundIDs := make([]uint64, 0, len(initial))
	seenTurnarounds := make(map[uint64]struct{}, len(initial))
	for _, check := range initial {
		if _, seen := seenTurnarounds[check.TurnaroundID]; !seen {
			seenTurnarounds[check.TurnaroundID] = struct{}{}
			turnaroundIDs = append(turnaroundIDs, check.TurnaroundID)
		}
	}
	sort.Slice(turnaroundIDs, func(i, j int) bool { return turnaroundIDs[i] < turnaroundIDs[j] })
	var reviewed []model.SafetyCheck
	err = s.db.Transaction(func(tx *gorm.DB) error {
		for _, turnaroundID := range turnaroundIDs {
			if err := assertTurnaroundReviewableTx(tx, s, turnaroundID); err != nil {
				return err
			}
		}
		locked, err := s.repo.FindByIDsTx(tx, ids)
		if err != nil {
			return err
		}
		lockedByID := make(map[uint64]*model.SafetyCheck, len(locked))
		for i := range locked {
			lockedByID[locked[i].ID] = &locked[i]
		}
		for _, id := range ids {
			check, ok := lockedByID[id]
			if !ok {
				return util.NewAppError(constants.CodeNotFound, constants.MsgNotFound)
			}
			if check.Result != constants.CheckPending {
				return util.NewAppError(constants.CodeStateConflict, "check has already been reviewed")
			}
			result := results[id]
			applyReviewResult(check, result, evidence, remark, operatorID)
			if err := s.repo.UpdateTx(tx, check); err != nil {
				return err
			}
		}
		for _, turnaroundID := range turnaroundIDs {
			if err := s.ensureTurnaroundCheckingTx(tx, turnaroundID); err != nil {
				return err
			}
		}
		batchNo := 1
		for _, id := range ids {
			check := lockedByID[id]
			if err := writeReviewAuditTx(tx, actor, check, results[id], evidence, remark, operatorID, &batchNo); err != nil {
				return err
			}
		}
		reviewed = locked
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogSafetyCheckBatchReview, "batch_size", len(reviewed))
	return reviewed, nil
}

// assertTurnaroundReviewableTx locks a turnaround and its clearance decision
// and confirms checks on it can still be reviewed.
func assertTurnaroundReviewableTx(tx *gorm.DB, s *SafetyCheckService, turnaroundID uint64) error {
	turnaround, err := s.turnaroundRepo.FindByIDTx(tx, turnaroundID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return util.NewAppError(constants.CodeStateConflict, "checks cannot be reviewed after a clearance decision")
		}
		return err
	}
	if turnaround.Status != constants.TurnaroundOpen && turnaround.Status != constants.TurnaroundChecking {
		return util.NewAppError(constants.CodeStateConflict, "checks cannot be reviewed after a clearance decision")
	}
	decision, err := s.clearanceRepo.FindByTurnaroundTx(tx, turnaroundID)
	if err != nil || decision.State != constants.ClearancePending {
		return util.NewAppError(constants.CodeStateConflict, "checks cannot be reviewed after a clearance decision")
	}
	return nil
}

// ensureTurnaroundCheckingTx moves an open turnaround into checking now that
// at least one of its checks carries a conclusion.
func (s *SafetyCheckService) ensureTurnaroundCheckingTx(tx *gorm.DB, turnaroundID uint64) error {
	turnaround, err := s.turnaroundRepo.FindByIDTx(tx, turnaroundID)
	if err != nil {
		return err
	}
	if turnaround.Status == constants.TurnaroundOpen {
		turnaround.Status = constants.TurnaroundChecking
		if err := s.turnaroundRepo.UpdateStatusTx(tx, turnaround, turnaround.Version); err != nil {
			return err
		}
	}
	return nil
}

func applyReviewResult(check *model.SafetyCheck, result string, evidence []string, remark string, operatorID uint64) {
	now := time.Now()
	check.Result = result
	check.Evidence = model.JSONList(evidence)
	check.Remark = remark
	check.CheckedBy = operatorID
	check.CheckedAt = &now
}

// writeReviewAuditTx persists one audit row per concluded check. Batch reviews
// add batch position metadata so every conclusion stays traceable.
func writeReviewAuditTx(tx *gorm.DB, actor AuditContext, check *model.SafetyCheck, result string,
	evidence []string, remark string, operatorID uint64, batchNo *int) error {
	detail := map[string]any{
		"turnaround_id": check.TurnaroundID, "result": result, "remark": remark,
		"evidence": evidence, "checked_by": operatorID,
	}
	if batchNo != nil {
		detail["batch_review"] = true
		detail["batch_index"] = *batchNo
		*batchNo++
	}
	return persistTransitionAudit(tx, actor, "SAFETY_CHECK_REVIEW", "checks", check.ID, detail)
}

func turnaroundHasUnit(row *model.Turnaround, unitID uint64) bool {
	for _, rawID := range row.GroundUnitIDs {
		parsed, err := strconv.ParseUint(rawID, 10, 64)
		if err == nil && parsed == unitID {
			return true
		}
	}
	return false
}

// validateBatchItems checks batch size, individual conclusions and duplicate
// check selections, returning check_id -> result for the rest of the flow.
func validateBatchItems(items []BatchReviewItem) (map[uint64]string, error) {
	if len(items) == 0 || len(items) > 100 {
		return nil, util.NewAppError(constants.CodeValidationFailed, "between 1 and 100 checks can be reviewed in one batch")
	}
	results := make(map[uint64]string, len(items))
	for _, item := range items {
		if item.CheckID == 0 || !constants.In(constants.CheckResultValues, item.Result) || item.Result == constants.CheckPending {
			return nil, util.NewAppError(constants.CodeValidationFailed, "invalid check result")
		}
		if _, duplicate := results[item.CheckID]; duplicate {
			return nil, util.NewAppError(constants.CodeValidationFailed, "check appears more than once in this batch")
		}
		results[item.CheckID] = item.Result
	}
	return results, nil
}

func normalizeEvidence(items []string) ([]string, error) {
	if len(items) == 0 || len(items) > 20 {
		return nil, util.NewAppError(constants.CodeValidationFailed, "between 1 and 20 evidence references are required")
	}
	normalized := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || len(item) > 255 {
			return nil, util.NewAppError(constants.CodeValidationFailed, "invalid evidence reference")
		}
		if _, duplicate := seen[item]; duplicate {
			continue
		}
		seen[item] = struct{}{}
		normalized = append(normalized, item)
	}
	if len(normalized) == 0 {
		return nil, util.NewAppError(constants.CodeValidationFailed, "evidence is required")
	}
	return normalized, nil
}
