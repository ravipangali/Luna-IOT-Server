package models

import (
	"time"

	"gorm.io/gorm"
)

// DeviceModel represents a device model/type
type DeviceModel struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	Name      string    `json:"name" gorm:"size:100;not null;uniqueIndex" validate:"required,min=2,max=100"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Devices []Device `json:"devices,omitempty" gorm:"foreignKey:ModelID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

// TableName returns the table name for DeviceModel
func (DeviceModel) TableName() string {
	return "device_models"
}

// BeforeCreate hook to validate data before creation
func (dm *DeviceModel) BeforeCreate(tx *gorm.DB) error {
	// Validation logic can be added here if needed
	return nil
}
