package repository

import (
	"errors"
	"fmt"
	"time"

	"groundclearance/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GroundUnitRepository struct{ db *gorm.DB }

func NewGroundUnitRepository(db *gorm.DB) *GroundUnitRepository {
	return &GroundUnitRepository{db: db}
}

func (r *GroundUnitRepository) LockAssignmentTx(tx *gorm.DB, unitID uint64) error {
	const namespace int64 = 504
	key := (namespace << 32) | int64(unitID&0xffffffff)
	if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", key).Error; err != nil {
		return fmt.Errorf("lock ground unit assignment: %w", err)
	}
	return nil
}

func (r *GroundUnitRepository) Create(unit *model.GroundUnit) error {
	if err := r.db.Create(unit).Error; err != nil {
		return fmt.Errorf("create ground unit: %w", err)
	}
	return nil
}

func (r *GroundUnitRepository) CreateTx(tx *gorm.DB, unit *model.GroundUnit) error {
	if err := tx.Create(unit).Error; err != nil {
		return fmt.Errorf("create ground unit: %w", err)
	}
	return nil
}

func (r *GroundUnitRepository) FindByID(id uint64) (*model.GroundUnit, error) {
	var unit model.GroundUnit
	if err := r.db.First(&unit, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find ground unit: %w", err)
	}
	return &unit, nil
}

func (r *GroundUnitRepository) FindByIDTx(tx *gorm.DB, id uint64) (*model.GroundUnit, error) {
	var unit model.GroundUnit
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&unit, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lock ground unit: %w", err)
	}
	return &unit, nil
}

func (r *GroundUnitRepository) List(page, pageSize int, state, unitType, search string) ([]model.GroundUnit, int64, error) {
	var rows []model.GroundUnit
	var total int64
	query := r.db.Model(&model.GroundUnit{})
	if state != "" {
		query = query.Where("state = ?", state)
	}
	if unitType != "" {
		query = query.Where("unit_type = ?", unitType)
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("unit_code ILIKE ? OR name ILIKE ? OR stand ILIKE ?", like, like, like)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count ground units: %w", err)
	}
	if err := query.Order("unit_code ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list ground units: %w", err)
	}
	return rows, total, nil
}

func (r *GroundUnitRepository) UpdateTx(tx *gorm.DB, unit *model.GroundUnit) error {
	now := time.Now()
	result := tx.Model(&model.GroundUnit{}).Where("id = ? AND version = ?", unit.ID, unit.Version).
		Updates(map[string]any{"state": unit.State, "notes": unit.Notes, "version": unit.Version + 1, "updated_at": now})
	if result.Error != nil {
		return fmt.Errorf("update ground unit: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrConflict
	}
	unit.Version++
	unit.UpdatedAt = now
	return nil
}

// Summary returns stable counters for the equipment status header. Zero-value
// states are included so clients do not need to special-case missing groups.
func (r *GroundUnitRepository) Summary() (map[string]any, error) {
	type countRow struct {
		Key   string
		Count int64
	}
	var stateRows []countRow
	if err := r.db.Model(&model.GroundUnit{}).
		Select("state AS key, COUNT(*) AS count").Group("state").Scan(&stateRows).Error; err != nil {
		return nil, fmt.Errorf("group ground units by state: %w", err)
	}
	states := map[string]int64{"available": 0, "inspection": 0, "blocked": 0, "retired": 0}
	var total int64
	for _, row := range stateRows {
		states[row.Key] = row.Count
		total += row.Count
	}
	var typeRows []countRow
	if err := r.db.Model(&model.GroundUnit{}).
		Select("unit_type AS key, COUNT(*) AS count").Group("unit_type").Order("count DESC").Scan(&typeRows).Error; err != nil {
		return nil, fmt.Errorf("group ground units by type: %w", err)
	}
	types := make(map[string]int64, len(typeRows))
	for _, row := range typeRows {
		types[row.Key] = row.Count
	}
	return map[string]any{
		"total": total, "states": states, "types": types,
		"dispatchable": states["available"], "unavailable": states["inspection"] + states["blocked"] + states["retired"],
	}, nil
}
