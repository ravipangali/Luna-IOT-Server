package controllers

import (
	"net/http"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/internal/protocol"
	"luna_iot_server/pkg/colors"

	"github.com/gin-gonic/gin"
)

// UserControlController handles user-based vehicle control operations
type UserControlController struct {
	controlController *ControlController
}

// NewUserControlController creates a new user control controller
func NewUserControlController(controlController *ControlController) *UserControlController {
	return &UserControlController{
		controlController: controlController,
	}
}

// UserControlResponse represents the response for user control operations
type UserControlResponse struct {
	Success         bool                      `json:"success"`
	Message         string                    `json:"message"`
	VehicleInfo     map[string]interface{}    `json:"vehicle_info,omitempty"`
	ControlResponse *protocol.ControlResponse `json:"control_response,omitempty"`
	Permissions     []models.Permission       `json:"permissions,omitempty"`
	Error           string                    `json:"error,omitempty"`
}

// validateUserVehicleAccess checks if user has access to vehicle and specific permission
func (ucc *UserControlController) validateUserVehicleAccess(c *gin.Context, imei string, permission models.Permission) (*models.UserVehicle, *UserControlResponse, error) {
	currentUser, exists := c.Get("user")
	if !exists {
		response := &UserControlResponse{
			Success: false,
			Error:   "User not authenticated",
		}
		return nil, response, gin.Error{Err: nil}
	}
	user := currentUser.(*models.User)

	// Check user access to this vehicle
	var userVehicle models.UserVehicle
	if err := db.GetDB().Where("user_id = ? AND vehicle_id = ? AND is_active = ?",
		user.ID, imei, true).Preload("Vehicle").Preload("Vehicle.Device").First(&userVehicle).Error; err != nil {
		response := &UserControlResponse{
			Success: false,
			Error:   "Vehicle not found or access denied",
		}
		return nil, response, err
	}

	if userVehicle.IsExpired() {
		response := &UserControlResponse{
			Success: false,
			Error:   "Vehicle access has expired",
		}
		return nil, response, gin.Error{Err: nil}
	}

	if !userVehicle.HasPermission(permission) && !userVehicle.HasPermission(models.PermissionAllAccess) {
		response := &UserControlResponse{
			Success:     false,
			Error:       "Insufficient permissions for this operation",
			Permissions: userVehicle.GetPermissions(),
		}
		return nil, response, gin.Error{Err: nil}
	}

	return &userVehicle, nil, nil
}

// CutOilAndElectricity cuts oil and electricity for user's vehicle
func (ucc *UserControlController) CutOilAndElectricity(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, UserControlResponse{
			Success: false,
			Error:   "Invalid IMEI format",
		})
		return
	}

	userVehicle, errorResponse, err := ucc.validateUserVehicleAccess(c, imei, models.PermissionVehicleEdit)
	if err != nil || errorResponse != nil {
		statusCode := http.StatusForbidden
		if err != nil {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, errorResponse)
		return
	}

	// Get active connection for this device
	conn, exists := ucc.controlController.GetActiveConnection(imei)
	if !exists {
		c.JSON(http.StatusNotFound, UserControlResponse{
			Success: false,
			Error:   "Device is not currently connected",
			VehicleInfo: map[string]interface{}{
				"imei":         userVehicle.Vehicle.IMEI,
				"reg_no":       userVehicle.Vehicle.RegNo,
				"name":         userVehicle.Vehicle.Name,
				"vehicle_type": userVehicle.Vehicle.VehicleType,
			},
		})
		return
	}

	// Create GPS tracker controller and send command
	controller := protocol.NewGPSTrackerController(conn, imei)
	response, err := controller.CutOilAndElectricity()

	if err != nil {
		colors.PrintError("Failed to cut oil and electricity for IMEI %s: %v", imei, err)
		c.JSON(http.StatusInternalServerError, UserControlResponse{
			Success: false,
			Error:   "Failed to send command to device",
			VehicleInfo: map[string]interface{}{
				"imei":         userVehicle.Vehicle.IMEI,
				"reg_no":       userVehicle.Vehicle.RegNo,
				"name":         userVehicle.Vehicle.Name,
				"vehicle_type": userVehicle.Vehicle.VehicleType,
			},
		})
		return
	}

	colors.PrintSuccess("Oil and electricity cut for vehicle %s (IMEI: %s) by user %s",
		userVehicle.Vehicle.RegNo, imei, c.GetString("user_email"))

	c.JSON(http.StatusOK, UserControlResponse{
		Success: true,
		Message: "Oil and electricity cut command sent successfully",
		VehicleInfo: map[string]interface{}{
			"imei":         userVehicle.Vehicle.IMEI,
			"reg_no":       userVehicle.Vehicle.RegNo,
			"name":         userVehicle.Vehicle.Name,
			"vehicle_type": userVehicle.Vehicle.VehicleType,
		},
		ControlResponse: response,
		Permissions:     userVehicle.GetPermissions(),
	})
}

