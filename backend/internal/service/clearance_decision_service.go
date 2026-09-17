package service

import (
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"groundclearance/internal/constants"
	"groundclearance/internal/model"
	"groundclearance/internal/repository"
	"groundclearance/internal/util"

	"gorm.io/gorm"
)

type ClearanceDecisionService struct {
	db             *gorm.DB
	repo           *repository.ClearanceDecisionRepository
	turnaroundRepo *repository.TurnaroundRepository
	checkRepo      *repository.SafetyCheckRepository
	unitRepo       *repository.GroundUnitRepository
	logger         *slog.Logger
}

func NewClearanceDecisionService(db *gorm.DB, repo *repository.ClearanceDecisionRepository,
	turnaroundRepo *repository.TurnaroundRepository, checkRepo *repository.SafetyCheckRepository,
	unitRepo *repository.GroundUnitRepository, logger *slog.Logger) *ClearanceDecisionService {
	return &ClearanceDecisionService{db: db, repo: repo, turnaroundRepo: turnaroundRepo, checkRepo: checkRepo, unitRepo: unitRepo, logger: logger}
}

func (s *ClearanceDecisionService) List(page, pageSize int, state string) ([]model.ClearanceDecision, int64, error) {
	state = strings.TrimSpace(state)
	if state != "" && !constants.IsValidClearanceState(state) {
		return nil, 0, util.NewAppError(constants.CodeValidationFailed, "invalid clearance state filter")
	}
	return s.repo.List(page, pageSize, state)
}

func (s *ClearanceDecisionService) Summary() (map[string]any, error) {
	return s.repo.Summary(time.Now())
}

func (s *ClearanceDecisionService) Get(id uint64) (*model.ClearanceDecision, error) {
	decision, err := s.repo.FindByID(id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, util.NewAppError(constants.CodeNotFound, constants.MsgNotFound)
	}
	return decision, err
}

func (s *ClearanceDecisionService) Decide(turnaroundID, operatorID uint64, state, restrictions, reason string,
	evidence []string, requestID, operatorName, ip string) (*model.ClearanceDecision, error) {
	if !constants.IsValidClearanceState(state) || state == constants.ClearancePending {
		return nil, util.NewAppError(constants.CodeValidationFailed, "invalid clearance state")
	}
	reason = strings.TrimSpace(reason)
	restrictions = strings.TrimSpace(restrictions)
	if reason == "" {
		return nil, util.NewAppError(constants.CodeValidationFailed, "decision reason is required")
	}
	evidence, err := normalizeEvidence(evidence)
	if err != nil {
		return nil, err
	}
	if state == constants.ClearanceRestricted && restrictions == "" {
		return nil, util.NewAppError(constants.CodeValidationFailed, "restrictions are required")
	}
	if state == constants.ClearanceCleared {
		restrictions = ""
	}
	var decision *model.ClearanceDecision
	err = s.db.Transaction(func(tx *gorm.DB) error {
		turnaround, err := s.turnaroundRepo.FindByIDTx(tx, turnaroundID)
		if err != nil {
			return util.NewAppError(constants.CodeNotFound, constants.MsgNotFound)
		}
		current, err := s.repo.FindByTurnaroundTx(tx, turnaroundID)
		if err != nil {
			return err
		}
		if !allowedClearanceTransition(current.State, state) {
			return util.NewAppError(constants.CodeStateConflict, "clearance transition is not allowed")
		}
		if turnaround.Status == constants.TurnaroundCompleted {
			return util.NewAppError(constants.CodeStateConflict, "completed turnarounds cannot be changed")
		}
		checks, err := s.checkRepo.ListByTurnaroundTx(tx, turnaroundID)
		if err != nil {
			return err
		}
		if state != constants.ClearanceRevoked {
			if turnaround.Status != constants.TurnaroundChecking {
				return util.NewAppError(constants.CodeStateConflict, "turnaround must be in checking state before clearance")
			}
			if len(checks) == 0 {
				return util.NewAppError(constants.CodeStateConflict, "at least one safety check is required")
			}
			for _, check := range checks {
				if check.Result == constants.CheckPending {
					return util.NewAppError(constants.CodeStateConflict, "all safety checks must be reviewed before clearance")
				}
				if state == constants.ClearanceCleared && check.Result == constants.CheckFailed {
					return util.NewAppError(constants.CodeStateConflict, "failed checks prevent full clearance")
				}
			}
		}
		if state == constants.ClearanceCleared {
			for _, rawID := range turnaround.GroundUnitIDs {
				unitID, parseErr := strconv.ParseUint(rawID, 10, 64)
				if parseErr != nil || unitID == 0 {
					return util.NewAppError(constants.CodeStateConflict, "turnaround contains an invalid ground unit")
				}
				unit, findErr := s.unitRepo.FindByIDTx(tx, unitID)
				if findErr != nil || unit.State != constants.UnitAvailable {
					return util.NewAppError(constants.CodeStateConflict, "all assigned ground units must be available for full clearance")
				}
			}
		}
		previous := current.State
		current.PreviousState = previous
		current.State = state
		current.Restrictions = restrictions
		current.Reason = reason
		current.Evidence = model.JSONList(evidence)
		current.OperatorID = operatorID
		current.RequestID = requestID
		current.DecidedAt = time.Now()
		if err := s.repo.SaveTx(tx, current); err != nil {
			return err
		}
		turnaround.Status = constants.TurnaroundDecisioned
		turnaround.Version++
		if err := s.turnaroundRepo.SaveTx(tx, turnaround); err != nil {
			return err
		}
		detail, _ := json.Marshal(map[string]any{
			"previous_state": previous, "state": state, "reason": reason,
			"restrictions": restrictions, "evidence": evidence, "request_id": requestID,
		})
		audit := &model.AuditLog{OperatorID: operatorID, OperatorName: operatorName, Action: "CLEARANCE_TRANSITION",
			EntityType: "clearance", EntityID: strconv.FormatUint(current.ID, 10), Detail: string(detail), IP: ip, CreatedAt: time.Now()}
		if err := tx.Create(audit).Error; err != nil {
			return err
		}
		decision = current
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogClearanceChanged, "turnaround_id", turnaroundID, "state", state, "operator_id", operatorID)
	return decision, nil
}

func allowedClearanceTransition(from, to string) bool {
	if from == constants.ClearancePending {
		return to == constants.ClearanceCleared || to == constants.ClearanceRestricted || to == constants.ClearanceRevoked
	}
	return (from == constants.ClearanceCleared || from == constants.ClearanceRestricted) && to == constants.ClearanceRevoked
}
