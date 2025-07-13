package controllers

import (
	"net/http"
	"time"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"
	"luna_iot_server/pkg/utils"

	"github.com/gin-gonic/gin"
)

// UserTrackingController handles all user-based tracking operations
type UserTrackingController struct{}

// NewUserTrackingController creates a new user tracking controller
func NewUserTrackingController() *UserTrackingController {
	return &UserTrackingController{}
}

// GetMyVehiclesTracking returns real-time tracking data for all user's vehicles
func (utc *UserTrackingController) GetMyVehiclesTracking(c *gin.Context) {
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User not authenticated"})
		return
	}
	user := currentUser.(*models.User)

	// Get user's accessible vehicles with live tracking permission
	var userVehicles []models.UserVehicle
	if err := db.GetDB().
		Where("user_id = ? AND is_active = ? AND (live_tracking = ? OR all_access = ?)", user.ID, true, true, true).
		Preload("Vehicle.UserAccess.User"). // Preload related data for permissions
		Find(&userVehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to fetch user vehicles"})
		return
	}

	if len(userVehicles) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    []map[string]interface{}{},
			"count":   0,
			"message": "User has no accessible vehicles.",
		})
		return
	}

	// Extract all vehicle IMEIs for an efficient bulk query
	var imeis []string
	for _, uv := range userVehicles {
		imeis = append(imeis, uv.VehicleID)
	}

	// Efficiently fetch the latest GPS data for all vehicles in a single query
	var latestGpsData []models.GPSData
	subQuery := db.GetDB().
		Select("MAX(id) as id").
		Model(&models.GPSData{}).
		Where("imei IN ?", imeis).
		Group("imei")

	if err := db.GetDB().
		Where("id IN (?)", subQuery).
		Find(&latestGpsData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to fetch latest GPS data"})
		return
	}

	// Create a map for quick lookup of GPS data by IMEI
	gpsDataMap := make(map[string]models.GPSData)
	for _, gps := range latestGpsData {
		gpsDataMap[gps.IMEI] = gps
	}

	// Manually load device for each vehicle and build the response
	var trackingData []map[string]interface{}
	for i := range userVehicles {
		vehicle := userVehicles[i].Vehicle
		if err := vehicle.LoadDevice(db.GetDB()); err != nil {
			colors.PrintWarning("Failed to load device for vehicle %s: %v", vehicle.IMEI, err)
			// Continue without device info, or handle as an error
		}

		if userVehicles[i].IsExpired() {
			continue // Skip expired vehicle access
		}

		vehicleData := map[string]interface{}{
			"vehicle":    vehicle,
			"latest_gps": nil, // Default to null
		}

		// If GPS data exists for this IMEI, add it to the response
		if gpsData, ok := gpsDataMap[vehicle.IMEI]; ok {
			vehicleData["latest_gps"] = gpsData
		}

		trackingData = append(trackingData, vehicleData)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    trackingData,
		"count":   len(trackingData),
		"message": "User vehicles tracking data retrieved successfully",
	})
}

// GetMyVehicleTracking returns detailed tracking data for a specific vehicle
func (utc *UserTrackingController) GetMyVehicleTracking(c *gin.Context) {
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

	// Manually load device for the vehicle
	if err := userVehicle.Vehicle.LoadDevice(db.GetDB()); err != nil {
		colors.PrintWarning("Failed to load device for vehicle %s: %v", userVehicle.Vehicle.IMEI, err)
	}

	if userVehicle.IsExpired() {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Vehicle access has expired",
		})
		return
	}

	if !userVehicle.HasPermission(models.PermissionLiveTracking) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "No live tracking permission for this vehicle",
		})
		return
	}

	// Get latest status data
	var latestGPS models.GPSData
	hasStatusData := false
	if err := db.GetDB().Where("imei = ?", imei).
		Order("timestamp DESC").First(&latestGPS).Error; err == nil {
		hasStatusData = true
	}

	// Get latest valid location data with extensive historical fallback
	var locationData *models.GPSData
	var allGPSData []models.GPSData
	hasLocationData := false
	if err := db.GetDB().Where("imei = ?", imei).
		Order("timestamp DESC").Limit(100).Find(&allGPSData).Error; err == nil {

		for _, data := range allGPSData {
			if data.Latitude != nil && data.Longitude != nil {
				lat := *data.Latitude
				lng := *data.Longitude
				if lat != 0 && lng != 0 && lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180 {
					locationData = &data
					hasLocationData = true
					break
				}
			}
		}
	}

	// Calculate vehicle statistics for today
	today := time.Now()
	startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	var todayData []models.GPSData
	db.GetDB().Where("imei = ? AND timestamp >= ?", imei, startOfDay).
		Order("timestamp ASC").Find(&todayData)

	stats := utc.calculateVehicleStats(todayData, userVehicle.Vehicle.Overspeed)

	response := gin.H{
		"success": true,
		"data": map[string]interface{}{
			"vehicle":           userVehicle.Vehicle,
			"permissions":       userVehicle.GetPermissions(),
			"user_role":         userVehicle.GetUserRole(),
			"has_status_data":   hasStatusData,
			"has_location_data": hasLocationData,
			"today_stats":       stats,
		},
		"message": "Vehicle tracking data retrieved successfully",
	}

	if hasStatusData {
		response["data"].(map[string]interface{})["latest_status"] = latestGPS
	}

	if hasLocationData {
		response["data"].(map[string]interface{})["latest_location"] = locationData
	}

	c.JSON(http.StatusOK, response)
}

