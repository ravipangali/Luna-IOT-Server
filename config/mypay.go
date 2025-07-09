package config

// MyPayConfig holds configuration for MyPay service
type MyPayConfig struct {
	Token string
	URL   string
}

// GetMyPayConfig returns MyPay configuration from environment variables
func GetMyPayConfig() *MyPayConfig {
	return &MyPayConfig{
		Token: getEnv("MY_PAY_TOKEN", "EMQx29Ap6KmSs2DWD0RiYs8EnrPZfv+Ga0Q2wLG4Ql0="), // Default for development
		URL:   getEnv("MY_PAY_URL", "https://smartdigitalnepal.com/api/service/topup-"),
	}
}
