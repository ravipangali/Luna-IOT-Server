package controllers

import (
	"net/http"
	"strconv"
	"time"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"
	"luna_iot_server/pkg/utils"

	"github.com/gin-gonic/gin"
)

// VehicleController handles vehicle-related HTTP requests
type VehicleController struct{}

// NewVehicleController creates a new vehicle controller
func NewVehicleController() *VehicleController {
	return &VehicleController{}
}

// GetVehicles returns all vehicles with pagination and filtering
func (vc *VehicleController) GetVehicles(c *gin.Context) {
	// Parse query parameters with defaults
	page := parseInt(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	limit := parseInt(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Optional filtering
	var query = db.GetDB()

	if vehicleType := c.Query("type"); vehicleType != "" {
		query = query.Where("vehicle_type = ?", vehicleType)
	}

	if regNo := c.Query("reg_no"); regNo != "" {
		query = query.Where("reg_no ILIKE ?", "%"+regNo+"%")
	}

	if name := c.Query("name"); name != "" {
		query = query.Where("name ILIKE ?", "%"+name+"%")
	}

	if imei := c.Query("imei"); imei != "" {
		query = query.Where("imei LIKE ?", "%"+imei+"%")
	}

	if userId := c.Query("userId"); userId != "" {
		// If userId is provided, filter vehicles for that user
		query = query.Joins("JOIN user_vehicles ON user_vehicles.vehicle_id = vehicles.imei").
			Where("user_vehicles.user_id = ? AND user_vehicles.is_active = ?", userId, true)
	}

	// Get total count for pagination
	var totalCount int64
	if err := query.Model(&models.Vehicle{}).Count(&totalCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to count vehicles",
		})
		return
	}

	// Get vehicles with pagination
	var vehicles []models.Vehicle
	if err := query.Limit(limit).Offset(offset).Find(&vehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch vehicles",
		})
		return
	}

	// Load additional data for each vehicle
	for i := range vehicles {
		// Load device information
		var device models.Device
		if err := db.GetDB().Where("imei = ?", vehicles[i].IMEI).First(&device).Error; err == nil {
			vehicles[i].Device = device
		}

		// Load user access information
		var userAccess []models.UserVehicle
		if err := db.GetDB().Preload("User").Where("vehicle_id = ? AND is_active = ?", vehicles[i].IMEI, true).Find(&userAccess).Error; err == nil {
			vehicles[i].UserAccess = userAccess
		}
	}

	// Calculate pagination info
	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))

	// Enhanced response with user summary
	var vehicleList []map[string]interface{}
	for _, vehicle := range vehicles {
		mainUserCount := 0
		sharedUserCount := 0
		var mainUserName string

		for _, access := range vehicle.UserAccess {
			if access.IsExpired() {
				continue
			}
			if access.IsMainUser {
				mainUserCount++
				mainUserName = access.User.Name
			} else {
				sharedUserCount++
			}
		}

		vehicleInfo := map[string]interface{}{
			"imei":              vehicle.IMEI,
			"reg_no":            vehicle.RegNo,
			"name":              vehicle.Name,
			"vehicle_type":      vehicle.VehicleType,
			"odometer":          vehicle.Odometer,
			"mileage":           vehicle.Mileage,
			"min_fuel":          vehicle.MinFuel,
			"overspeed":         vehicle.Overspeed,
			"created_at":        vehicle.CreatedAt,
			"updated_at":        vehicle.UpdatedAt,
			"device":            vehicle.Device,
			"main_user_count":   mainUserCount,
			"shared_user_count": sharedUserCount,
			"total_user_count":  mainUserCount + sharedUserCount,
			"main_user_name":    mainUserName,
		}

		vehicleList = append(vehicleList, vehicleInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": vehicleList,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total_count": totalCount,
			"total_pages": totalPages,
			"has_next":    page < totalPages,
			"has_prev":    page > 1,
		},
		"message": "Vehicles retrieved successfully",
	})
}

// Helper function to parse integer
func parseInt(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}

