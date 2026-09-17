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
	var check *model.SafetyCheck
	err = s.db.Transaction(func(tx *gorm.DB) error {
		turnaround, err := s.turnaroundRepo.FindByIDTx(tx, initial.TurnaroundID)
		if err != nil {
			return err
		}
		if turnaround.Status != constants.TurnaroundOpen && turnaround.Status != constants.TurnaroundChecking {
			return util.NewAppError(constants.CodeStateConflict, "checks cannot be reviewed after a clearance decision")
		}
		decision, err := s.clearanceRepo.FindByTurnaroundTx(tx, initial.TurnaroundID)
		if err != nil || decision.State != constants.ClearancePending {
			return util.NewAppError(constants.CodeStateConflict, "checks cannot be reviewed after a clearance decision")
		}
		locked, err := s.repo.FindByIDTx(tx, id)
		if err != nil {
			return err
		}
		if locked.TurnaroundID != turnaround.ID || locked.Result != constants.CheckPending {
			return util.NewAppError(constants.CodeStateConflict, "check has already been reviewed")
		}
		now := time.Now()
		locked.Result = result
		locked.Evidence = model.JSONList(evidence)
		locked.Remark = strings.TrimSpace(remark)
		locked.CheckedBy = operatorID
		locked.CheckedAt = &now
		if err := s.repo.UpdateTx(tx, locked); err != nil {
			return err
		}
		if turnaround.Status == constants.TurnaroundOpen {
			turnaround.Status = constants.TurnaroundChecking
			if err := s.turnaroundRepo.UpdateStatusTx(tx, turnaround, turnaround.Version); err != nil {
				return err
			}
		}
		if err := persistTransitionAudit(tx, actor, "SAFETY_CHECK_REVIEW", "checks", locked.ID, map[string]any{
			"turnaround_id": locked.TurnaroundID, "result": result, "remark": locked.Remark,
			"evidence": evidence, "checked_by": operatorID,
		}); err != nil {
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

// BatchReviewDecision pairs one selected check with its per-item conclusion.
type BatchReviewDecision struct {
	ID     uint64
	Result string
}

// BatchReview records several check conclusions that share one remark and one
// evidence set. Validation and every write happen in one transaction: if any
// item is missing, already reviewed, belongs to a decisioned turnaround, or
// the shared evidence is invalid, the whole batch fails without partial
// conclusions.
func (s *SafetyCheckService) BatchReview(decisions []BatchReviewDecision, evidence []string, remark string, operatorID uint64, actor AuditContext) ([]model.SafetyCheck, error) {
	if len(decisions) == 0 || len(decisions) > 50 {
		return nil, util.NewAppError(constants.CodeValidationFailed, "between 1 and 50 checks are required")
	}
	ids := make([]uint64, 0, len(decisions))
	resultByID := make(map[uint64]string, len(decisions))
	for _, decision := range decisions {
		if decision.ID == 0 || !constants.In(constants.CheckResultValues, decision.Result) || decision.Result == constants.CheckPending {
			return nil, util.NewAppError(constants.CodeValidationFailed, "invalid batch review item")
		}
		if _, duplicated := resultByID[decision.ID]; duplicated {
			return nil, util.NewAppError(constants.CodeValidationFailed, "duplicate check in batch review")
		}
		resultByID[decision.ID] = decision.Result
		ids = append(ids, decision.ID)
	}
	normalizedEvidence, err := normalizeEvidence(evidence)
	if err != nil {
		return nil, err
	}
	trimmedRemark := strings.TrimSpace(remark)
	updated := make([]model.SafetyCheck, 0, len(decisions))
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Snapshot preload to resolve turnaround IDs; rows are re-locked and
		// re-validated below so concurrent reviews cannot sneak through.
		preload, err := s.repo.FindByIDsTx(tx, ids, false)
		if err != nil {
			return err
		}
		if len(preload) != len(ids) {
			return util.NewAppError(constants.CodeNotFound, "one or more checks do not exist")
		}
		turnaroundIDs := make(map[uint64]struct{})
		for _, check := range preload {
			turnaroundIDs[check.TurnaroundID] = struct{}{}
		}
		sortedTurnaroundIDs := make([]uint64, 0, len(turnaroundIDs))
		for id := range turnaroundIDs {
			sortedTurnaroundIDs = append(sortedTurnaroundIDs, id)
		}
		sort.Slice(sortedTurnaroundIDs, func(i, j int) bool { return sortedTurnaroundIDs[i] < sortedTurnaroundIDs[j] })
		// Lock affected turnarounds and their clearance decisions first, in a
		// fixed order, to serialize with single-item reviews and other batches.
		turnarounds := make(map[uint64]*model.Turnaround, len(sortedTurnaroundIDs))
		for _, turnaroundID := range sortedTurnaroundIDs {
			turnaround, err := s.turnaroundRepo.FindByIDTx(tx, turnaroundID)
			if err != nil {
				return err
			}
			if turnaround.Status != constants.TurnaroundOpen && turnaround.Status != constants.TurnaroundChecking {
				return util.NewAppError(constants.CodeStateConflict, "checks cannot be reviewed after a clearance decision")
			}
			decision, err := s.clearanceRepo.FindByTurnaroundTx(tx, turnaroundID)
			if err != nil || decision.State != constants.ClearancePending {
				return util.NewAppError(constants.CodeStateConflict, "checks cannot be reviewed after a clearance decision")
			}
			turnarounds[turnaroundID] = turnaround
		}
		// Lock check rows in ascending ID order together with single reviews.
		lockedRows, err := s.repo.FindByIDsTx(tx, ids, true)
		if err != nil {
			return err
		}
		lockedByID := make(map[uint64]*model.SafetyCheck, len(lockedRows))
		for i := range lockedRows {
			lockedByID[lockedRows[i].ID] = &lockedRows[i]
		}
		now := time.Now()
		promotedTurnaround := make(map[uint64]struct{})
		for _, decision := range decisions {
			locked := lockedByID[decision.ID]
			if locked == nil {
				return util.NewAppError(constants.CodeNotFound, "one or more checks do not exist")
			}
			turnaround := turnarounds[locked.TurnaroundID]
			if turnaround == nil {
				return util.NewAppError(constants.CodeStateConflict, "check has already been reviewed")
			}
			if locked.Result != constants.CheckPending {
				return util.NewAppError(constants.CodeStateConflict, "check has already been reviewed")
			}
			locked.Result = decision.Result
			locked.Evidence = model.JSONList(normalizedEvidence)
			locked.Remark = trimmedRemark
			locked.CheckedBy = operatorID
			locked.CheckedAt = &now
			if err := s.repo.UpdateTx(tx, locked); err != nil {
				return err
			}
			if turnaround.Status == constants.TurnaroundOpen {
				if _, promoted := promotedTurnaround[turnaround.ID]; !promoted {
					turnaround.Status = constants.TurnaroundChecking
					if err := s.turnaroundRepo.UpdateStatusTx(tx, turnaround, turnaround.Version); err != nil {
						return err
					}
					promotedTurnaround[turnaround.ID] = struct{}{}
				}
			}
			if err := persistTransitionAudit(tx, actor, "SAFETY_CHECK_REVIEW", "checks", locked.ID, map[string]any{
				"turnaround_id": locked.TurnaroundID, "result": decision.Result, "remark": trimmedRemark,
				"evidence": normalizedEvidence, "checked_by": operatorID,
				"batch": true, "batch_size": len(decisions),
			}); err != nil {
				return err
			}
		}
		updated = lockedRows
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogSafetyCheckReviewed, "batch_size", len(decisions), "reviewed", len(updated))
	// Return conclusions in the order the caller submitted them.
	ordered := make([]model.SafetyCheck, 0, len(decisions))
	byID := make(map[uint64]model.SafetyCheck, len(updated))
	for _, check := range updated {
		byID[check.ID] = check
	}
	for _, decision := range decisions {
		ordered = append(ordered, byID[decision.ID])
	}
	return ordered, nil
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
