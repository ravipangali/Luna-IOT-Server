package controllers

import (
	"net/http"
	"strconv"
	"time"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"

	"github.com/gin-gonic/gin"
)

// GPSController handles GPS data related HTTP requests
type GPSController struct{}

// NewGPSController creates a new GPS controller
func NewGPSController() *GPSController {
	return &GPSController{}
}

// GetGPSData returns GPS data with optional filtering
func (gc *GPSController) GetGPSData(c *gin.Context) {
	var gpsData []models.GPSData
	query := db.GetDB().Preload("Device").Preload("Vehicle")

	// Optional filters
	if imei := c.Query("imei"); imei != "" {
		query = query.Where("imei = ?", imei)
	}

	if from := c.Query("from"); from != "" {
		if fromTime, err := time.Parse("2006-01-02T15:04:05Z", from); err == nil {
			query = query.Where("timestamp >= ?", fromTime)
		}
	}

	if to := c.Query("to"); to != "" {
		if toTime, err := time.Parse("2006-01-02T15:04:05Z", to); err == nil {
			query = query.Where("timestamp <= ?", toTime)
		}
	}

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset := (page - 1) * limit

	if err := query.Order("timestamp DESC").Limit(limit).Offset(offset).Find(&gpsData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch GPS data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gpsData,
		"count":   len(gpsData),
		"page":    page,
		"limit":   limit,
		"message": "GPS data retrieved successfully",
	})
}

// GetGPSDataByIMEI returns GPS data for a specific device
func (gc *GPSController) GetGPSDataByIMEI(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid IMEI format",
		})
		return
	}

	var gpsData []models.GPSData
	query := db.GetDB().Where("imei = ?", imei).Preload("Device").Preload("Vehicle")

	// Time range filtering
	if from := c.Query("from"); from != "" {
		if fromTime, err := time.Parse("2006-01-02T15:04:05Z", from); err == nil {
			query = query.Where("timestamp >= ?", fromTime)
		}
	}

	if to := c.Query("to"); to != "" {
		if toTime, err := time.Parse("2006-01-02T15:04:05Z", to); err == nil {
			query = query.Where("timestamp <= ?", toTime)
		}
	}

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset := (page - 1) * limit

	if err := query.Order("timestamp DESC").Limit(limit).Offset(offset).Find(&gpsData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch GPS data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gpsData,
		"count":   len(gpsData),
		"page":    page,
		"limit":   limit,
		"message": "GPS data retrieved successfully",
	})
}

// GetLatestGPSData returns the latest GPS data for each device with location fallback
func (gc *GPSController) GetLatestGPSData(c *gin.Context) {
	var gpsData []models.GPSData

	// Get latest GPS data for each IMEI regardless of device connection
	if err := db.GetDB().Raw(`
		SELECT DISTINCT ON (imei) *
		FROM gps_data
		WHERE deleted_at IS NULL
		ORDER BY imei, timestamp DESC
	`).Preload("Device").Preload("Vehicle").Scan(&gpsData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch latest GPS data",
		})
		return
	}

	// CRITICAL CHANGE: Do NOT use coordinate fallback
	// If latest GPS data has null coordinates, keep them as null
	// This ensures frontend knows when to show empty map vs markers
	for _, data := range gpsData {
		// Only log when coordinates are null - don't modify them
		if data.Latitude == nil || data.Longitude == nil {
			colors.PrintInfo("📍 IMEI %s latest GPS data has null coordinates - no fallback applied", data.IMEI)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gpsData,
		"count":   len(gpsData),
		"message": "Latest GPS data retrieved - coordinates preserved as-is (null if invalid)",
	})
}

