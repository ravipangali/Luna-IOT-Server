package main

import (
	"luna_iot_server/config"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"
	"math"
)

func main() {
	// Load environment variables
	if err := config.InitializeTimezone(); err != nil {
		colors.PrintError("Failed to initialize timezone: %v", err)
		return
	}

	colors.PrintHeader("GPS COORDINATE TESTING")
	colors.PrintInfo("Testing GPS coordinate validation and smoothing...")

	// Test coordinates for Nepal region
	testCoordinates := []struct {
		lat, lng float64
		desc     string
	}{
		{27.7172, 85.3240, "Kathmandu - Valid"},
		{26.3478, 80.0586, "Southern Nepal - Valid"},
		{30.4465, 88.2014, "Northern Nepal - Valid"},
		{25.0, 85.0, "Outside Nepal (South) - Invalid"},
		{32.0, 85.0, "Outside Nepal (North) - Invalid"},
		{27.0, 78.0, "Outside Nepal (West) - Invalid"},
		{27.0, 90.0, "Outside Nepal (East) - Invalid"},
	}

	for _, coord := range testCoordinates {
		gpsData := models.GPSData{
			Latitude:  &coord.lat,
			Longitude: &coord.lng,
		}

		isValid := gpsData.IsValidForNepal()
		status := "VALID"
		if !isValid {
			status = "INVALID"
		}

		colors.PrintInfo("%s: Lat=%.4f, Lng=%.4f - %s", coord.desc, coord.lat, coord.lng, status)
	}

	// Test GPS smoothing
	colors.PrintSubHeader("GPS Smoothing Test")

	// Simulate zigzag coordinates
	originalCoords := []struct {
		lat, lng float64
	}{
		{27.7172, 85.3240}, // Kathmandu
		{27.7175, 85.3245}, // Slight zigzag
		{27.7170, 85.3235}, // Back
		{27.7178, 85.3250}, // Forward
	}

	colors.PrintInfo("Original coordinates (simulated zigzag):")
	for i, coord := range originalCoords {
		colors.PrintInfo("  Point %d: %.6f, %.6f", i+1, coord.lat, coord.lng)
	}

	// Simulate smoothing
	colors.PrintInfo("Smoothed coordinates (70%% new + 30%% previous):")
	prevLat, prevLng := originalCoords[0].lat, originalCoords[0].lng
	colors.PrintInfo("  Point 1: %.6f, %.6f (original)", prevLat, prevLng)

	for i := 1; i < len(originalCoords); i++ {
		currLat, currLng := originalCoords[i].lat, originalCoords[i].lng
		smoothedLat := 0.7*currLat + 0.3*prevLat
		smoothedLng := 0.7*currLng + 0.3*prevLng

		colors.PrintInfo("  Point %d: %.6f, %.6f -> %.6f, %.6f",
			i+1, currLat, currLng, smoothedLat, smoothedLng)

		prevLat, prevLng = smoothedLat, smoothedLng
	}

	// Test distance calculation
	colors.PrintSubHeader("Distance Calculation Test")

	// Kathmandu to Pokhara (approximate)
	lat1, lng1 := 27.7172, 85.3240 // Kathmandu
	lat2, lng2 := 28.2096, 83.9856 // Pokhara

	distance := calculateDistance(lat1, lng1, lat2, lng2)
	colors.PrintInfo("Distance Kathmandu to Pokhara: %.2f km", distance)

	// Test erratic GPS detection
	colors.PrintSubHeader("Erratic GPS Detection Test")

	// Normal movement
	normalCoords := []struct {
		lat, lng float64
		desc     string
	}{
		{27.7172, 85.3240, "Start"},
		{27.7175, 85.3245, "Normal movement"},
		{27.7180, 85.3250, "Normal movement"},
	}

	// Erratic movement
	erraticCoords := []struct {
		lat, lng float64
		desc     string
	}{
		{27.7172, 85.3240, "Start"},
		{27.7175, 85.3245, "Normal movement"},
		{28.5000, 86.0000, "Erratic jump (>1km)"},
	}

	colors.PrintInfo("Normal movement test:")
	for i, coord := range normalCoords {
		if i > 0 {
			dist := calculateDistance(normalCoords[i-1].lat, normalCoords[i-1].lng, coord.lat, coord.lng)
			status := "NORMAL"
			if dist > 1.0 {
				status = "ERRATIC"
			}
			colors.PrintInfo("  %s: Distance=%.3f km - %s", coord.desc, dist, status)
		} else {
			colors.PrintInfo("  %s: Starting point", coord.desc)
		}
	}

	colors.PrintInfo("Erratic movement test:")
	for i, coord := range erraticCoords {
		if i > 0 {
			dist := calculateDistance(erraticCoords[i-1].lat, erraticCoords[i-1].lng, coord.lat, coord.lng)
			status := "NORMAL"
			if dist > 1.0 {
				status = "ERRATIC"
			}
			colors.PrintInfo("  %s: Distance=%.3f km - %s", coord.desc, dist, status)
		} else {
			colors.PrintInfo("  %s: Starting point", coord.desc)
		}
	}

	colors.PrintSuccess("GPS coordinate testing completed!")
}

// calculateDistance calculates the distance between two coordinates using Haversine formula
func calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371 // Earth's radius in kilometers

	// Convert to radians
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180

	// Haversine formula
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