// GetVehicle returns a single vehicle by IMEI
func (vc *VehicleController) GetVehicle(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid IMEI format",
		})
		return
	}

	var vehicle models.Vehicle
	if err := db.GetDB().Preload("Device").Where("imei = ?", imei).First(&vehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Vehicle not found",
		})
		return
	}

	// Load user access information with user details
	var userAccess []models.UserVehicle
	if err := db.GetDB().Preload("User").Where("vehicle_id = ? AND is_active = ?", vehicle.IMEI, true).Find(&userAccess).Error; err == nil {
		vehicle.UserAccess = userAccess
	}

	// Organize users by their roles
	var mainUsers []map[string]interface{}
	var sharedUsers []map[string]interface{}

	for _, access := range vehicle.UserAccess {
		if access.IsExpired() {
			continue // Skip expired access
		}

		userInfo := map[string]interface{}{
			"id":          access.User.ID,
			"name":        access.User.Name,
			"email":       access.User.Email,
			"role":        access.GetUserRole(),
			"permissions": access.GetPermissions(),
			"granted_at":  access.GrantedAt,
			"expires_at":  access.ExpiresAt,
			"notes":       access.Notes,
			"is_active":   access.IsActive,
		}

		if access.IsMainUser {
			mainUsers = append(mainUsers, userInfo)
		} else {
			sharedUsers = append(sharedUsers, userInfo)
		}
	}

	response := gin.H{
		"data":    vehicle,
		"message": "Vehicle retrieved successfully",
		"users": gin.H{
			"main_users":   mainUsers,
			"shared_users": sharedUsers,
			"total_users":  len(vehicle.UserAccess),
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetVehicleByRegNo returns a vehicle by registration number
func (vc *VehicleController) GetVehicleByRegNo(c *gin.Context) {
	regNo := c.Param("reg_no")

	var vehicle models.Vehicle
	if err := db.GetDB().Preload("Device").Where("reg_no = ?", regNo).First(&vehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Vehicle not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    vehicle,
		"message": "Vehicle retrieved successfully",
	})
}

// CreateVehicle creates a new vehicle
func (vc *VehicleController) CreateVehicle(c *gin.Context) {
	var requestData struct {
		models.Vehicle
		MainUserID uint `json:"main_user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		colors.PrintError("Invalid JSON in vehicle creation request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
			"message": "main_user_id is required for vehicle creation",
		})
		return
	}

	vehicle := requestData.Vehicle
	mainUserID := requestData.MainUserID

	colors.PrintInfo("Creating vehicle with IMEI: %s, RegNo: %s, Type: %s, MainUser: %d",
		vehicle.IMEI, vehicle.RegNo, vehicle.VehicleType, mainUserID)

	// Validate main user exists
	var mainUser models.User
	if err := db.GetDB().First(&mainUser, mainUserID).Error; err != nil {
		colors.PrintWarning("Main user with ID %d not found", mainUserID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Main user not found",
			"message": "The specified main user does not exist",
		})
		return
	}

	// Validate IMEI length
	if len(vehicle.IMEI) != 16 {
		colors.PrintWarning("Invalid IMEI length: %d (expected 16)", len(vehicle.IMEI))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "IMEI must be exactly 16 digits",
		})
		return
	}

	// Validate IMEI contains only digits
	for _, char := range vehicle.IMEI {
		if char < '0' || char > '9' {
			colors.PrintWarning("Invalid IMEI format: contains non-digit characters")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "IMEI must contain only digits",
			})
			return
		}
	}

	// Validate vehicle type
	validTypes := []models.VehicleType{
		models.VehicleTypeBike,
		models.VehicleTypeCar,
		models.VehicleTypeTruck,
		models.VehicleTypeBus,
		models.VehicleTypeSchoolBus,
	}

	isValidType := false
	for _, validType := range validTypes {
		if vehicle.VehicleType == validType {
			isValidType = true
			break
		}
	}

	if !isValidType {
		colors.PrintWarning("Invalid vehicle type: %s", vehicle.VehicleType)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       "Invalid vehicle type",
			"valid_types": []string{"bike", "car", "truck", "bus", "school_bus"},
		})
		return
	}

	// Check if device exists
	var device models.Device
	if err := db.GetDB().Where("imei = ?", vehicle.IMEI).First(&device).Error; err != nil {
		colors.PrintWarning("Device with IMEI %s not found in database", vehicle.IMEI)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device with this IMEI does not exist",
			"hint":  "Please register the device first",
		})
		return
	}

	// Check if vehicle with same registration number already exists
	var existingVehicle models.Vehicle
	if err := db.GetDB().Where("reg_no = ?", vehicle.RegNo).First(&existingVehicle).Error; err == nil {
		colors.PrintWarning("Vehicle with registration number %s already exists", vehicle.RegNo)
		c.JSON(http.StatusConflict, gin.H{
			"error": "Vehicle with this registration number already exists",
		})
		return
	}

	// Check if this IMEI is already assigned to another vehicle
	if err := db.GetDB().Where("imei = ?", vehicle.IMEI).First(&existingVehicle).Error; err == nil {
		colors.PrintWarning("IMEI %s is already assigned to vehicle %s", vehicle.IMEI, existingVehicle.RegNo)
		c.JSON(http.StatusConflict, gin.H{
			"error":            "This device is already assigned to another vehicle",
			"existing_vehicle": existingVehicle.RegNo,
		})
		return
	}

	// Get current user for audit
	currentUser, exists := c.Get("user")
	var grantedBy uint
	if exists {
		grantedBy = currentUser.(*models.User).ID
	} else {
		grantedBy = mainUserID // Fallback to main user if no current user
	}

	// Verify the grantedBy user exists (for foreign key constraint)
	var grantedByUser models.User
	if err := db.GetDB().First(&grantedByUser, grantedBy).Error; err != nil {
		colors.PrintWarning("GrantedBy user with ID %d not found, using main user", grantedBy)
		grantedBy = mainUserID
	}

	// Start transaction
	tx := db.GetDB().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create the vehicle
	if err := tx.Create(&vehicle).Error; err != nil {
		tx.Rollback()
		colors.PrintError("Failed to create vehicle in database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create vehicle",
			"details": err.Error(),
		})
		return
	}

	// Create main user assignment - create the struct manually to avoid potential issues
	mainUserAssignment := &models.UserVehicle{
		UserID:        mainUserID,
		VehicleID:     vehicle.IMEI,
		AllAccess:     true,
		LiveTracking:  true,
		History:       true,
		Report:        true,
		VehicleEdit:   true,
		Notification:  true,
		ShareTracking: true,
		IsMainUser:    true,
		GrantedBy:     grantedBy,
		GrantedAt:     time.Now(),
		IsActive:      true,
		Notes:         "Main user (Vehicle Owner)",
	}

	colors.PrintInfo("Creating user-vehicle assignment: UserID=%d, VehicleID=%s, GrantedBy=%d",
		mainUserAssignment.UserID, mainUserAssignment.VehicleID, mainUserAssignment.GrantedBy)

	if err := tx.Create(mainUserAssignment).Error; err != nil {
		tx.Rollback()
		colors.PrintError("Failed to assign main user to vehicle: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to assign main user to vehicle",
			"details": err.Error(),
		})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		colors.PrintError("Failed to commit vehicle creation transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to complete vehicle creation",
		})
		return
	}

	// Load device information and user assignments
	db.GetDB().Where("imei = ?", vehicle.IMEI).First(&device)
	vehicle.Device = device

	// Load user access information
	db.GetDB().Preload("User").Where("vehicle_id = ?", vehicle.IMEI).Find(&vehicle.UserAccess)

	colors.PrintSuccess("Vehicle created successfully: IMEI=%s, RegNo=%s, MainUser=%s",
		vehicle.IMEI, vehicle.RegNo, mainUser.Email)

	c.JSON(http.StatusCreated, gin.H{
		"data":    vehicle,
		"message": "Vehicle created successfully with main user assigned",
		"main_user": gin.H{
			"id":    mainUser.ID,
			"name":  mainUser.Name,
			"email": mainUser.Email,
		},
	})
}

// UpdateVehicle updates an existing vehicle
func (vc *VehicleController) UpdateVehicle(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid IMEI format",
		})
		return
	}

	var vehicle models.Vehicle
	if err := db.GetDB().Where("imei = ?", imei).First(&vehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Vehicle not found",
		})
		return
	}

	var updateData models.Vehicle
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
		})
		return
	}

	// Don't allow IMEI or registration number updates
	updateData.IMEI = vehicle.IMEI
	updateData.RegNo = vehicle.RegNo

	if err := db.GetDB().Model(&vehicle).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update vehicle",
		})
		return
	}

	// Manually load the device information
	var device models.Device
	if err := db.GetDB().Where("imei = ?", vehicle.IMEI).First(&device).Error; err == nil {
		vehicle.Device = device
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    vehicle,
		"message": "Vehicle updated successfully",
	})
}

// DeleteVehicle deletes a vehicle
func (vc *VehicleController) DeleteVehicle(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid IMEI format",
		})
		return
	}

	var vehicle models.Vehicle
	if err := db.GetDB().Where("imei = ?", imei).First(&vehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Vehicle not found",
		})
		return
	}

	if err := db.GetDB().Delete(&vehicle).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete vehicle",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Vehicle deleted successfully",
	})
}

// GetVehiclesByType returns vehicles filtered by type
func (vc *VehicleController) GetVehiclesByType(c *gin.Context) {
	vehicleType := c.Param("type")

	var vehicles []models.Vehicle
	if err := db.GetDB().Where("vehicle_type = ?", vehicleType).Find(&vehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch vehicles",
		})
		return
	}

	// Manually load device information for each vehicle
	for i := range vehicles {
		var device models.Device
		if err := db.GetDB().Where("imei = ?", vehicles[i].IMEI).First(&device).Error; err == nil {
			vehicles[i].Device = device
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    vehicles,
		"count":   len(vehicles),
		"message": "Vehicles retrieved successfully",
	})
}

// ===== CUSTOMER VEHICLE MANAGEMENT METHODS =====

// GetMyVehicles returns vehicles accessible to the current user
func (vc *VehicleController) GetMyVehicles(c *gin.Context) {
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}
	user := currentUser.(*models.User)

	// Get user's vehicle access with vehicle data preloaded
	var userVehicles []models.UserVehicle
	if err := db.GetDB().
		Where("user_id = ? AND is_active = ?", user.ID, true).
		Preload("Vehicle").
		Find(&userVehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch vehicles",
		})
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

	var results []map[string]interface{}
	for _, userVehicle := range userVehicles {
		vehicleData := map[string]interface{}{
			"vehicle":         userVehicle.Vehicle,
			"latest_status":   nil, // For status data (ignition, voltage, signal, etc.)
			"latest_location": nil, // For location data (lat, lng, speed)
			"access_info":     userVehicle.GetAccessInfo(),
			"today_km":        0.0,
			"today_fuel":      0.0,
			"total_odometer":  userVehicle.Vehicle.Odometer,
			"last_update":     nil,
			"since_duration":  nil,
		}

		imei := userVehicle.Vehicle.IMEI

		// 1. Fetch latest status data with non-null status fields
		var statusData *models.GPSData
		statusQuery := `
			SELECT * FROM gps_data 
			WHERE imei = ? 
			AND (voltage_level IS NOT NULL OR gsm_signal IS NOT NULL OR ignition != '' OR charger != '' OR oil_electricity != '')
			ORDER BY timestamp DESC 
			LIMIT 10`

		var statusCandidates []models.GPSData
		if err := db.GetDB().Raw(statusQuery, imei).Scan(&statusCandidates).Error; err == nil {
			for _, candidate := range statusCandidates {
				if candidate.VoltageLevel != nil || candidate.GSMSignal != nil ||
					candidate.Ignition != "" || candidate.Charger != "" || candidate.OilElectricity != "" {
					statusData = &candidate
					break
				}
			}
		}

		// 2. Fetch latest location data with non-null location fields
		var locationData *models.GPSData
		locationQuery := `
			SELECT * FROM gps_data 
			WHERE imei = ? 
			AND latitude IS NOT NULL AND longitude IS NOT NULL
			ORDER BY timestamp DESC 
			LIMIT 10`

		var locationCandidates []models.GPSData
		if err := db.GetDB().Raw(locationQuery, imei).Scan(&locationCandidates).Error; err == nil {
			for _, candidate := range locationCandidates {
				if candidate.Latitude != nil && candidate.Longitude != nil {
					locationData = &candidate
					break
				}
			}
		}

		// 3. Calculate today's travel distance and fuel consumption
		today := time.Now().Format("2006-01-02")
		tomorrowStart := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

		var todayGPSData []models.GPSData
		if err := db.GetDB().Where("imei = ? AND timestamp >= ? AND timestamp < ? AND latitude IS NOT NULL AND longitude IS NOT NULL AND speed IS NOT NULL",
			imei, today, tomorrowStart).Order("timestamp ASC").Find(&todayGPSData).Error; err == nil {

			var totalDistance float64
			if len(todayGPSData) > 1 {
				for i := 0; i < len(todayGPSData)-1; i++ {
					p1 := todayGPSData[i]
					p2 := todayGPSData[i+1]
					if p1.Latitude != nil && p1.Longitude != nil && p2.Latitude != nil && p2.Longitude != nil {
						distance := utils.CalculateDistance(*p1.Latitude, *p1.Longitude, *p2.Latitude, *p2.Longitude)
						totalDistance += distance
					}
				}
			}

			vehicleData["today_km"] = totalDistance

			// Calculate fuel consumption
			if userVehicle.Vehicle.Mileage > 0 {
				vehicleData["today_fuel"] = totalDistance / userVehicle.Vehicle.Mileage
			}
		}

		// 4. Calculate total odometer by adding today's distance to base odometer
		vehicleData["total_odometer"] = userVehicle.Vehicle.Odometer + vehicleData["today_km"].(float64)

		// 5. Determine last update and since duration
		var mostRecentData *models.GPSData
		if statusData != nil && locationData != nil {
			if statusData.Timestamp.After(locationData.Timestamp) {
				mostRecentData = statusData
			} else {
				mostRecentData = locationData
			}
		} else if statusData != nil {
			mostRecentData = statusData
		} else if locationData != nil {
			mostRecentData = locationData
		}

		if mostRecentData != nil {
			vehicleData["last_update"] = mostRecentData.Timestamp
			sinceDuration := time.Since(mostRecentData.Timestamp)
			vehicleData["since_duration"] = sinceDuration.String()
		}

		// Add the status and location data to response
		if statusData != nil {
			vehicleData["latest_status"] = statusData
		}
		if locationData != nil {
			vehicleData["latest_location"] = locationData
		}

		results = append(results, vehicleData)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
		"count":   len(results),
		"message": "User vehicles retrieved successfully",
	})
}

// GetMyVehicle returns a specific vehicle accessible to the current user
func (vc *VehicleController) GetMyVehicle(c *gin.Context) {
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

	// Check if user has access to this vehicle
	var userVehicle models.UserVehicle
	if err := db.GetDB().Where("user_id = ? AND vehicle_id = ? AND is_active = ?", user.ID, imei, true).
		Preload("Vehicle").First(&userVehicle).Error; err != nil {
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

	// Get all users with access to this vehicle (if user is main user or has share_tracking permission)
	var users []map[string]interface{}
	if userVehicle.IsMainUser || userVehicle.ShareTracking {
		var allUserVehicles []models.UserVehicle
		if err := db.GetDB().Where("vehicle_id = ? AND is_active = ?", imei, true).
			Preload("User").Find(&allUserVehicles).Error; err == nil {
			for _, uv := range allUserVehicles {
				if uv.IsExpired() {
					continue
				}
				userInfo := map[string]interface{}{
					"id":           uv.User.ID,
					"name":         uv.User.Name,
					"email":        uv.User.Email,
					"role":         uv.GetUserRole(),
					"permissions":  uv.GetPermissions(),
					"is_main_user": uv.IsMainUser,
					"access_id":    uv.ID,
				}
				users = append(users, userInfo)
			}
		}
	}

	response := gin.H{
		"success": true,
		"data": map[string]interface{}{
			"imei":         userVehicle.Vehicle.IMEI,
			"reg_no":       userVehicle.Vehicle.RegNo,
			"name":         userVehicle.Vehicle.Name,
			"vehicle_type": userVehicle.Vehicle.VehicleType,
			"odometer":     userVehicle.Vehicle.Odometer,
			"mileage":      userVehicle.Vehicle.Mileage,
			"min_fuel":     userVehicle.Vehicle.MinFuel,
			"overspeed":    userVehicle.Vehicle.Overspeed,
			"created_at":   userVehicle.Vehicle.CreatedAt,
			"updated_at":   userVehicle.Vehicle.UpdatedAt,
			"device":       userVehicle.Vehicle.Device,
			"user_role":    userVehicle.GetUserRole(),
			"permissions":  userVehicle.GetPermissions(),
			"is_main_user": userVehicle.IsMainUser,
			"users":        users,
		},
		"message": "Vehicle retrieved successfully",
	}

	c.JSON(http.StatusOK, response)
}

// CreateMyVehicle creates a new vehicle for the current user
func (vc *VehicleController) CreateMyVehicle(c *gin.Context) {
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}
	user := currentUser.(*models.User)

	var vehicle models.Vehicle
	if err := c.ShouldBindJSON(&vehicle); err != nil {
		colors.PrintError("Invalid JSON in vehicle creation request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	colors.PrintInfo("Creating vehicle with IMEI: %s, RegNo: %s, Type: %s, User: %d",
		vehicle.IMEI, vehicle.RegNo, vehicle.VehicleType, user.ID)

	// Validate IMEI length
	if len(vehicle.IMEI) != 16 {
		colors.PrintWarning("Invalid IMEI length: %d (expected 16)", len(vehicle.IMEI))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "IMEI must be exactly 16 digits",
		})
		return
	}

	// Validate IMEI contains only digits
	for _, char := range vehicle.IMEI {
		if char < '0' || char > '9' {
			colors.PrintWarning("Invalid IMEI format: contains non-digit characters")
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "IMEI must contain only digits",
			})
			return
		}
	}

	// Validate vehicle type
	validTypes := []models.VehicleType{
		models.VehicleTypeBike,
		models.VehicleTypeCar,
		models.VehicleTypeTruck,
		models.VehicleTypeBus,
		models.VehicleTypeSchoolBus,
	}

	isValidType := false
	for _, validType := range validTypes {
		if vehicle.VehicleType == validType {
			isValidType = true
			break
		}
	}

	if !isValidType {
		colors.PrintWarning("Invalid vehicle type: %s", vehicle.VehicleType)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":     false,
			"error":       "Invalid vehicle type",
			"valid_types": []string{"bike", "car", "truck", "bus", "school_bus"},
		})
		return
	}

	// Check if device exists
	var device models.Device
	if err := db.GetDB().Where("imei = ?", vehicle.IMEI).First(&device).Error; err != nil {
		colors.PrintWarning("Device with IMEI %s not found in database", vehicle.IMEI)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Device with this IMEI does not exist",
			"hint":    "Please contact admin to register the device first",
		})
		return
	}

	// Check if vehicle with same registration number already exists
	var existingVehicle models.Vehicle
	if err := db.GetDB().Where("reg_no = ?", vehicle.RegNo).First(&existingVehicle).Error; err == nil {
		colors.PrintWarning("Vehicle with registration number %s already exists", vehicle.RegNo)
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error":   "Vehicle with this registration number already exists",
		})
		return
	}

	// Check if this IMEI is already assigned to another vehicle
	if err := db.GetDB().Where("imei = ?", vehicle.IMEI).First(&existingVehicle).Error; err == nil {
		colors.PrintWarning("IMEI %s is already assigned to vehicle %s", vehicle.IMEI, existingVehicle.RegNo)
		c.JSON(http.StatusConflict, gin.H{
			"success":          false,
			"error":            "This device is already assigned to another vehicle",
			"existing_vehicle": existingVehicle.RegNo,
		})
		return
	}

	// Start transaction
	tx := db.GetDB().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create the vehicle
	if err := tx.Create(&vehicle).Error; err != nil {
		tx.Rollback()
		colors.PrintError("Failed to create vehicle in database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create vehicle",
			"details": err.Error(),
		})
		return
	}

	// Create main user assignment for current user
	mainUserAssignment := &models.UserVehicle{
		UserID:        user.ID,
		VehicleID:     vehicle.IMEI,
		AllAccess:     true,
		LiveTracking:  true,
		History:       true,
		Report:        true,
		VehicleEdit:   true,
		Notification:  true,
		ShareTracking: true,
		IsMainUser:    true,
		GrantedBy:     user.ID,
		GrantedAt:     time.Now(),
		IsActive:      true,
		Notes:         "Main user (Vehicle Owner)",
	}

	colors.PrintInfo("Creating user-vehicle assignment: UserID=%d, VehicleID=%s, GrantedBy=%d",
		mainUserAssignment.UserID, mainUserAssignment.VehicleID, mainUserAssignment.GrantedBy)

	if err := tx.Create(mainUserAssignment).Error; err != nil {
		tx.Rollback()
		colors.PrintError("Failed to assign main user to vehicle: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to assign main user to vehicle",
			"details": err.Error(),
		})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		colors.PrintError("Failed to commit vehicle creation transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to complete vehicle creation",
		})
		return
	}

	// Load device information
	db.GetDB().Where("imei = ?", vehicle.IMEI).First(&device)
	vehicle.Device = device

	colors.PrintSuccess("Vehicle created successfully: IMEI=%s, RegNo=%s, User=%s",
		vehicle.IMEI, vehicle.RegNo, user.Email)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"imei":         vehicle.IMEI,
			"reg_no":       vehicle.RegNo,
			"name":         vehicle.Name,
			"vehicle_type": vehicle.VehicleType,
			"odometer":     vehicle.Odometer,
			"mileage":      vehicle.Mileage,
			"min_fuel":     vehicle.MinFuel,
			"overspeed":    vehicle.Overspeed,
			"created_at":   vehicle.CreatedAt,
			"updated_at":   vehicle.UpdatedAt,
			"device":       vehicle.Device,
			"user_role":    "Main User",
			"permissions":  []string{"all_access", "live_tracking", "history", "report", "vehicle_edit", "notification", "share_tracking"},
			"is_main_user": true,
		},
		"message": "Vehicle created successfully",
	})
}

// UpdateMyVehicle updates a vehicle owned by the current user
func (vc *VehicleController) UpdateMyVehicle(c *gin.Context) {
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

	// Check if user has edit permission for this vehicle
	var userVehicle models.UserVehicle
	if err := db.GetDB().Where("user_id = ? AND vehicle_id = ? AND is_active = ?", user.ID, imei, true).
		First(&userVehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Vehicle not found or access denied",
		})
		return
	}

	if userVehicle.IsExpired() || (!userVehicle.VehicleEdit && !userVehicle.AllAccess) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "You don't have permission to edit this vehicle",
		})
		return
	}

	var vehicle models.Vehicle
	if err := db.GetDB().Where("imei = ?", imei).First(&vehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Vehicle not found",
		})
		return
	}

	var updateData models.Vehicle
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Don't allow IMEI or registration number updates
	updateData.IMEI = vehicle.IMEI
	updateData.RegNo = vehicle.RegNo

	if err := db.GetDB().Model(&vehicle).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update vehicle",
		})
		return
	}

	// Load device information
	var device models.Device
	if err := db.GetDB().Where("imei = ?", vehicle.IMEI).First(&device).Error; err == nil {
		vehicle.Device = device
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"imei":         vehicle.IMEI,
			"reg_no":       vehicle.RegNo,
			"name":         vehicle.Name,
			"vehicle_type": vehicle.VehicleType,
			"odometer":     vehicle.Odometer,
			"mileage":      vehicle.Mileage,
			"min_fuel":     vehicle.MinFuel,
			"overspeed":    vehicle.Overspeed,
			"created_at":   vehicle.CreatedAt,
			"updated_at":   vehicle.UpdatedAt,
			"device":       vehicle.Device,
			"user_role":    userVehicle.GetUserRole(),
			"permissions":  userVehicle.GetPermissions(),
			"is_main_user": userVehicle.IsMainUser,
		},
		"message": "Vehicle updated successfully",
	})
}

// DeleteMyVehicle deletes a vehicle owned by the current user (only main users can delete)
func (vc *VehicleController) DeleteMyVehicle(c *gin.Context) {
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

	// Check if user is the main user of this vehicle
	var userVehicle models.UserVehicle
	if err := db.GetDB().Where("user_id = ? AND vehicle_id = ? AND is_main_user = ? AND is_active = ?",
		user.ID, imei, true, true).First(&userVehicle).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Only the main user can delete a vehicle",
		})
		return
	}

	if userVehicle.IsExpired() {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Vehicle access has expired",
		})
		return
	}

	var vehicle models.Vehicle
	if err := db.GetDB().Where("imei = ?", imei).First(&vehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Vehicle not found",
		})
		return
	}

	// Start transaction to delete vehicle and all related user access
	tx := db.GetDB().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Delete all user-vehicle relationships first
	if err := tx.Where("vehicle_id = ?", imei).Delete(&models.UserVehicle{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete vehicle access records",
		})
		return
	}

	// Delete the vehicle
	if err := tx.Delete(&vehicle).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete vehicle",
		})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to complete vehicle deletion",
		})
		return
	}

	colors.PrintSuccess("Vehicle deleted successfully: IMEI=%s, RegNo=%s, User=%s",
		vehicle.IMEI, vehicle.RegNo, user.Email)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Vehicle deleted successfully",
	})
}

// GetVehicleShares returns sharing information for a vehicle
func (vc *VehicleController) GetVehicleShares(c *gin.Context) {
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

	// Check if user has access to this vehicle and share_tracking permission
	var userVehicle models.UserVehicle
	if err := db.GetDB().Where("user_id = ? AND vehicle_id = ? AND is_active = ?", user.ID, imei, true).
		First(&userVehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Vehicle not found or access denied",
		})
		return
	}

	if userVehicle.IsExpired() || (!userVehicle.ShareTracking && !userVehicle.AllAccess && !userVehicle.IsMainUser) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "You don't have permission to view vehicle sharing information",
		})
		return
	}

	// Get all users with access to this vehicle
	var allUserVehicles []models.UserVehicle
	if err := db.GetDB().Where("vehicle_id = ? AND is_active = ?", imei, true).
		Preload("User").Preload("GrantedByUser").Find(&allUserVehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch vehicle sharing information",
		})
		return
	}

	var shares []map[string]interface{}
	for _, uv := range allUserVehicles {
		if uv.IsExpired() {
			continue
		}
		shareInfo := map[string]interface{}{
			"access_id":    uv.ID,
			"user_id":      uv.User.ID,
			"user_name":    uv.User.Name,
			"user_email":   uv.User.Email,
			"role":         uv.GetUserRole(),
			"permissions":  uv.GetPermissions(),
			"is_main_user": uv.IsMainUser,
			"granted_at":   uv.GrantedAt,
			"expires_at":   uv.ExpiresAt,
			"notes":        uv.Notes,
			"granted_by":   uv.GrantedByUser.Name,
		}
		shares = append(shares, shareInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    shares,
		"count":   len(shares),
		"message": "Vehicle sharing information retrieved successfully",
	})
}

// ShareMyVehicle shares a vehicle with another user
func (vc *VehicleController) ShareMyVehicle(c *gin.Context) {
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

	// Check if user is main user or has share_tracking permission
	var userVehicle models.UserVehicle
	if err := db.GetDB().Where("user_id = ? AND vehicle_id = ? AND is_active = ?", user.ID, imei, true).
		First(&userVehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Vehicle not found or access denied",
		})
		return
	}

	if userVehicle.IsExpired() || (!userVehicle.ShareTracking && !userVehicle.AllAccess && !userVehicle.IsMainUser) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "You don't have permission to share this vehicle",
		})
		return
	}

	var req struct {
		UserID      uint            `json:"user_id" binding:"required"`
		Permissions map[string]bool `json:"permissions" binding:"required"`
		ExpiresAt   *time.Time      `json:"expires_at,omitempty"`
		Notes       string          `json:"notes,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Verify the target user exists
	var targetUser models.User
	if err := db.GetDB().First(&targetUser, req.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Target user not found",
		})
		return
	}

	// Check if target user already has access
	var existingAccess models.UserVehicle
	if err := db.GetDB().Where("user_id = ? AND vehicle_id = ?", req.UserID, imei).First(&existingAccess).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error":   "User already has access to this vehicle",
		})
		return
	}

	// Create new user-vehicle relationship
	newUserVehicle := models.UserVehicle{
		UserID:        req.UserID,
		VehicleID:     imei,
		AllAccess:     req.Permissions["all_access"],
		LiveTracking:  req.Permissions["live_tracking"],
		History:       req.Permissions["history"],
		Report:        req.Permissions["report"],
		VehicleEdit:   req.Permissions["vehicle_edit"],
		Notification:  req.Permissions["notification"],
		ShareTracking: req.Permissions["share_tracking"],
		IsMainUser:    false, // Shared users are never main users
		GrantedBy:     user.ID,
		GrantedAt:     time.Now(),
		ExpiresAt:     req.ExpiresAt,
		IsActive:      true,
		Notes:         req.Notes,
	}

	if err := db.GetDB().Create(&newUserVehicle).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to share vehicle access",
		})
		return
	}

	// Load relationships
	db.GetDB().Preload("User").Preload("GrantedByUser").First(&newUserVehicle, newUserVehicle.ID)

	colors.PrintSuccess("Vehicle %s shared with user %s by user %s", imei, targetUser.Email, user.Email)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"access_id":    newUserVehicle.ID,
			"user_id":      newUserVehicle.User.ID,
			"user_name":    newUserVehicle.User.Name,
			"user_email":   newUserVehicle.User.Email,
			"role":         newUserVehicle.GetUserRole(),
			"permissions":  newUserVehicle.GetPermissions(),
			"is_main_user": newUserVehicle.IsMainUser,
			"granted_at":   newUserVehicle.GrantedAt,
			"expires_at":   newUserVehicle.ExpiresAt,
			"notes":        newUserVehicle.Notes,
			"granted_by":   newUserVehicle.GrantedByUser.Name,
		},
		"message": "Vehicle shared successfully",
	})
}

