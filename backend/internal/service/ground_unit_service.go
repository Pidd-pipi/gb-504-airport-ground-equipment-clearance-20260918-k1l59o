package service

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"
	"groundclearance/internal/constants"
	"groundclearance/internal/model"
	"groundclearance/internal/repository"
	"groundclearance/internal/util"
)

type GroundUnitService struct {
	db             *gorm.DB
	repo           *repository.GroundUnitRepository
	turnaroundRepo *repository.TurnaroundRepository
	clearanceRepo  *repository.ClearanceDecisionRepository
	logger         *slog.Logger
}

func NewGroundUnitService(db *gorm.DB, repo *repository.GroundUnitRepository, turnaroundRepo *repository.TurnaroundRepository,
	clearanceRepo *repository.ClearanceDecisionRepository, logger *slog.Logger) *GroundUnitService {
	return &GroundUnitService{db: db, repo: repo, turnaroundRepo: turnaroundRepo, clearanceRepo: clearanceRepo, logger: logger}
}

func (s *GroundUnitService) CreateUnit(unit *model.GroundUnit, actor AuditContext) (*model.GroundUnit, error) {
	unit.UnitCode = strings.ToUpper(strings.TrimSpace(unit.UnitCode))
	unit.Name = strings.TrimSpace(unit.Name)
	unit.UnitType = strings.TrimSpace(unit.UnitType)
	unit.Stand = strings.ToUpper(strings.TrimSpace(unit.Stand))
	unit.Notes = strings.TrimSpace(unit.Notes)
	if unit.UnitCode == "" || unit.Name == "" || unit.UnitType == "" || unit.Stand == "" {
		return nil, util.NewAppError(constants.CodeValidationFailed, "unit code, name, type and stand are required")
	}
	if !constants.IsValidUnitState(unit.State) {
		return nil, util.NewAppError(constants.CodeValidationFailed, "invalid unit state")
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.CreateTx(tx, unit); err != nil {
			return err
		}
		return persistTransitionAudit(tx, actor, "GROUND_UNIT_CREATED", "ground-units", unit.ID, map[string]any{
			"unit_code": unit.UnitCode, "state": unit.State, "stand": unit.Stand,
		})
	}); err != nil {
		return nil, util.Wrap(err, "GroundUnit[unit_code=%s] create failed", unit.UnitCode)
	}
	s.logger.Info(constants.LogGroundUnitCreated, "ground_unit_id", unit.ID, "unit_code", unit.UnitCode)
	return unit, nil
}

func (s *GroundUnitService) List(page, pageSize int, state, unitType, search string) ([]model.GroundUnit, int64, error) {
	state = strings.TrimSpace(state)
	unitType = strings.TrimSpace(unitType)
	search = strings.TrimSpace(search)
	if state != "" && !constants.IsValidUnitState(state) {
		return nil, 0, util.NewAppError(constants.CodeValidationFailed, "invalid unit state filter")
	}
	if len(unitType) > 40 || len(search) > 100 {
		return nil, 0, util.NewAppError(constants.CodeValidationFailed, "filter value is too long")
	}
	return s.repo.List(page, pageSize, state, unitType, search)
}

func (s *GroundUnitService) Summary() (map[string]any, error) {
	return s.repo.Summary()
}

func (s *GroundUnitService) Get(id uint64) (*model.GroundUnit, error) {
	unit, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeNotFound, constants.MsgNotFound)
		}
		return nil, util.Wrap(err, "GroundUnit[id=%d] get failed", id)
	}
	return unit, nil
}

func (s *GroundUnitService) ChangeState(id uint64, state, notes string, version int, role string, actor AuditContext) (*model.GroundUnit, error) {
	if !constants.IsValidUnitState(state) {
		return nil, util.NewAppError(constants.CodeValidationFailed, "invalid unit state")
	}
	notes = strings.TrimSpace(notes)
	if state == constants.UnitAvailable && notes == "" {
		return nil, util.NewAppError(constants.CodeValidationFailed, "recovery evidence is required when restoring equipment")
	}
	var unit *model.GroundUnit
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var turnarounds []model.Turnaround
		decisions := make(map[uint64]*model.ClearanceDecision)
		if state != constants.UnitAvailable {
			var err error
			turnarounds, err = s.turnaroundRepo.FindActiveByGroundUnitTx(tx, id)
			if err != nil {
				return err
			}
			for index := range turnarounds {
				decision, findErr := s.clearanceRepo.FindByTurnaroundTx(tx, turnarounds[index].ID)
				if findErr != nil {
					return findErr
				}
				decisions[turnarounds[index].ID] = decision
			}
		}
		locked, err := s.repo.FindByIDTx(tx, id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, constants.MsgNotFound)
			}
			return err
		}
		if locked.Version != version {
			return util.NewAppError(constants.CodeConflict, "equipment was updated by another operator")
		}
		if !allowedUnitTransition(locked.State, state) {
			return util.NewAppError(constants.CodeStateConflict, "equipment state transition is not allowed")
		}
		if role == constants.RoleInspector && state != constants.UnitInspection && state != constants.UnitBlocked {
			return util.NewAppError(constants.CodeForbidden, constants.MsgForbidden)
		}
		previous := locked.State
		locked.State = state
		locked.Notes = notes
		if err := s.repo.UpdateTx(tx, locked); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				return util.NewAppError(constants.CodeConflict, "equipment was updated by another operator")
			}
			return err
		}
		for _, turnaround := range turnarounds {
			decision := decisions[turnaround.ID]
			if decision.State != constants.ClearanceCleared && decision.State != constants.ClearanceRestricted {
				continue
			}
			previousDecision := decision.State
			decision.PreviousState = previousDecision
			decision.State = constants.ClearanceRevoked
			decision.Reason = fmt.Sprintf("assigned equipment %s changed to %s: %s", locked.UnitCode, state, notes)
			decision.Evidence = append(decision.Evidence, "ground-unit:"+locked.UnitCode)
			decision.OperatorID = actor.OperatorID
			decision.RequestID = actor.RequestID
			decision.DecidedAt = time.Now()
			if err := s.clearanceRepo.SaveTx(tx, decision); err != nil {
				return err
			}
			if err := persistTransitionAudit(tx, actor, "CLEARANCE_TRANSITION", "clearance", decision.ID, map[string]any{
				"previous_state": previousDecision, "state": constants.ClearanceRevoked,
				"reason": decision.Reason, "evidence": decision.Evidence, "trigger_ground_unit_id": locked.ID,
			}); err != nil {
				return err
			}
		}
		if err := persistTransitionAudit(tx, actor, "GROUND_UNIT_STATE_TRANSITION", "ground-units", locked.ID, map[string]any{
			"previous_state": previous, "state": state, "notes": notes, "version": locked.Version,
		}); err != nil {
			return err
		}
		unit = locked
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogGroundUnitStateChanged, "ground_unit_id", unit.ID, "state", state)
	return unit, nil
}

func allowedUnitTransition(from, to string) bool {
	if from == to || from == constants.UnitRetired {
		return false
	}
	switch from {
	case constants.UnitAvailable:
		return to == constants.UnitInspection || to == constants.UnitBlocked || to == constants.UnitRetired
	case constants.UnitInspection, constants.UnitBlocked:
		return to == constants.UnitAvailable || to == constants.UnitInspection || to == constants.UnitBlocked || to == constants.UnitRetired
	default:
		return false
	}
}
