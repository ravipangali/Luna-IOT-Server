package models

import (
	"time"

	"gorm.io/gorm"
)

// SimOperator represents the SIM operator enum
type SimOperator string

const (
	SimOperatorNcell SimOperator = "Ncell"
	SimOperatorNtc   SimOperator = "Ntc"
)

// Protocol represents the device protocol enum
type Protocol string

const (
	ProtocolGT06 Protocol = "GT06"
)

// Device represents a GPS tracking device
type Device struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	IMEI        string         `json:"imei" gorm:"uniqueIndex;not null;size:15" validate:"required,len=15"`
	SimNo       string         `json:"sim_no" gorm:"size:20" validate:"required"`
	SimOperator SimOperator    `json:"sim_operator" gorm:"type:varchar(10);not null" validate:"required,oneof=Ncell Ntc"`
	Protocol    Protocol       `json:"protocol" gorm:"type:varchar(10);not null;default:'GT06'" validate:"required"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships (has many vehicles)
	// Note: This creates a reverse relationship without foreign key constraints
}

// TableName specifies the table name for Device model
func (Device) TableName() string {
	return "devices"
}

// BeforeCreate hook to validate device before creation
func (d *Device) BeforeCreate(tx *gorm.DB) error {
	// Additional validation can be added here
	return nil
}
