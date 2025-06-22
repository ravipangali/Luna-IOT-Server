package controllers

import (
	"net/http"
	"strconv"
	"time"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"

	"github.com/gin-gonic/gin"
)

// UserGPSController handles user-based GPS tracking operations
type UserGPSController struct{}

// NewUserGPSController creates a new user GPS controller
func NewUserGPSController() *UserGPSController {
	return &UserGPSController{}
}

// GetUserVehicleTracking returns tracking data for all vehicles accessible to the user
func (ugc *UserGPSController) GetUserVehicleTracking(c *gin.Context) {
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}
	user := currentUser.(*models.User)

	// Get user's accessible vehicles
	var userVehicles []models.UserVehicle
	if err := db.GetDB().Where("user_id = ? AND is_active = ?", user.ID, true).
		Preload("Vehicle").Find(&userVehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch user vehicles",
		})
		return
	}

	// Manually load device for each vehicle
	for i := range userVehicles {
		if err := userVehicles[i].Vehicle.LoadDevice(db.GetDB()); err != nil {
			// If device loading fails, continue with empty device
		}
	}

	var trackingData []map[string]interface{}

	for _, userVehicle := range userVehicles {
		if userVehicle.IsExpired() || !userVehicle.HasPermission(models.PermissionLiveTracking) {
			continue
		}

		// Get latest GPS data for this vehicle
		var latestGPS models.GPSData
		if err := db.GetDB().Where("imei = ?", userVehicle.Vehicle.IMEI).
			Order("timestamp DESC").First(&latestGPS).Error; err != nil {
			continue // Skip if no GPS data found
		}

		// Get latest valid location data (fallback through history)
		var locationData *models.GPSData
		var allGPSData []models.GPSData
		if err := db.GetDB().Where("imei = ?", userVehicle.Vehicle.IMEI).
			Order("timestamp DESC").Limit(50).Find(&allGPSData).Error; err == nil {

			for _, data := range allGPSData {
				if data.Latitude != nil && data.Longitude != nil {
					lat := *data.Latitude
					lng := *data.Longitude
					if lat != 0 && lng != 0 && lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180 {
						locationData = &data
						break
					}
				}
			}
		}

		vehicleTracking := map[string]interface{}{
			"imei":            userVehicle.Vehicle.IMEI,
			"reg_no":          userVehicle.Vehicle.RegNo,
			"name":            userVehicle.Vehicle.Name,
			"vehicle_type":    userVehicle.Vehicle.VehicleType,
			"user_role":       userVehicle.GetUserRole(),
			"permissions":     userVehicle.GetPermissions(),
			"latest_status":   latestGPS,
			"latest_location": locationData,
			"device":          userVehicle.Vehicle.Device,
		}

		trackingData = append(trackingData, vehicleTracking)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    trackingData,
		"count":   len(trackingData),
		"message": "User vehicle tracking data retrieved successfully",
	})
}

// GetUserVehicleLocation returns location data for a specific vehicle accessible to the user
func (ugc *UserGPSController) GetUserVehicleLocation(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid IMEI format",
		})
		return
	}

	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}
	user := currentUser.(*models.User)

	// Check user access to this vehicle
	var userVehicle models.UserVehicle
	if err := db.GetDB().Where("user_id = ? AND vehicle_id = ? AND is_active = ?",
		user.ID, imei, true).Preload("Vehicle").First(&userVehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Vehicle not found or access denied",
		})
		return
	}

	if userVehicle.IsExpired() || !userVehicle.HasPermission(models.PermissionLiveTracking) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "No live tracking permission for this vehicle",
		})
		return
	}

	// Get latest valid location data with historical fallback
	var allGPSData []models.GPSData
	if err := db.GetDB().Where("imei = ?", imei).
		Order("timestamp DESC").Limit(100).Find(&allGPSData).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "No GPS data found for this vehicle",
		})
		return
	}

	var locationData *models.GPSData
	for _, data := range allGPSData {
		if data.Latitude != nil && data.Longitude != nil {
			lat := *data.Latitude
			lng := *data.Longitude
			if lat != 0 && lng != 0 && lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180 {
				locationData = &data
				break
			}
		}
	}

	if locationData == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "No valid location data found for this vehicle",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"imei":        imei,
			"vehicle":     userVehicle.Vehicle,
			"permissions": userVehicle.GetPermissions(),
			"location":    locationData,
		},
		"message": "Vehicle location retrieved successfully",
	})
}

