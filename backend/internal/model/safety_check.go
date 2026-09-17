package model

import "time"

// SafetyCheck records an operational check and its evidence.
type SafetyCheck struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TurnaroundID uint64     `gorm:"not null;index;uniqueIndex:ux_safety_check_code,priority:1;uniqueIndex:ux_safety_check_sequence,priority:1" json:"turnaround_id"`
	GroundUnitID *uint64    `gorm:"index" json:"ground_unit_id"`
	Sequence     int        `gorm:"not null;default:1;uniqueIndex:ux_safety_check_sequence,priority:2" json:"sequence"`
	CheckCode    string     `gorm:"size:40;not null;uniqueIndex:ux_safety_check_code,priority:2" json:"check_code"`
	ItemName     string     `gorm:"size:200;not null" json:"item_name"`
	RiskLevel    string     `gorm:"size:20;not null;default:medium;check:chk_safety_checks_risk,risk_level IN ('low','medium','high','critical')" json:"risk_level"`
	Result       string     `gorm:"size:20;not null;default:pending;index;check:chk_safety_checks_result,result IN ('pending','passed','failed')" json:"result"`
	Evidence     JSONList   `gorm:"type:jsonb" json:"evidence"`
	Remark       string     `gorm:"type:text" json:"remark"`
	CheckedBy    uint64     `gorm:"index" json:"checked_by"`
	CheckedAt    *time.Time `json:"checked_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (SafetyCheck) TableName() string { return "safety_checks" }
