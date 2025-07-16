package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Notification represents a notification in the system
type Notification struct {
	ID        uint       `json:"id" gorm:"primarykey"`
	Title     string     `json:"title" gorm:"size:255;not null"`
	Body      string     `json:"body" gorm:"type:text;not null"`
	Type      string     `json:"type" gorm:"size:50;default:'system_notification'"`
	ImageURL  string     `json:"image_url" gorm:"type:text"`
	Sound     string     `json:"sound" gorm:"size:50"`
	Priority  string     `json:"priority" gorm:"size:20;default:'normal'"`
	Data      string     `json:"data" gorm:"type:text"` // JSON string for additional data
	IsSent    bool       `json:"is_sent" gorm:"default:false"`
	SentAt    *time.Time `json:"sent_at"`
	CreatedBy uint       `json:"created_by" gorm:"not null;index"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	// Many-to-many relationship with users
	Users []User `json:"users,omitempty" gorm:"many2many:notification_users;foreignKey:ID;joinForeignKey:NotificationID;References:ID;joinReferences:UserID"`

	// Relationship with creator
	Creator User `json:"creator,omitempty" gorm:"foreignKey:CreatedBy;references:ID"`
}

// NotificationUser represents the many-to-many relationship between notifications and users
type NotificationUser struct {
	ID             uint       `json:"id" gorm:"primarykey;autoIncrement"`
	NotificationID uint       `json:"notification_id" gorm:"not null;index"`
	UserID         uint       `json:"user_id" gorm:"not null;index"`
	IsRead         bool       `json:"is_read" gorm:"default:false"`
	ReadAt         *time.Time `json:"read_at"`
	IsSent         bool       `json:"is_sent" gorm:"default:false"`
	SentAt         *time.Time `json:"sent_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Relationships
	Notification Notification `json:"notification,omitempty" gorm:"foreignKey:NotificationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	User         User         `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// TableName specifies the table name for Notification model
func (Notification) TableName() string {
	return "notifications"
}

// TableName specifies the table name for NotificationUser model
func (NotificationUser) TableName() string {
	return "notification_users"
}

// BeforeCreate hook to set default values
func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.Type == "" {
		n.Type = "system_notification"
	}
	if n.Priority == "" {
		n.Priority = "normal"
	}
	return nil
}

// GetDataMap converts the JSON data string to a map
func (n *Notification) GetDataMap() map[string]interface{} {
	if n.Data == "" {
		return make(map[string]interface{})
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(n.Data), &data); err != nil {
		// Return empty map if unmarshaling fails
		return make(map[string]interface{})
	}
	return data
}

// SetDataMap converts a map to JSON string for storage
func (n *Notification) SetDataMap(data map[string]interface{}) {
	if data == nil {
		n.Data = ""
		return
	}

	if dataBytes, err := json.Marshal(data); err == nil {
		n.Data = string(dataBytes)
	} else {
		// Set empty string if marshaling fails
		n.Data = ""
	}
}
