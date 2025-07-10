package models

import (
	"time"

	"gorm.io/gorm"
)

type Popup struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	Title     string         `json:"title" gorm:"size:255;not null"`
	IsActive  bool           `json:"is_active" gorm:"default:true"`
	Image     string         `json:"image" gorm:"type:text"` // Base64 encoded image or URL
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Popup) TableName() string {
	return "popups"
}