// ConnectOilAndElectricity connects oil and electricity for user's vehicle
func (ucc *UserControlController) ConnectOilAndElectricity(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, UserControlResponse{
			Success: false,
			Error:   "Invalid IMEI format",
		})
		return
	}

	userVehicle, errorResponse, err := ucc.validateUserVehicleAccess(c, imei, models.PermissionVehicleEdit)
	if err != nil || errorResponse != nil {
		statusCode := http.StatusForbidden
		if err != nil {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, errorResponse)
		return
	}

	// Get active connection for this device
	conn, exists := ucc.controlController.GetActiveConnection(imei)
	if !exists {
		c.JSON(http.StatusNotFound, UserControlResponse{
			Success: false,
			Error:   "Device is not currently connected",
			VehicleInfo: map[string]interface{}{
				"imei":         userVehicle.Vehicle.IMEI,
				"reg_no":       userVehicle.Vehicle.RegNo,
				"name":         userVehicle.Vehicle.Name,
				"vehicle_type": userVehicle.Vehicle.VehicleType,
			},
		})
		return
	}

	// Create GPS tracker controller and send command
	controller := protocol.NewGPSTrackerController(conn, imei)
	response, err := controller.ConnectOilAndElectricity()

	if err != nil {
		colors.PrintError("Failed to connect oil and electricity for IMEI %s: %v", imei, err)
		c.JSON(http.StatusInternalServerError, UserControlResponse{
			Success: false,
			Error:   "Failed to send command to device",
			VehicleInfo: map[string]interface{}{
				"imei":         userVehicle.Vehicle.IMEI,
				"reg_no":       userVehicle.Vehicle.RegNo,
				"name":         userVehicle.Vehicle.Name,
				"vehicle_type": userVehicle.Vehicle.VehicleType,
			},
		})
		return
	}

	colors.PrintSuccess("Oil and electricity connected for vehicle %s (IMEI: %s) by user %s",
		userVehicle.Vehicle.RegNo, imei, c.GetString("user_email"))

	c.JSON(http.StatusOK, UserControlResponse{
		Success: true,
		Message: "Oil and electricity connect command sent successfully",
		VehicleInfo: map[string]interface{}{
			"imei":         userVehicle.Vehicle.IMEI,
			"reg_no":       userVehicle.Vehicle.RegNo,
			"name":         userVehicle.Vehicle.Name,
			"vehicle_type": userVehicle.Vehicle.VehicleType,
		},
		ControlResponse: response,
		Permissions:     userVehicle.GetPermissions(),
	})
}