// GetUserVehicleStatus returns status data for a specific vehicle accessible to the user
func (ugc *UserGPSController) GetUserVehicleStatus(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid IMEI format",
		})
		return
	}

	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}
	user := currentUser.(*models.User)

	// Check user access to this vehicle
	var userVehicle models.UserVehicle
	if err := db.GetDB().Where("user_id = ? AND vehicle_id = ? AND is_active = ?",
		user.ID, imei, true).Preload("Vehicle").First(&userVehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Vehicle not found or access denied",
		})
		return
	}

	if userVehicle.IsExpired() || !userVehicle.HasPermission(models.PermissionLiveTracking) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "No live tracking permission for this vehicle",
		})
		return
	}

	// Get latest GPS data for status
	var latestGPS models.GPSData
	if err := db.GetDB().Where("imei = ?", imei).
		Order("timestamp DESC").First(&latestGPS).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "No status data found for this vehicle",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"imei":        imei,
			"vehicle":     userVehicle.Vehicle,
			"permissions": userVehicle.GetPermissions(),
			"status":      latestGPS,
		},
		"message": "Vehicle status retrieved successfully",
	})
}

// GetUserVehicleHistory returns GPS history for a specific vehicle accessible to the user
func (ugc *UserGPSController) GetUserVehicleHistory(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid IMEI format",
		})
		return
	}

	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}
	user := currentUser.(*models.User)

	// Check user access to this vehicle
	var userVehicle models.UserVehicle
	if err := db.GetDB().Where("user_id = ? AND vehicle_id = ? AND is_active = ?",
		user.ID, imei, true).Preload("Vehicle").First(&userVehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Vehicle not found or access denied",
		})
		return
	}

	if userVehicle.IsExpired() || !userVehicle.HasPermission(models.PermissionHistory) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "No history permission for this vehicle",
		})
		return
	}

	// Parse time filters
	query := db.GetDB().Where("imei = ?", imei)

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

	var gpsData []models.GPSData
	if err := query.Order("timestamp DESC").Limit(limit).Offset(offset).Find(&gpsData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch GPS history",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"imei":        imei,
			"vehicle":     userVehicle.Vehicle,
			"permissions": userVehicle.GetPermissions(),
			"history":     gpsData,
			"page":        page,
			"limit":       limit,
			"count":       len(gpsData),
		},
		"message": "Vehicle history retrieved successfully",
	})
}

// GetUserVehicleRoute returns route data for a specific vehicle accessible to the user
func (ugc *UserGPSController) GetUserVehicleRoute(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid IMEI format",
		})
		return
	}

	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}
	user := currentUser.(*models.User)

	// Check user access to this vehicle
	var userVehicle models.UserVehicle
	if err := db.GetDB().Where("user_id = ? AND vehicle_id = ? AND is_active = ?",
		user.ID, imei, true).Preload("Vehicle").First(&userVehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Vehicle not found or access denied",
		})
		return
	}

	if userVehicle.IsExpired() || !userVehicle.HasPermission(models.PermissionHistory) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "No history permission for this vehicle",
		})
		return
	}

	from := c.Query("from")
	to := c.Query("to")

	if from == "" || to == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "from and to query parameters are required",
		})
		return
	}

	fromTime, err := time.Parse("2006-01-02T15:04:05Z", from)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid from time format. Use: 2006-01-02T15:04:05Z",
		})
		return
	}

	toTime, err := time.Parse("2006-01-02T15:04:05Z", to)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid to time format. Use: 2006-01-02T15:04:05Z",
		})
		return
	}

	var gpsData []models.GPSData
	if err := db.GetDB().Where("imei = ? AND timestamp BETWEEN ? AND ? AND latitude IS NOT NULL AND longitude IS NOT NULL",
		imei, fromTime, toTime).Order("timestamp ASC").Find(&gpsData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch GPS route data",
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
			"ignition":  data.Ignition,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"imei":         imei,
			"vehicle":      userVehicle.Vehicle,
			"permissions":  userVehicle.GetPermissions(),
			"from":         fromTime,
			"to":           toTime,
			"route":        routePoints,
			"total_points": len(routePoints),
		},
		"message": "Vehicle route retrieved successfully",
	})
}

