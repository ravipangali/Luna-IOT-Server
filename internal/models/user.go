package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserRole represents the user role enum
type UserRole int

const (
	UserRoleAdmin  UserRole = 0 // Admin role
	UserRoleClient UserRole = 1 // Client role
)

// User represents a system user
type User struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	Name      string         `json:"name" gorm:"size:100;not null" validate:"required,min=2,max=100"`
	Phone     string         `json:"phone" gorm:"size:15;uniqueIndex" validate:"required,min=10,max=15"`
	Email     string         `json:"email" gorm:"size:100;uniqueIndex" validate:"required,email"`
	Password  string         `json:"password" gorm:"size:255;not null" validate:"required,min=6"`
	Role      UserRole       `json:"role" gorm:"type:integer;not null;default:1" validate:"required,oneof=0 1"`
	Image     string         `json:"image" gorm:"type:text"`
	Token     string         `json:"-" gorm:"size:255;uniqueIndex"` // Authentication token (hidden from JSON)
	TokenExp  *time.Time     `json:"-" gorm:"index"`                // Token expiration time
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships - many-to-many with vehicles through UserVehicle
	VehicleAccess []UserVehicle `json:"vehicle_access,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Vehicles      []Vehicle     `json:"vehicles,omitempty" gorm:"many2many:user_vehicles;foreignKey:ID;joinForeignKey:UserID;References:IMEI;joinReferences:VehicleID"`
}

// TableName specifies the table name for User model
func (User) TableName() string {
	return "users"
}

// BeforeCreate hook to hash password before saving
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(hashedPassword)
	}
	return nil
}

// BeforeUpdate hook to hash password before updating
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	if tx.Statement.Changed("Password") && u.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(hashedPassword)
	}
	return nil
}

// CheckPassword verifies if the provided password matches the user's password
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// GenerateToken creates a new authentication token for the user
func (u *User) GenerateToken() error {
	// Generate a random 32-byte token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}

	u.Token = hex.EncodeToString(tokenBytes)
	// Set token expiration to 24 hours from now
	expirationTime := time.Now().Add(24 * time.Hour)
	u.TokenExp = &expirationTime

	return nil
}

// IsTokenValid checks if the user's token is still valid
func (u *User) IsTokenValid() bool {
	if u.Token == "" || u.TokenExp == nil {
		return false
	}
	return time.Now().Before(*u.TokenExp)
}

// ClearToken removes the authentication token
func (u *User) ClearToken() {
	u.Token = ""
	u.TokenExp = nil
}

// GetRoleString returns the string representation of the user role
func (u *User) GetRoleString() string {
	switch u.Role {
	case UserRoleAdmin:
		return "admin"
	case UserRoleClient:
		return "client"
	default:
		return "unknown"
	}
}

// ToSafeUser returns user data without sensitive information
func (u *User) ToSafeUser() map[string]interface{} {
	return map[string]interface{}{
		"id":             u.ID,
		"name":           u.Name,
		"phone":          u.Phone,
		"email":          u.Email,
		"role":           u.Role,
		"role_name":      u.GetRoleString(),
		"image":          u.Image,
		"vehicle_access": u.VehicleAccess,
		"vehicles":       u.Vehicles,
		"created_at":     u.CreatedAt,
		"updated_at":     u.UpdatedAt,
	}
}

// HasVehiclePermission checks if user has a specific permission for a vehicle
func (u *User) HasVehiclePermission(vehicleID string, permission Permission) bool {
	// Admin users have all permissions
	if u.Role == UserRoleAdmin {
		return true
	}

	for _, access := range u.VehicleAccess {
		if access.VehicleID == vehicleID {
			return access.HasPermission(permission)
		}
	}
	return false
}

// GetVehiclePermissions returns all permissions for a specific vehicle
func (u *User) GetVehiclePermissions(vehicleID string) []Permission {
	// Admin users have all permissions
	if u.Role == UserRoleAdmin {
		return []Permission{
			PermissionAllAccess,
			PermissionLiveTracking,
			PermissionHistory,
			PermissionReport,
			PermissionVehicleEdit,
			PermissionNotification,
			PermissionShareTracking,
		}
	}

	for _, access := range u.VehicleAccess {
		if access.VehicleID == vehicleID {
			return access.GetPermissions()
		}
	}
	return []Permission{}
}

// GetAccessibleVehicles returns all vehicles the user has access to
func (u *User) GetAccessibleVehicles() []string {
	var vehicleIDs []string
	for _, access := range u.VehicleAccess {
		if access.IsActive && !access.IsExpired() {
			vehicleIDs = append(vehicleIDs, access.VehicleID)
		}
	}
	return vehicleIDs
}