// GetMyVehicleLocation returns location data for user's vehicle
func (utc *UserTrackingController) GetMyVehicleLocation(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid IMEI format",
		})
		return
	}

	userVehicle, err := utc.validateUserVehicleAccess(c, imei, models.PermissionLiveTracking)
	if err != nil {
		return // Error already sent in response
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

// GetMyVehicleStatus returns status data for user's vehicle
func (utc *UserTrackingController) GetMyVehicleStatus(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid IMEI format",
		})
		return
	}

	userVehicle, err := utc.validateUserVehicleAccess(c, imei, models.PermissionLiveTracking)
	if err != nil {
		return // Error already sent in response
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

// GetMyVehicleHistory returns GPS history for user's vehicle
func (utc *UserTrackingController) GetMyVehicleHistory(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid IMEI format",
		})
		return
	}

	userVehicle, err := utc.validateUserVehicleAccess(c, imei, models.PermissionHistory)
	if err != nil {
		return // Error already sent in response
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

	// Get ALL GPS data for the date range (no pagination for history)
	// Order by timestamp ASC (oldest first) for proper route plotting
	var gpsData []models.GPSData
	if err := query.Order("timestamp ASC").Find(&gpsData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch GPS history",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"imei":                imei,
			"vehicle":             userVehicle.Vehicle,
			"permissions":         userVehicle.GetPermissions(),
			"history":             gpsData,
			"count":               len(gpsData),
			"overspeed_threshold": userVehicle.Vehicle.Overspeed, // Add overspeed threshold
		},
		"message": "Vehicle history retrieved successfully",
	})

	// Debug: Log the overspeed threshold being sent
	colors.PrintInfo("Vehicle history response - IMEI: %s, Overspeed threshold: %d km/h", imei, userVehicle.Vehicle.Overspeed)
}

// GetMyVehicleRoute returns route data for user's vehicle
func (utc *UserTrackingController) GetMyVehicleRoute(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid IMEI format",
		})
		return
	}

	userVehicle, err := utc.validateUserVehicleAccess(c, imei, models.PermissionHistory)
	if err != nil {
		return // Error already sent in response
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
	if err := db.GetDB().Where("imei = ? AND timestamp BETWEEN ? AND ? AND latitude IS NOT NULL AND longitude IS NOT NULL AND speed IS NOT NULL",
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

	// Calculate route statistics
	stats := utc.calculateVehicleStats(gpsData, userVehicle.Vehicle.Overspeed)

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
			"statistics":   stats,
		},
		"message": "Vehicle route retrieved successfully",
	})
}

// GetMyVehicleReports returns analytics/report data for user's vehicles
func (utc *UserTrackingController) GetMyVehicleReports(c *gin.Context) {
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}
	user := currentUser.(*models.User)

	// Parse date range
	from := c.DefaultQuery("from", time.Now().AddDate(0, 0, -7).Format("2006-01-02T15:04:05Z"))
	to := c.DefaultQuery("to", time.Now().Format("2006-01-02T15:04:05Z"))

	fromTime, _ := time.Parse("2006-01-02T15:04:05Z", from)
	toTime, _ := time.Parse("2006-01-02T15:04:05Z", to)

	// Get user's vehicles with report permission
	var userVehicles []models.UserVehicle
	if err := db.GetDB().Where("user_id = ? AND is_active = ? AND (report = ? OR all_access = ?)",
		user.ID, true, true, true).Preload("Vehicle").Find(&userVehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch user vehicles",
		})
		return
	}

	var reportData []map[string]interface{}

	for _, userVehicle := range userVehicles {
		if userVehicle.IsExpired() {
			continue
		}

		// Get GPS data for the date range
		var gpsData []models.GPSData
		if err := db.GetDB().Where("imei = ? AND timestamp BETWEEN ? AND ?",
			userVehicle.Vehicle.IMEI, fromTime, toTime).Order("timestamp ASC").Find(&gpsData).Error; err != nil {
			continue
		}

		stats := utc.calculateVehicleStats(gpsData, userVehicle.Vehicle.Overspeed)

		vehicleReport := map[string]interface{}{
			"imei":         userVehicle.Vehicle.IMEI,
			"reg_no":       userVehicle.Vehicle.RegNo,
			"name":         userVehicle.Vehicle.Name,
			"vehicle_type": userVehicle.Vehicle.VehicleType,
			"permissions":  userVehicle.GetPermissions(),
			"from":         fromTime,
			"to":           toTime,
			"statistics":   stats,
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

// Helper function to validate user vehicle access
func (utc *UserTrackingController) validateUserVehicleAccess(c *gin.Context, imei string, permission models.Permission) (*models.UserVehicle, error) {
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return nil, gin.Error{Err: nil}
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
		return nil, err
	}

	// Manually load device for the vehicle
	if err := userVehicle.Vehicle.LoadDevice(db.GetDB()); err != nil {
		colors.PrintWarning("Failed to load device for vehicle %s: %v", userVehicle.Vehicle.IMEI, err)
	}

	if userVehicle.IsExpired() {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Vehicle access has expired",
		})
		return nil, gin.Error{Err: nil}
	}

	if !userVehicle.HasPermission(permission) && !userVehicle.HasPermission(models.PermissionAllAccess) {
		c.JSON(http.StatusForbidden, gin.H{
			"success":             false,
			"error":               "Insufficient permissions for this operation",
			"required_permission": string(permission),
			"user_permissions":    userVehicle.GetPermissions(),
		})
		return nil, gin.Error{Err: nil}
	}

	return &userVehicle, nil
}

