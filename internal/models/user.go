package models

import (
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
	Password  string         `json:"-" gorm:"size:255;not null" validate:"required,min=6"`
	Role      UserRole       `json:"role" gorm:"type:integer;not null;default:1" validate:"required,oneof=0 1"`
	Image     string         `json:"image" gorm:"size:255"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
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