// RevokeVehicleShare revokes access to a shared vehicle
func (vc *VehicleController) RevokeVehicleShare(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid IMEI format",
		})
		return
	}

	shareId, err := strconv.ParseUint(c.Param("shareId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid share ID",
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

	// Check if user is main user or has share_tracking permission
	var userVehicle models.UserVehicle
	if err := db.GetDB().Where("user_id = ? AND vehicle_id = ? AND is_active = ?", user.ID, imei, true).
		First(&userVehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Vehicle not found or access denied",
		})
		return
	}

	if userVehicle.IsExpired() || (!userVehicle.ShareTracking && !userVehicle.AllAccess && !userVehicle.IsMainUser) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "You don't have permission to revoke vehicle access",
		})
		return
	}

	// Find the share to revoke
	var shareToRevoke models.UserVehicle
	if err := db.GetDB().Where("id = ? AND vehicle_id = ?", uint(shareId), imei).First(&shareToRevoke).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Share not found",
		})
		return
	}

	// Don't allow revoking main user access
	if shareToRevoke.IsMainUser {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Cannot revoke main user access",
		})
		return
	}

	// Delete the share
	if err := db.GetDB().Delete(&shareToRevoke).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to revoke access",
		})
		return
	}

	colors.PrintSuccess("Vehicle access revoked: IMEI=%s, ShareID=%d, RevokedBy=%s", imei, shareId, user.Email)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Vehicle access revoked successfully",
	})
}

// ForceDeleteVehiclesBackupData permanently deletes all soft-deleted vehicles
func (vc *VehicleController) ForceDeleteVehiclesBackupData(c *gin.Context) {
	gormDB := db.GetDB()

	// Count records to be deleted for confirmation
	var deletedVehicles int64
	gormDB.Unscoped().Model(&models.Vehicle{}).Where("deleted_at IS NOT NULL").Count(&deletedVehicles)

	if deletedVehicles == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success":       true,
			"message":       "No deleted vehicle backup data found to force delete",
			"deleted_count": 0,
		})
		return
	}

	// Perform the permanent deletion
	result := gormDB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.Vehicle{})
	if result.Error != nil {
		colors.PrintError("Failed to force delete vehicles: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to force delete vehicle backup data"})
		return
	}

	colors.PrintSuccess("Force deleted %d vehicles permanently", deletedVehicles)

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "Vehicle backup data has been permanently removed",
		"deleted_count": deletedVehicles,
	})
}
