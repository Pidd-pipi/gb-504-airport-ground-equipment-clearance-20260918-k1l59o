package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type JSONList []string

func (j JSONList) Value() (driver.Value, error) {
	if j == nil {
		return "[]", nil
	}
	raw, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(raw), nil
}

func (j *JSONList) Scan(value any) error {
	if value == nil {
		*j = JSONList{}
		return nil
	}
	var raw []byte
	switch typed := value.(type) {
	case []byte:
		raw = typed
	case string:
		raw = []byte(typed)
	default:
		return fmt.Errorf("unsupported JSON value %T", value)
	}
	return json.Unmarshal(raw, j)
}

type User struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Phone        string    `gorm:"size:20;uniqueIndex;not null" json:"phone"`
	PasswordHash string    `gorm:"size:100;not null" json:"-"`
	Name         string    `gorm:"size:50;not null" json:"name"`
	Role         string    `gorm:"size:30;not null;default:worker;index" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

func (User) TableName() string { return "users" }

type AuditLog struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	OperatorID   uint64    `gorm:"not null;default:0;index" json:"operator_id"`
	OperatorName string    `gorm:"size:50;not null;default:''" json:"operator_name"`
	Action       string    `gorm:"size:50;not null;index" json:"action"`
	EntityType   string    `gorm:"size:50;not null;index" json:"entity_type"`
	EntityID     string    `gorm:"size:50;not null;default:''" json:"entity_id"`
	Detail       string    `gorm:"type:text" json:"detail"`
	IP           string    `gorm:"size:50;not null;default:''" json:"ip"`
	CreatedAt    time.Time `gorm:"index" json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }
