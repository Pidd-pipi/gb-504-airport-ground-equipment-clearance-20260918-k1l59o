package repository

import (
	"errors"
	"fmt"
	"time"

	"groundclearance/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ClearanceDecisionRepository struct{ db *gorm.DB }

func NewClearanceDecisionRepository(db *gorm.DB) *ClearanceDecisionRepository {
	return &ClearanceDecisionRepository{db: db}
}

func (r *ClearanceDecisionRepository) List(page, pageSize int, state string) ([]model.ClearanceDecision, int64, error) {
	var rows []model.ClearanceDecision
	var total int64
	query := r.db.Model(&model.ClearanceDecision{})
	if state != "" {
		query = query.Where("state = ?", state)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count clearance decisions: %w", err)
	}
	if err := query.Order("decided_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list clearance decisions: %w", err)
	}
	return rows, total, nil
}

func (r *ClearanceDecisionRepository) FindByID(id uint64) (*model.ClearanceDecision, error) {
	var row model.ClearanceDecision
	if err := r.db.First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find clearance decision: %w", err)
	}
	return &row, nil
}

func (r *ClearanceDecisionRepository) FindByTurnaround(turnaroundID uint64) (*model.ClearanceDecision, error) {
	var row model.ClearanceDecision
	if err := r.db.Where("turnaround_id = ?", turnaroundID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find clearance by turnaround: %w", err)
	}
	return &row, nil
}

func (r *ClearanceDecisionRepository) FindByTurnaroundTx(tx *gorm.DB, turnaroundID uint64) (*model.ClearanceDecision, error) {
	var row model.ClearanceDecision
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("turnaround_id = ?", turnaroundID).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find clearance by turnaround: %w", err)
	}
	return &row, nil
}

func (r *ClearanceDecisionRepository) SaveTx(tx *gorm.DB, row *model.ClearanceDecision) error {
	if err := tx.Save(row).Error; err != nil {
		return fmt.Errorf("save clearance decision: %w", err)
	}
	return nil
}

// Summary keeps the release desk counters consistent with persisted state.
func (r *ClearanceDecisionRepository) Summary(now time.Time) (map[string]any, error) {
	type countRow struct {
		Key   string
		Count int64
	}
	var rows []countRow
	if err := r.db.Model(&model.ClearanceDecision{}).
		Select("state AS key, COUNT(*) AS count").Group("state").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("group clearance decisions: %w", err)
	}
	states := map[string]int64{"pending": 0, "cleared": 0, "restricted": 0, "revoked": 0}
	var total int64
	for _, row := range rows {
		states[row.Key] = row.Count
		total += row.Count
	}
	var changedToday int64
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if err := r.db.Model(&model.ClearanceDecision{}).
		Where("decided_at >= ?", dayStart).Count(&changedToday).Error; err != nil {
		return nil, fmt.Errorf("count daily clearance decisions: %w", err)
	}
	return map[string]any{"total": total, "states": states, "changed_today": changedToday}, nil
}
