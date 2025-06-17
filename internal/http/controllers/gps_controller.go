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
			"error": "Failed to fetch GPS data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
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
			"error": "Failed to fetch GPS data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
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

	// ENHANCED FIX: Get latest GPS data for each IMEI regardless of device connection
	// This ensures we always have GPS data even when devices are disconnected
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

	// ENHANCED: For each GPS data entry, if coordinates are null, get latest valid coordinates
	// Status is always based on latest GPS data, coordinates use fallback for positioning
	for i, data := range gpsData {
		if data.Latitude == nil || data.Longitude == nil {
			// Find latest GPS data with valid coordinates for this IMEI
			var validGPS models.GPSData
			if err := db.GetDB().Where("imei = ? AND latitude IS NOT NULL AND longitude IS NOT NULL", data.IMEI).
				Order("timestamp DESC").
				First(&validGPS).Error; err == nil {
				// Use coordinates from latest valid location for map positioning
				// Keep original timestamp for accurate status calculation
				gpsData[i].Latitude = validGPS.Latitude
				gpsData[i].Longitude = validGPS.Longitude

				// Log the coordinate fallback for debugging
				colors.PrintInfo("📍 Using coordinate fallback for %s: lat=%.6f, lng=%.6f from %s",
					data.IMEI, *validGPS.Latitude, *validGPS.Longitude,
					validGPS.Timestamp.Format("2006-01-02T15:04:05Z"))
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gpsData,
		"count":   len(gpsData),
		"message": "Latest GPS data retrieved - coordinates with database fallback, status from latest timestamp",
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

	var gpsData models.GPSData

	// First try to get the latest GPS data with valid coordinates
	if err := db.GetDB().Where("imei = ? AND latitude IS NOT NULL AND longitude IS NOT NULL").
		Preload("Device").
		Preload("Vehicle").
		Order("timestamp DESC").
		First(&gpsData).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "No GPS data with valid coordinates found for this device",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    gpsData,
		"message": "Latest valid GPS data retrieved successfully",
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

	// ENHANCED: If coordinates are null, get latest valid coordinates for positioning
	if gpsData.Latitude == nil || gpsData.Longitude == nil {
		var validGPS models.GPSData
		if err := db.GetDB().Where("imei = ? AND latitude IS NOT NULL AND longitude IS NOT NULL", imei).
			Order("timestamp DESC").
			First(&validGPS).Error; err == nil {
			// Use coordinates from latest valid location
			gpsData.Latitude = validGPS.Latitude
			gpsData.Longitude = validGPS.Longitude

			colors.PrintInfo("📍 Coordinate fallback for %s: using lat=%.6f, lng=%.6f from %s",
				imei, *validGPS.Latitude, *validGPS.Longitude,
				validGPS.Timestamp.Format("2006-01-02T15:04:05Z"))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gpsData,
		"message": "Latest GPS data retrieved - coordinates with database fallback if needed",
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
