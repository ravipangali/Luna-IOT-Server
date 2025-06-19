package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// GPSData represents GPS tracking data from devices
type GPSData struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	IMEI      string    `json:"imei" gorm:"size:16;not null;index" validate:"required,len=16"`
	Timestamp time.Time `json:"timestamp" gorm:"not null;index"`

	// GPS Location Data - Enhanced precision for accurate tracking
	Latitude  *float64 `json:"latitude" gorm:"type:decimal(15,12)"`
	Longitude *float64 `json:"longitude" gorm:"type:decimal(15,12)"`
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

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	Device  Device  `json:"device,omitempty" gorm:"foreignKey:IMEI;references:IMEI;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Vehicle Vehicle `json:"vehicle,omitempty" gorm:"foreignKey:IMEI;references:IMEI;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
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
	// Latitude: Only positive values (0-90)
	// Longitude: Both negative and positive values (-180 to +180)
	return g.Latitude != nil && g.Longitude != nil &&
		*g.Latitude > 0 && *g.Latitude <= 90 &&
		*g.Longitude >= -180 && *g.Longitude <= 180
}

// GetLocationString returns a formatted location string
func (g *GPSData) GetLocationString() string {
	if !g.IsValidLocation() {
		return "No valid location"
	}
	return fmt.Sprintf("%.12f,%.12f", *g.Latitude, *g.Longitude)
}
