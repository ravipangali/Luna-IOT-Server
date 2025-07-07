package models

import (
	"time"

	"gorm.io/gorm"
)

// VehicleType represents the vehicle type enum
type VehicleType string

const (
	VehicleTypeBike      VehicleType = "bike"
	VehicleTypeCar       VehicleType = "car"
	VehicleTypeTruck     VehicleType = "truck"
	VehicleTypeBus       VehicleType = "bus"
	VehicleTypeSchoolBus VehicleType = "school_bus"
)

// Vehicle represents a vehicle in the tracking system
type Vehicle struct {
	IMEI        string         `json:"imei" gorm:"primaryKey;size:16;not null" validate:"required,len=16"`
	RegNo       string         `json:"reg_no" gorm:"size:20;uniqueIndex;not null" validate:"required"`
	Name        string         `json:"name" gorm:"size:100;not null" validate:"required"`
	Odometer    float64        `json:"odometer" gorm:"type:decimal(10,2);default:0"`
	Mileage     float64        `json:"mileage" gorm:"type:decimal(5,2)"`
	MinFuel     float64        `json:"min_fuel" gorm:"type:decimal(5,2)"`
	Overspeed   int            `json:"overspeed" gorm:"type:integer;default:60"`
	VehicleType VehicleType    `json:"vehicle_type" gorm:"type:varchar(20);not null" validate:"required,oneof=bike car truck bus school_bus"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationship - Reference device by IMEI but no foreign key constraint
	// This allows devices to be created independently
	Device Device `json:"device,omitempty" gorm:"foreignKey:IMEI;references:IMEI"`

	// User relationships - many-to-many with users through UserVehicle
	UserAccess []UserVehicle `json:"user_access,omitempty" gorm:"foreignKey:VehicleID;references:IMEI"`
	Users      []User        `json:"-" gorm:"many2many:user_vehicles;foreignKey:IMEI;joinForeignKey:VehicleID;References:ID;joinReferences:UserID"`
}

// TableName specifies the table name for Vehicle model
func (Vehicle) TableName() string {
	return "vehicles"
}

// BeforeCreate hook to validate vehicle before creation
func (v *Vehicle) BeforeCreate(tx *gorm.DB) error {
	// Additional validation can be added here
	if v.Overspeed <= 0 {
		v.Overspeed = 60 // Default overspeed limit
	}
	return nil
}

// LoadDevice manually loads the associated device for this vehicle
func (v *Vehicle) LoadDevice(db *gorm.DB) error {
	if v.IMEI == "" {
		return nil // No IMEI, no device to load
	}

	var device Device
	if err := db.Where("imei = ?", v.IMEI).First(&device).Error; err != nil {
		return err // Device not found or database error
	}

	v.Device = device
	return nil
}

// GetAuthorizedUsers returns all users who have access to this vehicle
func (v *Vehicle) GetAuthorizedUsers() []uint {
	var userIDs []uint
	for _, access := range v.UserAccess {
		if access.IsActive && !access.IsExpired() {
			userIDs = append(userIDs, access.UserID)
		}
	}
	return userIDs
}

// HasUserAccess checks if a user has any access to this vehicle
func (v *Vehicle) HasUserAccess(userID uint) bool {
	for _, access := range v.UserAccess {
		if access.UserID == userID && access.IsActive && !access.IsExpired() {
			return true
		}
	}
	return false
}

// GetUserPermissions returns the permissions a specific user has for this vehicle
func (v *Vehicle) GetUserPermissions(userID uint) []Permission {
	for _, access := range v.UserAccess {
		if access.UserID == userID && access.IsActive && !access.IsExpired() {
			return access.GetPermissions()
		}
	}
	return []Permission{}
}
