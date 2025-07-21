package main

import (
	"luna_iot_server/config"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"
	"time"
)

func main() {
	// Load environment variables
	if err := config.InitializeTimezone(); err != nil {
		colors.PrintError("Failed to initialize timezone: %v", err)
		return
	}

	colors.PrintHeader("GPS FILTERING TEST")
	colors.PrintInfo("Testing GPS filtering logic for ignition and speed conditions...")

	// Test scenarios
	testScenarios := []struct {
		desc     string
		ignition string
		speed    int
		expected string
	}{
		{"Vehicle stopped, ignition OFF", "OFF", 0, "FILTERED - Ignition OFF"},
		{"Vehicle parked, ignition ON, speed 0", "ON", 0, "FILTERED - Speed < 5"},
		{"Vehicle slow, ignition ON, speed 3", "ON", 3, "FILTERED - Speed < 5"},
		{"Vehicle starting, ignition ON, speed 5", "ON", 5, "ACCEPTED - Speed >= 5 and ignition ON"},
		{"Vehicle moving, ignition ON, speed 25", "ON", 25, "ACCEPTED - Speed >= 5 and ignition ON"},
		{"Vehicle fast, ignition ON, speed 80", "ON", 80, "ACCEPTED - Speed >= 5 and ignition ON"},
		{"Vehicle moving, ignition OFF, speed 15", "OFF", 15, "FILTERED - Ignition OFF (regardless of speed)"},
	}

	colors.PrintSubHeader("Testing Filtering Conditions")

	for i, scenario := range testScenarios {
		colors.PrintInfo("--- Test Case %d ---", i+1)
		colors.PrintInfo("Scenario: %s", scenario.desc)
		colors.PrintInfo("Ignition: %s, Speed: %d km/h", scenario.ignition, scenario.speed)
		colors.PrintInfo("Expected: %s", scenario.expected)

		// Apply filtering logic
		shouldFilter := checkShouldFilterLocation(scenario.ignition, scenario.speed)

		if shouldFilter {
			colors.PrintWarning("🚫 RESULT: Location data FILTERED")
			colors.PrintInfo("   - Latitude: nil")
			colors.PrintInfo("   - Longitude: nil")
			colors.PrintInfo("   - Speed: nil")
			colors.PrintInfo("   - Status data: SAVED (ignition, charger, etc.)")
		} else {
			colors.PrintSuccess("✅ RESULT: Location data ACCEPTED")
			colors.PrintInfo("   - Latitude: will be saved")
			colors.PrintInfo("   - Longitude: will be saved")
			colors.PrintInfo("   - Speed: will be saved")
			colors.PrintInfo("   - All data: SAVED")
		}

		colors.PrintInfo("")
		time.Sleep(200 * time.Millisecond)
	}

	colors.PrintSubHeader("Simulation of Filtered GPS Data Structure")

	// Show example of how filtered vs unfiltered data would look
	colors.PrintInfo("Example GPS data structures:")

	colors.PrintInfo("UNFILTERED (ignition ON, speed 25 km/h):")
	unfiltered := models.GPSData{
		IMEI:      "1234567890123456",
		Timestamp: config.GetCurrentTime(),
		Latitude:  floatPtr(27.7172),
		Longitude: floatPtr(85.3240),
		Speed:     intPtr(25),
		Course:    intPtr(45),
		Ignition:  "ON",
		Charger:   "CONNECTED",
	}
	colors.PrintSuccess("   IMEI: %s", unfiltered.IMEI)
	colors.PrintSuccess("   Latitude: %.6f", *unfiltered.Latitude)
	colors.PrintSuccess("   Longitude: %.6f", *unfiltered.Longitude)
	colors.PrintSuccess("   Speed: %d km/h", *unfiltered.Speed)
	colors.PrintSuccess("   Ignition: %s", unfiltered.Ignition)

	colors.PrintInfo("FILTERED (ignition OFF, speed 0 km/h):")
	filtered := models.GPSData{
		IMEI:      "1234567890123456",
		Timestamp: config.GetCurrentTime(),
		Latitude:  nil, // FILTERED OUT
		Longitude: nil, // FILTERED OUT
		Speed:     nil, // FILTERED OUT
		Course:    nil, // FILTERED OUT
		Ignition:  "OFF",
		Charger:   "CONNECTED",
	}
	colors.PrintInfo("   IMEI: %s", filtered.IMEI)
	colors.PrintWarning("   Latitude: nil (filtered)")
	colors.PrintWarning("   Longitude: nil (filtered)")
	colors.PrintWarning("   Speed: nil (filtered)")
	colors.PrintInfo("   Ignition: %s", filtered.Ignition)

	colors.PrintSuccess("✅ GPS filtering test completed!")
	colors.PrintInfo("💡 Benefits of filtering:")
	colors.PrintInfo("   - Reduces database storage for unnecessary location data")
	colors.PrintInfo("   - Prevents false movement tracking when vehicle is stationary")
	colors.PrintInfo("   - Still preserves device status for monitoring")
	colors.PrintInfo("   - WebSocket clients receive status updates without location spam")
}

// checkShouldFilterLocation implements the same logic as in the TCP server
func checkShouldFilterLocation(ignition string, speed int) bool {
	// Filter conditions: ignition OFF or speed < 5
	if ignition == "OFF" {
		return true
	}
	if speed < 5 {
		return true
	}
	return false
}

// Helper functions to create pointers
func floatPtr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}
