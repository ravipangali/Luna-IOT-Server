package config

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
