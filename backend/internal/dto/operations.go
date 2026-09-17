package dto

import "time"

type GroundUnitCreateRequest struct {
	UnitCode         string     `json:"unit_code" binding:"required,max=40"`
	Name             string     `json:"name" binding:"required,max=120"`
	UnitType         string     `json:"unit_type" binding:"required,max=40"`
	Stand            string     `json:"stand" binding:"required,max=30"`
	State            string     `json:"state" binding:"required,oneof=available inspection blocked retired"`
	LastInspectionAt *time.Time `json:"last_inspection_at"`
	Notes            string     `json:"notes" binding:"max=2000"`
}

type GroundUnitStateRequest struct {
	State   string `json:"state" binding:"required,oneof=available inspection blocked retired"`
	Notes   string `json:"notes" binding:"max=2000"`
	Version int    `json:"version" binding:"required,min=1"`
}

type ClearanceDecisionRequest struct {
	TurnaroundID uint64   `json:"turnaround_id" binding:"required"`
	State        string   `json:"state" binding:"required,oneof=cleared restricted revoked"`
	Restrictions string   `json:"restrictions" binding:"max=2000"`
	Reason       string   `json:"reason" binding:"required,max=2000"`
	Evidence     []string `json:"evidence" binding:"required,min=1"`
}
