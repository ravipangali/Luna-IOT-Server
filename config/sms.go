package config

import (
	"os"
)

// SMSConfig holds the configuration for the SMS service
type SMSConfig struct {
	APIKey     string
	APIURL     string
	CampaignID string
	RouteID    string
	SenderID   string
}

// GetSMSConfig returns SMS configuration from environment variables
func GetSMSConfig() *SMSConfig {
	return &SMSConfig{
		APIKey:     getEnv("SMS_API_KEY", "568383D0C5AA82"),
		APIURL:     getEnv("SMS_API_URL", "https://sms.kaichogroup.com/smsapi/index.php"),
		CampaignID: getEnv("SMS_CAMPAIGN_ID", "9148"),
		RouteID:    getEnv("SMS_ROUTE_ID", "130"),
		SenderID:   getEnv("SMS_SENDER_ID", "SMSBit"),
	}
}

// Helper function to get environment variables with a fallback
// This is duplicated from database.go but is kept here for modularity.
// In a larger application, this could be moved to a shared util package.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