type vehicleState int

const (
	stateUnknown vehicleState = iota
	stateOverspeed
	stateRunning
	stateIdle
	stateStopped
)

func getVehicleState(data models.GPSData, overspeedThreshold int) vehicleState {
	speed := 0
	if data.Speed != nil {
		speed = *data.Speed
	}
	ignitionOn := data.Ignition == "ON"

	if speed > overspeedThreshold {
		return stateOverspeed
	}
	if speed > 5 {
		return stateRunning
	}
	if ignitionOn {
		return stateIdle
	}
	return stateStopped
}

// Helper function to calculate vehicle statistics
func (utc *UserTrackingController) calculateVehicleStats(gpsData []models.GPSData, vehicleOverspeed int) map[string]interface{} {
	if len(gpsData) < 2 {
		return map[string]interface{}{
			"total_points":         0,
			"total_distance":       0.0,
			"max_speed":            0,
			"avg_speed":            0.0,
			"ignition_on_hours":    0.0,
			"moving_time_hours":    0.0,
			"running_time_hours":   0.0,
			"overspeed_time_hours": 0.0,
			"idle_time_hours":      0.0,
			"stopped_time_hours":   0.0,
		}
	}

	totalPoints := len(gpsData)
	var totalDistance float64
	maxSpeed := 0

	var totalIgnitionOnTime, movingTime, runningTime, overspeedTime, idleTime, stoppedTime time.Duration

	// Calculate total distance and max speed first
	for i := 0; i < len(gpsData); i++ {
		if gpsData[i].Speed != nil && *gpsData[i].Speed > maxSpeed {
			maxSpeed = *gpsData[i].Speed
		}
		if i > 0 {
			p1 := gpsData[i-1]
			p2 := gpsData[i]
			if p1.Latitude != nil && p1.Longitude != nil && p2.Latitude != nil && p2.Longitude != nil {
				totalDistance += utils.CalculateDistance(*p1.Latitude, *p1.Longitude, *p2.Latitude, *p2.Longitude)
			}
		}
	}

	// Calculate state durations
	for i := 1; i < len(gpsData); i++ {
		p1 := gpsData[i-1]
		p2 := gpsData[i]
		duration := p2.Timestamp.Sub(p1.Timestamp)

		// State is determined by the starting point of the interval
		state := getVehicleState(p1, vehicleOverspeed)

		switch state {
		case stateOverspeed:
			overspeedTime += duration
		case stateRunning:
			runningTime += duration
		case stateIdle:
			idleTime += duration
		case stateStopped:
			stoppedTime += duration
		}

		// Independently calculate ignition on time
		if p1.Ignition == "ON" {
			totalIgnitionOnTime += duration
		}
	}

	movingTime = runningTime + overspeedTime
	avgSpeed := 0.0
	if movingTime.Hours() > 0 {
		avgSpeed = totalDistance / movingTime.Hours()
	}

	stats := map[string]interface{}{
		"total_points":         totalPoints,
		"total_distance":       totalDistance,
		"max_speed":            maxSpeed,
		"avg_speed":            avgSpeed,
		"ignition_on_hours":    totalIgnitionOnTime.Hours(),
		"moving_time_hours":    movingTime.Hours(),
		"running_time_hours":   runningTime.Hours(),
		"overspeed_time_hours": overspeedTime.Hours(),
		"idle_time_hours":      idleTime.Hours(),
		"stopped_time_hours":   stoppedTime.Hours(),
	}

	return stats
}
