package repository

import (
	"errors"
	"fmt"

	"groundclearance/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SafetyCheckRepository struct{ db *gorm.DB }

func NewSafetyCheckRepository(db *gorm.DB) *SafetyCheckRepository {
	return &SafetyCheckRepository{db: db}
}

func (r *SafetyCheckRepository) Create(check *model.SafetyCheck) error {
	if err := r.db.Create(check).Error; err != nil {
		return fmt.Errorf("create safety check: %w", err)
	}
	return nil
}

func (r *SafetyCheckRepository) CreateTx(tx *gorm.DB, check *model.SafetyCheck) error {
	if err := tx.Create(check).Error; err != nil {
		return fmt.Errorf("create safety check: %w", err)
	}
	return nil
}

func (r *SafetyCheckRepository) CreateManyTx(tx *gorm.DB, checks []model.SafetyCheck) error {
	if len(checks) == 0 {
		return nil
	}
	if err := tx.Create(&checks).Error; err != nil {
		return fmt.Errorf("create turnaround checks: %w", err)
	}
	return nil
}

func (r *SafetyCheckRepository) List(page, pageSize int, turnaroundID uint64, result string) ([]model.SafetyCheck, int64, error) {
	var rows []model.SafetyCheck
	var total int64
	query := r.db.Model(&model.SafetyCheck{})
	if turnaroundID > 0 {
		query = query.Where("turnaround_id = ?", turnaroundID)
	}
	if result != "" {
		query = query.Where("result = ?", result)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count safety checks: %w", err)
	}
	if err := query.Order("turnaround_id DESC, sequence ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list safety checks: %w", err)
	}
	return rows, total, nil
}

func (r *SafetyCheckRepository) ListByTurnaround(turnaroundID uint64) ([]model.SafetyCheck, error) {
	var rows []model.SafetyCheck
	if err := r.db.Where("turnaround_id = ?", turnaroundID).Order("sequence ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list turnaround checks: %w", err)
	}
	return rows, nil
}

func (r *SafetyCheckRepository) ListByTurnaroundTx(tx *gorm.DB, turnaroundID uint64) ([]model.SafetyCheck, error) {
	var rows []model.SafetyCheck
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("turnaround_id = ?", turnaroundID).
		Order("sequence ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("lock turnaround checks: %w", err)
	}
	return rows, nil
}

func (r *SafetyCheckRepository) FindByID(id uint64) (*model.SafetyCheck, error) {
	var check model.SafetyCheck
	if err := r.db.First(&check, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find safety check: %w", err)
	}
	return &check, nil
}

func (r *SafetyCheckRepository) FindByIDTx(tx *gorm.DB, id uint64) (*model.SafetyCheck, error) {
	var check model.SafetyCheck
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&check, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find safety check: %w", err)
	}
	return &check, nil
}

func (r *SafetyCheckRepository) UpdateTx(tx *gorm.DB, check *model.SafetyCheck) error {
	if err := tx.Save(check).Error; err != nil {
		return fmt.Errorf("update safety check: %w", err)
	}
	return nil
}

func (r *SafetyCheckRepository) CountFailed(turnaroundID uint64) (int64, error) {
	var count int64
	if err := r.db.Model(&model.SafetyCheck{}).Where("turnaround_id = ? AND result = ?", turnaroundID, "failed").Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count failed checks: %w", err)
	}
	return count, nil
}

func (r *SafetyCheckRepository) CountPending(turnaroundID uint64) (int64, error) {
	var count int64
	if err := r.db.Model(&model.SafetyCheck{}).Where("turnaround_id = ? AND result = ?", turnaroundID, "pending").Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count pending checks: %w", err)
	}
	return count, nil
}

// Summary reports progress and high-risk backlog for the inspection desk.
func (r *SafetyCheckRepository) Summary() (map[string]any, error) {
	type countRow struct {
		Key   string
		Count int64
	}
	var rows []countRow
	if err := r.db.Model(&model.SafetyCheck{}).
		Select("result AS key, COUNT(*) AS count").Group("result").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("group checks by result: %w", err)
	}
	results := map[string]int64{"pending": 0, "passed": 0, "failed": 0}
	var total int64
	for _, row := range rows {
		results[row.Key] = row.Count
		total += row.Count
	}
	var criticalPending int64
	if err := r.db.Model(&model.SafetyCheck{}).
		Where("result = ? AND risk_level IN ?", "pending", []string{"high", "critical"}).
		Count(&criticalPending).Error; err != nil {
		return nil, fmt.Errorf("count critical pending checks: %w", err)
	}
	completion := 0.0
	if total > 0 {
		completion = float64(results["passed"]+results["failed"]) / float64(total) * 100
	}
	return map[string]any{
		"total": total, "results": results, "critical_pending": criticalPending,
		"completion_percent": completion,
	}, nil
}
