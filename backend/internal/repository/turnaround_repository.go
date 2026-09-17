package repository

import (
	"errors"
	"fmt"
	"time"

	"groundclearance/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TurnaroundRepository struct{ db *gorm.DB }

func NewTurnaroundRepository(db *gorm.DB) *TurnaroundRepository {
	return &TurnaroundRepository{db: db}
}

func (r *TurnaroundRepository) CreateTx(tx *gorm.DB, row *model.Turnaround) error {
	if err := tx.Create(row).Error; err != nil {
		return fmt.Errorf("create turnaround: %w", err)
	}
	return nil
}

func (r *TurnaroundRepository) FindByID(id uint64) (*model.Turnaround, error) {
	var row model.Turnaround
	if err := r.db.First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find turnaround: %w", err)
	}
	return &row, nil
}

func (r *TurnaroundRepository) FindByIDTx(tx *gorm.DB, id uint64) (*model.Turnaround, error) {
	var row model.Turnaround
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lock turnaround: %w", err)
	}
	return &row, nil
}

func (r *TurnaroundRepository) FindActiveByGroundUnitTx(tx *gorm.DB, unitID uint64) ([]model.Turnaround, error) {
	var rows []model.Turnaround
	unitJSON := fmt.Sprintf(`["%d"]`, unitID)
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("status <> ? AND ground_unit_ids @> ?::jsonb", "completed", unitJSON).
		Order("id ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("lock active turnarounds by ground unit: %w", err)
	}
	return rows, nil
}

func (r *TurnaroundRepository) List(page, pageSize int, status, risk, search string) ([]model.Turnaround, int64, error) {
	var rows []model.Turnaround
	var total int64
	query := r.db.Model(&model.Turnaround{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if risk != "" {
		query = query.Where("risk_level = ?", risk)
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("flight_no ILIKE ? OR stand ILIKE ?", like, like)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count turnarounds: %w", err)
	}
	if err := query.Order("scheduled_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list turnarounds: %w", err)
	}
	return rows, total, nil
}

func (r *TurnaroundRepository) UpdateStatusTx(tx *gorm.DB, row *model.Turnaround, expectedVersion int) error {
	now := time.Now()
	result := tx.Model(&model.Turnaround{}).Where("id = ? AND version = ?", row.ID, expectedVersion).
		Updates(map[string]any{"status": row.Status, "version": expectedVersion + 1, "updated_at": now})
	if result.Error != nil {
		return fmt.Errorf("update turnaround status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrConflict
	}
	row.Version = expectedVersion + 1
	row.UpdatedAt = now
	return nil
}

func (r *TurnaroundRepository) SaveTx(tx *gorm.DB, row *model.Turnaround) error {
	if err := tx.Save(row).Error; err != nil {
		return fmt.Errorf("save turnaround: %w", err)
	}
	return nil
}

// Summary aggregates the active board without loading every turnaround row.
func (r *TurnaroundRepository) Summary(now time.Time) (map[string]any, error) {
	type countRow struct {
		Key   string
		Count int64
	}
	var statusRows []countRow
	if err := r.db.Model(&model.Turnaround{}).
		Select("status AS key, COUNT(*) AS count").Group("status").Scan(&statusRows).Error; err != nil {
		return nil, fmt.Errorf("group turnarounds by status: %w", err)
	}
	statuses := map[string]int64{"open": 0, "checking": 0, "decisioned": 0, "completed": 0}
	var total int64
	for _, row := range statusRows {
		statuses[row.Key] = row.Count
		total += row.Count
	}
	var riskRows []countRow
	if err := r.db.Model(&model.Turnaround{}).
		Select("risk_level AS key, COUNT(*) AS count").Group("risk_level").Scan(&riskRows).Error; err != nil {
		return nil, fmt.Errorf("group turnarounds by risk: %w", err)
	}
	risks := map[string]int64{"low": 0, "medium": 0, "high": 0, "critical": 0}
	for _, row := range riskRows {
		risks[row.Key] = row.Count
	}
	var dueSoon int64
	if err := r.db.Model(&model.Turnaround{}).
		Where("scheduled_at >= ? AND scheduled_at <= ? AND status IN ?", now, now.Add(2*time.Hour), []string{"open", "checking"}).
		Count(&dueSoon).Error; err != nil {
		return nil, fmt.Errorf("count upcoming turnarounds: %w", err)
	}
	return map[string]any{"total": total, "statuses": statuses, "risks": risks, "due_within_two_hours": dueSoon}, nil
}