// GetVehicleLocation requests current location for user's vehicle
func (ucc *UserControlController) GetVehicleLocation(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, UserControlResponse{
			Success: false,
			Error:   "Invalid IMEI format",
		})
		return
	}

	userVehicle, errorResponse, err := ucc.validateUserVehicleAccess(c, imei, models.PermissionLiveTracking)
	if err != nil || errorResponse != nil {
		statusCode := http.StatusForbidden
		if err != nil {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, errorResponse)
		return
	}

	// Get active connection for this device
	conn, exists := ucc.controlController.GetActiveConnection(imei)
	if !exists {
		c.JSON(http.StatusNotFound, UserControlResponse{
			Success: false,
			Error:   "Device is not currently connected",
			VehicleInfo: map[string]interface{}{
				"imei":         userVehicle.Vehicle.IMEI,
				"reg_no":       userVehicle.Vehicle.RegNo,
				"name":         userVehicle.Vehicle.Name,
				"vehicle_type": userVehicle.Vehicle.VehicleType,
			},
		})
		return
	}

	// Create GPS tracker controller and send command
	controller := protocol.NewGPSTrackerController(conn, imei)
	response, err := controller.GetLocation()

	if err != nil {
		colors.PrintError("Failed to get location for IMEI %s: %v", imei, err)
		c.JSON(http.StatusInternalServerError, UserControlResponse{
			Success: false,
			Error:   "Failed to send command to device",
			VehicleInfo: map[string]interface{}{
				"imei":         userVehicle.Vehicle.IMEI,
				"reg_no":       userVehicle.Vehicle.RegNo,
				"name":         userVehicle.Vehicle.Name,
				"vehicle_type": userVehicle.Vehicle.VehicleType,
			},
		})
		return
	}

	colors.PrintInfo("Location requested for vehicle %s (IMEI: %s) by user %s",
		userVehicle.Vehicle.RegNo, imei, c.GetString("user_email"))

	c.JSON(http.StatusOK, UserControlResponse{
		Success: true,
		Message: "Location request command sent successfully",
		VehicleInfo: map[string]interface{}{
			"imei":         userVehicle.Vehicle.IMEI,
			"reg_no":       userVehicle.Vehicle.RegNo,
			"name":         userVehicle.Vehicle.Name,
			"vehicle_type": userVehicle.Vehicle.VehicleType,
		},
		ControlResponse: response,
		Permissions:     userVehicle.GetPermissions(),
	})
}

// GetUserActiveDevices returns list of active devices for user's vehicles
func (ucc *UserControlController) GetUserActiveDevices(c *gin.Context) {
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, UserControlResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}
	user := currentUser.(*models.User)

	// Get user's accessible vehicles
	var userVehicles []models.UserVehicle
	if err := db.GetDB().Where("user_id = ? AND is_active = ?", user.ID, true).
		Preload("Vehicle").Preload("Vehicle.Device").Find(&userVehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, UserControlResponse{
			Success: false,
			Error:   "Failed to fetch user vehicles",
		})
		return
	}

	// Get registered IMEIs from control controller
	registeredIMEIs := ucc.controlController.getRegisteredIMEIs()
	registeredSet := make(map[string]bool)
	for _, imei := range registeredIMEIs {
		registeredSet[imei] = true
	}

	var activeDevices []map[string]interface{}
	var inactiveDevices []map[string]interface{}

	for _, userVehicle := range userVehicles {
		if userVehicle.IsExpired() {
			continue
		}

		deviceInfo := map[string]interface{}{
			"imei":         userVehicle.Vehicle.IMEI,
			"reg_no":       userVehicle.Vehicle.RegNo,
			"name":         userVehicle.Vehicle.Name,
			"vehicle_type": userVehicle.Vehicle.VehicleType,
			"permissions":  userVehicle.GetPermissions(),
			"device":       userVehicle.Vehicle.Device,
		}

		if registeredSet[userVehicle.Vehicle.IMEI] {
			deviceInfo["status"] = "connected"
			activeDevices = append(activeDevices, deviceInfo)
		} else {
			deviceInfo["status"] = "disconnected"
			inactiveDevices = append(inactiveDevices, deviceInfo)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"active_devices":   activeDevices,
			"inactive_devices": inactiveDevices,
			"total_active":     len(activeDevices),
			"total_inactive":   len(inactiveDevices),
			"total_devices":    len(activeDevices) + len(inactiveDevices),
		},
		"message": "User active devices retrieved successfully",
	})
}
