package models

import (
	"luna_iot_server/pkg/colors"
	"time"

	"gorm.io/gorm"
)

// Setting holds global application settings.
// This table is designed to have only one row.
type Setting struct {
	ID           uint    `json:"id" gorm:"primarykey"`
	MyPayBalance float64 `json:"my_pay_balance" gorm:"type:decimal(10,2);default:0"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Setting) TableName() string {
	return "settings"
}

// EnsureSettingExists checks if a setting record exists, and creates one if not.
// This should be called on application startup.
func EnsureSettingExists(db *gorm.DB) {
	var count int64
	db.Model(&Setting{}).Count(&count)
	if count == 0 {
		colors.PrintInfo("No settings record found, creating default settings...")
		setting := Setting{ID: 1, MyPayBalance: 0}
		if err := db.Create(&setting).Error; err != nil {
			colors.PrintError("Failed to create default settings: %v", err)
		} else {
			colors.PrintSuccess("Default settings created successfully.")
		}
	}
}
