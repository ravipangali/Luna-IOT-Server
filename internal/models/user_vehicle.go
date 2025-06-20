package models

import (
	"time"

	"gorm.io/gorm"
)

// Permission represents different access levels
type Permission string

const (
	PermissionAllAccess     Permission = "all_access"
	PermissionLiveTracking  Permission = "live_tracking"
	PermissionHistory       Permission = "history"
	PermissionReport        Permission = "report"
	PermissionVehicleEdit   Permission = "vehicle_edit"
	PermissionNotification  Permission = "notification"
	PermissionShareTracking Permission = "share_tracking"
)

// UserVehicle represents the many-to-many relationship between users and vehicles with permissions
type UserVehicle struct {
	ID        uint   `json:"id" gorm:"primarykey"`
	UserID    uint   `json:"user_id" gorm:"not null;index"`
	VehicleID string `json:"vehicle_id" gorm:"not null;size:16;index"` // IMEI

	// Permission flags - each can be individually granted
	AllAccess     bool `json:"all_access" gorm:"default:false"`
	LiveTracking  bool `json:"live_tracking" gorm:"default:false"`
	History       bool `json:"history" gorm:"default:false"`
	Report        bool `json:"report" gorm:"default:false"`
	VehicleEdit   bool `json:"vehicle_edit" gorm:"default:false"`
	Notification  bool `json:"notification" gorm:"default:false"`
	ShareTracking bool `json:"share_tracking" gorm:"default:false"`

	// Main user flag - indicates if this user is the primary owner of the vehicle
	IsMainUser bool `json:"is_main_user" gorm:"default:false"`

	// Additional metadata
	GrantedBy uint       `json:"granted_by" gorm:"index"` // User ID who granted the access
	GrantedAt time.Time  `json:"granted_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"` // Optional expiration
	IsActive  bool       `json:"is_active" gorm:"default:true"`
	Notes     string     `json:"notes" gorm:"type:text"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	User          User    `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Vehicle       Vehicle `json:"vehicle,omitempty" gorm:"foreignKey:VehicleID;references:IMEI;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	GrantedByUser User    `json:"granted_by_user,omitempty" gorm:"foreignKey:GrantedBy;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}

// TableName specifies the table name for UserVehicle model
func (UserVehicle) TableName() string {
	return "user_vehicles"
}

// BeforeCreate hook to set default values
func (uv *UserVehicle) BeforeCreate(tx *gorm.DB) error {
	if uv.GrantedAt.IsZero() {
		uv.GrantedAt = time.Now()
	}
	return nil
}

// HasPermission checks if user has a specific permission for this vehicle
func (uv *UserVehicle) HasPermission(permission Permission) bool {
	if !uv.IsActive {
		return false
	}

	// Check if access has expired
	if uv.ExpiresAt != nil && time.Now().After(*uv.ExpiresAt) {
		return false
	}

	// All access grants everything
	if uv.AllAccess {
		return true
	}

	switch permission {
	case PermissionLiveTracking:
		return uv.LiveTracking
	case PermissionHistory:
		return uv.History
	case PermissionReport:
		return uv.Report
	case PermissionVehicleEdit:
		return uv.VehicleEdit
	case PermissionNotification:
		return uv.Notification
	case PermissionShareTracking:
		return uv.ShareTracking
	case PermissionAllAccess:
		return uv.AllAccess
	default:
		return false
	}
}

// GetPermissions returns a list of all granted permissions
func (uv *UserVehicle) GetPermissions() []Permission {
	var permissions []Permission

	if !uv.IsActive {
		return permissions
	}

	if uv.ExpiresAt != nil && time.Now().After(*uv.ExpiresAt) {
		return permissions
	}

	if uv.AllAccess {
		permissions = append(permissions, PermissionAllAccess)
		return permissions
	}

	if uv.LiveTracking {
		permissions = append(permissions, PermissionLiveTracking)
	}
	if uv.History {
		permissions = append(permissions, PermissionHistory)
	}
	if uv.Report {
		permissions = append(permissions, PermissionReport)
	}
	if uv.VehicleEdit {
		permissions = append(permissions, PermissionVehicleEdit)
	}
	if uv.Notification {
		permissions = append(permissions, PermissionNotification)
	}
	if uv.ShareTracking {
		permissions = append(permissions, PermissionShareTracking)
	}

	return permissions
}

// IsExpired checks if the access has expired
func (uv *UserVehicle) IsExpired() bool {
	if uv.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*uv.ExpiresAt)
}

// GrantPermission grants a specific permission
func (uv *UserVehicle) GrantPermission(permission Permission) {
	switch permission {
	case PermissionAllAccess:
		uv.AllAccess = true
	case PermissionLiveTracking:
		uv.LiveTracking = true
	case PermissionHistory:
		uv.History = true
	case PermissionReport:
		uv.Report = true
	case PermissionVehicleEdit:
		uv.VehicleEdit = true
	case PermissionNotification:
		uv.Notification = true
	case PermissionShareTracking:
		uv.ShareTracking = true
	}
}

// RevokePermission revokes a specific permission
func (uv *UserVehicle) RevokePermission(permission Permission) {
	switch permission {
	case PermissionAllAccess:
		uv.AllAccess = false
	case PermissionLiveTracking:
		uv.LiveTracking = false
	case PermissionHistory:
		uv.History = false
	case PermissionReport:
		uv.Report = false
	case PermissionVehicleEdit:
		uv.VehicleEdit = false
	case PermissionNotification:
		uv.Notification = false
	case PermissionShareTracking:
		uv.ShareTracking = false
	}
}

// IsMainUserOfVehicle checks if this user is the main owner of the vehicle
func (uv *UserVehicle) IsMainUserOfVehicle() bool {
	return uv.IsMainUser
}

// CreateMainUserAssignment creates a UserVehicle relationship for the main user (owner)
func CreateMainUserAssignment(userID uint, vehicleID string, grantedBy uint) *UserVehicle {
	return &UserVehicle{
		UserID:        userID,
		VehicleID:     vehicleID,
		AllAccess:     true, // Main user gets all access
		LiveTracking:  true,
		History:       true,
		Report:        true,
		VehicleEdit:   true,
		Notification:  true,
		ShareTracking: true,
		IsMainUser:    true, // Mark as main user
		GrantedBy:     grantedBy,
		GrantedAt:     time.Now(),
		IsActive:      true,
		Notes:         "Main user (Vehicle Owner)",
	}
}

// GetUserRole returns the role of the user for this vehicle (main or shared)
func (uv *UserVehicle) GetUserRole() string {
	if uv.IsMainUser {
		return "main"
	}
	return "shared"
}
