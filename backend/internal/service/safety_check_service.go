package service

import (
	"errors"
	"log/slog"
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
