package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// GPSData represents GPS data from tracking devices
type GPSData struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	IMEI      string    `json:"imei" gorm:"size:16;not null;index" validate:"required,len=16"`
	Timestamp time.Time `json:"timestamp" gorm:"not null;index"`

	// GPS Location Data
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	Speed     *int     `json:"speed"`    // km/h
	Course    *int     `json:"course"`   // degrees (0-360)
	Altitude  *int     `json:"altitude"` // meters

	// GPS Status
	GPSRealTime   *bool `json:"gps_real_time"`
	GPSPositioned *bool `json:"gps_positioned"`
	Satellites    *int  `json:"satellites"`

	// Device Status
	Ignition       string `json:"ignition"`        // ON/OFF
	Charger        string `json:"charger"`         // CONNECTED/DISCONNECTED
	GPSTracking    string `json:"gps_tracking"`    // ENABLED/DISABLED
	OilElectricity string `json:"oil_electricity"` // CONNECTED/DISCONNECTED
	DeviceStatus   string `json:"device_status"`   // ACTIVATED/DEACTIVATED

	// Signal & Power
	VoltageLevel  *int   `json:"voltage_level"`
	VoltageStatus string `json:"voltage_status"`
	GSMSignal     *int   `json:"gsm_signal"`
	GSMStatus     string `json:"gsm_status"`

	// LBS Data
	MCC    *int `json:"mcc"`     // Mobile Country Code
	MNC    *int `json:"mnc"`     // Mobile Network Code
	LAC    *int `json:"lac"`     // Location Area Code
	CellID *int `json:"cell_id"` // Cell ID

	// Alarm Data
	AlarmActive bool   `json:"alarm_active"`
	AlarmType   string `json:"alarm_type"`
	AlarmCode   int    `json:"alarm_code"`

	// Raw Data
	ProtocolName string `json:"protocol_name"`
	RawPacket    string `json:"raw_packet"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships - we'll use manual loading instead of foreign keys to avoid circular references
	Device  Device  `json:"-" gorm:"-"`
	Vehicle Vehicle `json:"-" gorm:"-"`
}

// TableName specifies the table name for GPSData model
func (GPSData) TableName() string {
	return "gps_data"
}

// BeforeCreate hook to set default values
func (g *GPSData) BeforeCreate(tx *gorm.DB) error {
	if g.Timestamp.IsZero() {
		g.Timestamp = time.Now()
	}
	return nil
}

// IsValidLocation checks if GPS coordinates are valid
func (g *GPSData) IsValidLocation() bool {
	// Only check if coordinates are not null
	return g.Latitude != nil && g.Longitude != nil
}

// IsValidForNepal checks if coordinates are within Nepal's boundaries
func (g *GPSData) IsValidForNepal() bool {
	if !g.IsValidLocation() {
		return false
	}

	lat := *g.Latitude
	lng := *g.Longitude

	// Nepal coordinates: Lat: 26.3478° to 30.4465°, Lng: 80.0586° to 88.2014°
	return lat >= 26.0 && lat <= 31.0 && lng >= 79.0 && lng <= 89.0
}

// HasGoodGPSAccuracy checks if GPS has good accuracy
func (g *GPSData) HasGoodGPSAccuracy() bool {
	// Check if GPS is positioned
	if g.GPSPositioned != nil && !*g.GPSPositioned {
		return false
	}

	// Check satellite count
	if g.Satellites != nil && *g.Satellites < 3 {
		return false
	}

	return true
}

// GetLocationString returns a formatted location string
func (g *GPSData) GetLocationString() string {
	if !g.IsValidLocation() {
		return "No valid location"
	}
	return fmt.Sprintf("%.12f,%.12f", *g.Latitude, *g.Longitude)
}
