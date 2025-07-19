package main

import (
	"luna_iot_server/config"
	"luna_iot_server/pkg/colors"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		colors.PrintWarning("No .env file found, using system environment variables")
	}

	// Initialize timezone
	if err := config.InitializeTimezone(); err != nil {
		colors.PrintError("Failed to initialize timezone: %v", err)
		return
	}

	// Test timezone functionality
	colors.PrintHeader("TIMEZONE TEST")
	colors.PrintInfo("Server timezone: %s", config.GetTimezoneString())
	colors.PrintInfo("Timezone offset: UTC+%d", config.GetTimezoneOffset())

	// Test current time
	currentTime := config.GetCurrentTime()
	colors.PrintInfo("Current time in Kathmandu: %s", currentTime.Format("2006-01-02 15:04:05 MST"))

	// Test time parsing
	testTimeStr := "2025-07-19 18:30:00"
	parsedTime, err := config.ParseTimeInTimezone(testTimeStr, "2006-01-02 15:04:05")
	if err != nil {
		colors.PrintError("Failed to parse time: %v", err)
	} else {
		colors.PrintInfo("Parsed time: %s", parsedTime.Format("2006-01-02 15:04:05 MST"))
	}

	// Test time formatting
	formattedTime := config.FormatTimeInTimezone(currentTime, "2006-01-02 15:04:05")
	colors.PrintInfo("Formatted time: %s", formattedTime)

	// Compare with system time
	systemTime := time.Now()
	colors.PrintInfo("System time: %s", systemTime.Format("2006-01-02 15:04:05 MST"))
	colors.PrintInfo("Kathmandu time: %s", currentTime.Format("2006-01-02 15:04:05 MST"))

	colors.PrintSuccess("✅ Timezone test completed successfully!")
}