// GetLatestValidGPSDataByIMEI returns the latest GPS data with valid coordinates for a specific device
func (gc *GPSController) GetLatestValidGPSDataByIMEI(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid IMEI format",
		})
		return
	}

	colors.PrintInfo("📍 Searching for valid GPS data for IMEI: %s", imei)

	var gpsData models.GPSData

	// ENHANCED: Try multiple fallback levels to find valid GPS data
	// Level 1: Latest GPS data with non-null and non-zero coordinates
	err := db.GetDB().Where("imei = ? AND latitude IS NOT NULL AND longitude IS NOT NULL AND latitude != 0 AND longitude != 0").
		Preload("Device").
		Preload("Vehicle").
		Order("timestamp DESC").
		First(&gpsData).Error

	if err != nil {
		colors.PrintWarning("📍 Level 1 failed for IMEI %s, trying Level 2 (non-null coordinates)...", imei)

		// Level 2: Latest GPS data with non-null coordinates (allow zero values)
		err = db.GetDB().Where("imei = ? AND latitude IS NOT NULL AND longitude IS NOT NULL").
			Preload("Device").
			Preload("Vehicle").
			Order("timestamp DESC").
			First(&gpsData).Error
	}

	if err != nil {
		colors.PrintWarning("📍 Level 2 failed for IMEI %s, trying Level 3 (any GPS data)...", imei)

		// Level 3: Any GPS data for this IMEI
		err = db.GetDB().Where("imei = ?").
			Preload("Device").
			Preload("Vehicle").
			Order("timestamp DESC").
			First(&gpsData).Error
	}

	if err != nil {
		colors.PrintError("📍 No GPS data found for IMEI %s: %v", imei, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "No GPS data found for this device",
			"message": "This device has never sent GPS data to the server",
			"imei":    imei,
		})
		return
	}

	// Check if we have valid coordinates
	hasValidCoords := gpsData.Latitude != nil && gpsData.Longitude != nil
	validCoordsMsg := "Found GPS data but coordinates are null/invalid"

	if hasValidCoords {
		lat := *gpsData.Latitude
		lng := *gpsData.Longitude
		if lat != 0 && lng != 0 && lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180 {
			validCoordsMsg = "Found GPS data with valid coordinates"
		}
	}

	colors.PrintSuccess("📍 %s for IMEI %s: timestamp=%s", validCoordsMsg, imei, gpsData.Timestamp.Format("2006-01-02 15:04:05"))

	c.JSON(http.StatusOK, gin.H{
		"success":               true,
		"data":                  gpsData,
		"message":               "Latest GPS data retrieved successfully",
		"has_valid_coordinates": hasValidCoords,
	})
}

// GetLatestGPSDataByIMEI returns the latest GPS data for a specific device (including null coordinates)
func (gc *GPSController) GetLatestGPSDataByIMEI(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid IMEI format",
		})
		return
	}

	var gpsData models.GPSData
	if err := db.GetDB().Where("imei = ?", imei).
		Preload("Device").
		Preload("Vehicle").
		Order("timestamp DESC").
		First(&gpsData).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "No GPS data found for this device",
		})
		return
	}

	// CRITICAL CHANGE: Do NOT apply coordinate fallback
	// Keep coordinates as null if they are null in latest GPS data
	// Frontend will handle this by showing empty map
	if gpsData.Latitude == nil || gpsData.Longitude == nil {
		colors.PrintInfo("📍 IMEI %s latest GPS data has null coordinates - preserving as-is", imei)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gpsData,
		"message": "Latest GPS data retrieved - coordinates preserved as-is (null if invalid)",
	})
}

// GetGPSRoute returns GPS route data for tracking
func (gc *GPSController) GetGPSRoute(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid IMEI format",
		})
		return
	}

	from := c.Query("from")
	to := c.Query("to")

	if from == "" || to == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "from and to query parameters are required",
		})
		return
	}

	fromTime, err := time.Parse("2006-01-02T15:04:05Z", from)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid from time format. Use: 2006-01-02T15:04:05Z",
		})
		return
	}

	toTime, err := time.Parse("2006-01-02T15:04:05Z", to)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid to time format. Use: 2006-01-02T15:04:05Z",
		})
		return
	}

	var gpsData []models.GPSData
	if err := db.GetDB().Where("imei = ? AND timestamp BETWEEN ? AND ? AND latitude IS NOT NULL AND longitude IS NOT NULL",
		imei, fromTime, toTime).
		Order("timestamp ASC").
		Find(&gpsData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch GPS route data",
		})
		return
	}

	// Create route points
	routePoints := make([]gin.H, len(gpsData))
	for i, data := range gpsData {
		routePoints[i] = gin.H{
			"latitude":  data.Latitude,
			"longitude": data.Longitude,
			"timestamp": data.Timestamp,
			"speed":     data.Speed,
			"course":    data.Course,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"imei":         imei,
		"from":         fromTime,
		"to":           toTime,
		"route":        routePoints,
		"total_points": len(routePoints),
		"message":      "GPS route retrieved successfully",
	})
}

// DeleteGPSData deletes GPS data (admin only - implement auth middleware)
func (gc *GPSController) DeleteGPSData(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid GPS data ID",
		})
		return
	}

	var gpsData models.GPSData
	if err := db.GetDB().First(&gpsData, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "GPS data not found",
		})
		return
	}

	if err := db.GetDB().Delete(&gpsData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete GPS data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "GPS data deleted successfully",
	})
}

