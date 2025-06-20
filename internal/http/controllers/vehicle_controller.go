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
