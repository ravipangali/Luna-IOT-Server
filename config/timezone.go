package config

import (
	"time"
)

// TimezoneConfig holds timezone configuration
type TimezoneConfig struct {
	Location *time.Location
}

var (
	// Default timezone for Kathmandu, Nepal
	KathmanduLocation *time.Location
	// Global timezone configuration
	AppTimezone *TimezoneConfig
)

// InitializeTimezone sets up the application timezone
func InitializeTimezone() error {
	// Try to get timezone from environment variable, default to Kathmandu
	tzName := getEnv("APP_TIMEZONE", "Asia/Kathmandu")

	location, err := time.LoadLocation(tzName)
	if err != nil {
		// Fallback to Kathmandu if the specified timezone is invalid
		location, err = time.LoadLocation("Asia/Kathmandu")
		if err != nil {
			return err
		}
	}

	KathmanduLocation = location
	AppTimezone = &TimezoneConfig{Location: location}

	return nil
}

// GetCurrentTime returns current time in Kathmandu timezone
func GetCurrentTime() time.Time {
	if AppTimezone != nil && AppTimezone.Location != nil {
		return time.Now().In(AppTimezone.Location)
	}
	// Fallback to Kathmandu timezone
	return time.Now().In(KathmanduLocation)
}

// ParseTimeInTimezone parses a time string in the application timezone
func ParseTimeInTimezone(timeStr, layout string) (time.Time, error) {
	if AppTimezone != nil && AppTimezone.Location != nil {
		return time.ParseInLocation(layout, timeStr, AppTimezone.Location)
	}
	return time.ParseInLocation(layout, timeStr, KathmanduLocation)
}

// FormatTimeInTimezone formats a time in the application timezone
func FormatTimeInTimezone(t time.Time, layout string) string {
	if AppTimezone != nil && AppTimezone.Location != nil {
		return t.In(AppTimezone.Location).Format(layout)
	}
	return t.In(KathmanduLocation).Format(layout)
}

// GetTimezoneOffset returns the timezone offset in hours
func GetTimezoneOffset() int {
	if AppTimezone != nil && AppTimezone.Location != nil {
		_, offset := time.Now().In(AppTimezone.Location).Zone()
		return offset / 3600
	}
	// Kathmandu is UTC+5:45
	return 5
}

// GetTimezoneString returns the timezone string
func GetTimezoneString() string {
	if AppTimezone != nil && AppTimezone.Location != nil {
		return AppTimezone.Location.String()
	}
	return "Asia/Kathmandu"
}