// GetLatestValidGPSData returns the latest GPS data with valid coordinates for all devices
func (gc *GPSController) GetLatestValidGPSData(c *gin.Context) {
	var gpsData []models.GPSData

	// Get latest GPS data with valid coordinates for each IMEI
	// This query selects the most recent GPS record with non-null coordinates for each device
	if err := db.GetDB().Raw(`
		SELECT DISTINCT ON (imei) *
		FROM gps_data
		WHERE deleted_at IS NULL 
		AND latitude IS NOT NULL 
		AND longitude IS NOT NULL
		AND latitude != 0 
		AND longitude != 0
		ORDER BY imei, timestamp DESC
	`).Preload("Device").Preload("Vehicle").Scan(&gpsData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch latest valid GPS data",
		})
		return
	}

	colors.PrintInfo("📍 Retrieved latest valid GPS data for %d devices", len(gpsData))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gpsData,
		"count":   len(gpsData),
		"message": "Latest valid GPS data retrieved successfully",
	})
}

// GetLatestLocationData returns the latest GPS data with valid coordinates for all devices
// This is for location/positioning - coordinates are required
func (gc *GPSController) GetLatestLocationData(c *gin.Context) {
	var gpsData []models.GPSData

	// Get latest location data for each IMEI - ONLY records with valid coordinates
	if err := db.GetDB().Raw(`
		SELECT DISTINCT ON (imei) *
		FROM gps_data
		WHERE deleted_at IS NULL 
		AND latitude IS NOT NULL 
		AND longitude IS NOT NULL
		AND latitude != 0 
		AND longitude != 0
		ORDER BY imei, timestamp DESC
	`).Preload("Device").Preload("Vehicle").Scan(&gpsData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch latest location data",
		})
		return
	}

	colors.PrintInfo("📍 Retrieved latest location data for %d devices", len(gpsData))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gpsData,
		"count":   len(gpsData),
		"message": "Latest location data retrieved successfully",
		"type":    "location",
	})
}

// GetLatestStatusData returns the latest GPS data for device status information
// This is for status display - coordinates are not required
func (gc *GPSController) GetLatestStatusData(c *gin.Context) {
	var gpsData []models.GPSData

	// Get latest status data for each IMEI - regardless of coordinates
	if err := db.GetDB().Raw(`
		SELECT DISTINCT ON (imei) *
		FROM gps_data
		WHERE deleted_at IS NULL
		ORDER BY imei, timestamp DESC
	`).Preload("Device").Preload("Vehicle").Scan(&gpsData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch latest status data",
		})
		return
	}

	colors.PrintInfo("📊 Retrieved latest status data for %d devices", len(gpsData))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gpsData,
		"count":   len(gpsData),
		"message": "Latest status data retrieved successfully",
		"type":    "status",
	})
}

// GetLocationDataByIMEI returns the latest location data for a specific device
// This is for map positioning - will fallback through history to find valid coordinates
func (gc *GPSController) GetLocationDataByIMEI(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid IMEI format",
		})
		return
	}

	var gpsData models.GPSData

	// First try to get the latest GPS data with valid coordinates
	if err := db.GetDB().Where("imei = ? AND latitude IS NOT NULL AND longitude IS NOT NULL AND latitude != 0 AND longitude != 0").
		Preload("Device").
		Preload("Vehicle").
		Order("timestamp DESC").
		First(&gpsData).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "No location data with valid coordinates found for this device",
		})
		return
	}

	colors.PrintInfo("📍 Retrieved location data for IMEI %s: lat=%.12f, lng=%.12f",
		imei, *gpsData.Latitude, *gpsData.Longitude)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gpsData,
		"message": "Location data retrieved successfully",
		"type":    "location",
	})
}

// GetStatusDataByIMEI returns the latest status data for a specific device
// This is for device status information - coordinates are not required
func (gc *GPSController) GetStatusDataByIMEI(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid IMEI format",
		})
		return
	}

	var gpsData models.GPSData
	if err := db.GetDB().Where("imei = ?", imei).
		Preload("Device").
		Preload("Vehicle").
		Order("timestamp DESC").
		First(&gpsData).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "No status data found for this device",
		})
		return
	}

	colors.PrintInfo("📊 Retrieved status data for IMEI %s: ignition=%s, speed=%v, battery=%v",
		imei, gpsData.Ignition, gpsData.Speed, gpsData.VoltageLevel)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gpsData,
		"message": "Status data retrieved successfully",
		"type":    "status",
	})
}