// GetUserVehicleReport returns analytics/report data for vehicles accessible to the user
func (ugc *UserGPSController) GetUserVehicleReport(c *gin.Context) {
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}
	user := currentUser.(*models.User)

	// Get user's accessible vehicles with report permission
	var userVehicles []models.UserVehicle
	if err := db.GetDB().Where("user_id = ? AND is_active = ? AND (report = ? OR all_access = ?)",
		user.ID, true, true, true).Preload("Vehicle").Find(&userVehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch user vehicles",
		})
		return
	}

	// Parse date range
	from := c.DefaultQuery("from", time.Now().AddDate(0, 0, -7).Format("2006-01-02T15:04:05Z"))
	to := c.DefaultQuery("to", time.Now().Format("2006-01-02T15:04:05Z"))

	fromTime, _ := time.Parse("2006-01-02T15:04:05Z", from)
	toTime, _ := time.Parse("2006-01-02T15:04:05Z", to)

	var reportData []map[string]interface{}

	for _, userVehicle := range userVehicles {
		if userVehicle.IsExpired() {
			continue
		}

		// Get GPS data for the date range
		var gpsData []models.GPSData
		if err := db.GetDB().Where("imei = ? AND timestamp BETWEEN ? AND ?",
			userVehicle.Vehicle.IMEI, fromTime, toTime).Find(&gpsData).Error; err != nil {
			continue
		}

		// Calculate basic statistics
		totalPoints := len(gpsData)
		totalDistance := 0.0
		maxSpeed := 0
		totalIgnitionOnTime := 0.0

		var lastPoint *models.GPSData
		var ignitionOnStart *time.Time

		for _, data := range gpsData {
			// Calculate distance if we have coordinates
			if lastPoint != nil && data.Latitude != nil && data.Longitude != nil &&
				lastPoint.Latitude != nil && lastPoint.Longitude != nil {
				totalDistance += calculateDistance(*lastPoint.Latitude, *lastPoint.Longitude,
					*data.Latitude, *data.Longitude)
			}

			// Track max speed
			if data.Speed != nil && *data.Speed > maxSpeed {
				maxSpeed = *data.Speed
			}

			// Track ignition time
			if data.Ignition == "ON" && ignitionOnStart == nil {
				ignitionOnStart = &data.Timestamp
			} else if data.Ignition == "OFF" && ignitionOnStart != nil {
				totalIgnitionOnTime += data.Timestamp.Sub(*ignitionOnStart).Hours()
				ignitionOnStart = nil
			}

			lastPoint = &data
		}

		// If ignition was still on at the end
		if ignitionOnStart != nil && lastPoint != nil {
			totalIgnitionOnTime += lastPoint.Timestamp.Sub(*ignitionOnStart).Hours()
		}

		avgSpeed := 0.0
		if totalIgnitionOnTime > 0 {
			avgSpeed = totalDistance / totalIgnitionOnTime
		}

		vehicleReport := map[string]interface{}{
			"imei":              userVehicle.Vehicle.IMEI,
			"reg_no":            userVehicle.Vehicle.RegNo,
			"name":              userVehicle.Vehicle.Name,
			"vehicle_type":      userVehicle.Vehicle.VehicleType,
			"permissions":       userVehicle.GetPermissions(),
			"total_points":      totalPoints,
			"total_distance":    totalDistance,
			"max_speed":         maxSpeed,
			"avg_speed":         avgSpeed,
			"ignition_on_hours": totalIgnitionOnTime,
			"from":              fromTime,
			"to":                toTime,
		}

		reportData = append(reportData, vehicleReport)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    reportData,
		"count":   len(reportData),
		"from":    fromTime,
		"to":      toTime,
		"message": "User vehicle reports retrieved successfully",
	})
}

// Helper function to calculate distance between two coordinates (Haversine formula)
func calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371 // Earth's radius in kilometers

	dLat := (lat2 - lat1) * (3.14159265359 / 180)
	dLng := (lng2 - lng1) * (3.14159265359 / 180)

	a := 0.5 - (0.5 * (1 + (dLat * dLat))) +
		(1+lat1*(3.14159265359/180))*(1+lat2*(3.14159265359/180))*
			0.5*(1-(1+(dLng*dLng)))

	return R * 2 * (a * a)
}
