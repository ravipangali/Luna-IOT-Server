package controllers

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DeviceController handles device-related HTTP requests
type DeviceController struct{}

// NewDeviceController creates a new device controller
func NewDeviceController() *DeviceController {
	return &DeviceController{}
}

// Enhanced error response structure
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
	Code    string            `json:"code,omitempty"`
}

// Success response structure
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Count   int         `json:"count,omitempty"`
}

// Helper function to create error response
func (dc *DeviceController) createErrorResponse(c *gin.Context, statusCode int, errorCode string, message string, details map[string]string) {
	c.JSON(statusCode, ErrorResponse{
		Error:   errorCode,
		Message: message,
		Details: details,
		Code:    errorCode,
	})
}

// Helper function to create success response
func (dc *DeviceController) createSuccessResponse(c *gin.Context, statusCode int, message string, data interface{}, count int) {
	response := SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	if count > 0 {
		response.Count = count
	}
	c.JSON(statusCode, response)
}

// GetDevices returns all devices with their associated vehicles
func (dc *DeviceController) GetDevices(c *gin.Context) {
	var devices []models.Device

	query := db.GetDB().Preload("Model")

	if err := query.Find(&devices).Error; err != nil {
		dc.createErrorResponse(c, http.StatusInternalServerError, "DATABASE_ERROR",
			"Unable to retrieve devices from database",
			map[string]string{
				"database_error": err.Error(),
				"suggestion":     "Please check database connection and try again",
			})
		return
	}

	// Manually load vehicles for each device
	for i := range devices {
		var vehicles []models.Vehicle
		db.GetDB().Where("imei = ?", devices[i].IMEI).Find(&vehicles)
		// Add vehicles data to response (we'll include this in a separate field)
	}

	dc.createSuccessResponse(c, http.StatusOK, "Devices retrieved successfully", devices, len(devices))
}

// GetDevice returns a single device by ID
func (dc *DeviceController) GetDevice(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		dc.createErrorResponse(c, http.StatusBadRequest, "INVALID_DEVICE_ID",
			"Device ID must be a valid number",
			map[string]string{
				"provided_id":     c.Param("id"),
				"expected_format": "Numeric ID (e.g., 1, 2, 3)",
				"suggestion":      "Please provide a valid numeric device ID",
			})
		return
	}

	var device models.Device
	if err := db.GetDB().Preload("Model").First(&device, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dc.createErrorResponse(c, http.StatusNotFound, "DEVICE_NOT_FOUND",
				"No device found with the specified ID",
				map[string]string{
					"device_id":  strconv.FormatUint(id, 10),
					"suggestion": "Please verify the device ID and try again",
				})
		} else {
			dc.createErrorResponse(c, http.StatusInternalServerError, "DATABASE_ERROR",
				"Failed to retrieve device from database",
				map[string]string{
					"database_error": err.Error(),
					"device_id":      strconv.FormatUint(id, 10),
				})
		}
		return
	}

	dc.createSuccessResponse(c, http.StatusOK, "Device retrieved successfully", device, 0)
}

// GetDeviceByIMEI returns a device by IMEI
func (dc *DeviceController) GetDeviceByIMEI(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		dc.createErrorResponse(c, http.StatusBadRequest, "INVALID_IMEI_FORMAT",
			"IMEI must be exactly 16 digits",
			map[string]string{
				"provided_imei":   imei,
				"provided_length": strconv.Itoa(len(imei)),
				"expected_length": "16",
				"expected_format": "16 numeric digits (e.g., 1234567890123456)",
				"suggestion":      "Please provide a valid 16-digit IMEI number",
			})
		return
	}

	// Validate IMEI contains only digits
	for _, char := range imei {
		if char < '0' || char > '9' {
			dc.createErrorResponse(c, http.StatusBadRequest, "INVALID_IMEI_CHARACTERS",
				"IMEI must contain only numeric digits",
				map[string]string{
					"provided_imei":     imei,
					"invalid_character": string(char),
					"expected_format":   "16 numeric digits only",
					"suggestion":        "Please ensure IMEI contains only numbers 0-9",
				})
			return
		}
	}

	var device models.Device
	if err := db.GetDB().Preload("Model").Where("imei = ?", imei).First(&device).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dc.createErrorResponse(c, http.StatusNotFound, "DEVICE_NOT_FOUND",
				"No device found with the specified IMEI",
				map[string]string{
					"imei":       imei,
					"suggestion": "Please verify the IMEI number and ensure the device is registered",
				})
		} else {
			dc.createErrorResponse(c, http.StatusInternalServerError, "DATABASE_ERROR",
				"Failed to retrieve device from database",
				map[string]string{
					"database_error": err.Error(),
					"imei":           imei,
				})
		}
		return
	}

	dc.createSuccessResponse(c, http.StatusOK, "Device retrieved successfully", device, 0)
}

// CreateDevice creates a new device
func (dc *DeviceController) CreateDevice(c *gin.Context) {
	var device models.Device

	// Log the incoming request
	colors.PrintInfo("📥 Received device creation request from %s", c.ClientIP())

	// Read raw body for debugging
	body, _ := c.GetRawData()
	colors.PrintDebug("📋 Raw request body: %s", string(body))

	// Reset body for binding
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	if err := c.ShouldBindJSON(&device); err != nil {
		colors.PrintError("❌ JSON binding failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid JSON format in request body",
			"details": err.Error(),
			"help":    "Please check your JSON syntax and required fields",
		})
		return
	}

	colors.PrintInfo("📋 Parsed device data: IMEI=%s, SimNo=%s, Operator=%s, Protocol=%s",
		device.IMEI, device.SimNo, device.SimOperator, device.Protocol)

	// Validate IMEI length
	if len(device.IMEI) != 16 {
		colors.PrintWarning("⚠️ Invalid IMEI length: %d (expected 16)", len(device.IMEI))
		c.JSON(http.StatusBadRequest, gin.H{
			"success":         false,
			"error":           "IMEI must be exactly 16 digits",
			"provided_imei":   device.IMEI,
			"provided_length": len(device.IMEI),
		})
		return
	}

	// Validate IMEI is numeric
	for i, char := range device.IMEI {
		if char < '0' || char > '9' {
			colors.PrintWarning("⚠️ Invalid IMEI format: non-numeric character '%c' at position %d", char, i)
			c.JSON(http.StatusBadRequest, gin.H{
				"success":           false,
				"error":             "IMEI must contain only digits",
				"invalid_character": string(char),
				"position":          i,
			})
			return
		}
	}

	// Validate SIM number
	if strings.TrimSpace(device.SimNo) == "" {
		colors.PrintWarning("⚠️ Missing SIM number")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "SIM number is required",
		})
		return
	}

	// Check if device with this IMEI already exists
	var existingDevice models.Device
	if err := db.GetDB().Where("imei = ?", device.IMEI).First(&existingDevice).Error; err == nil {
		colors.PrintWarning("⚠️ Device with IMEI %s already exists (ID: %d)", device.IMEI, existingDevice.ID)
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error":   "Device with this IMEI already exists",
			"existing_device": gin.H{
				"id":         existingDevice.ID,
				"imei":       existingDevice.IMEI,
				"sim_no":     existingDevice.SimNo,
				"created_at": existingDevice.CreatedAt,
			},
		})
		return
	}

	// Check if SIM number already exists
	var existingSim models.Device
	if err := db.GetDB().Where("sim_no = ?", device.SimNo).First(&existingSim).Error; err == nil {
		colors.PrintWarning("⚠️ Device with SIM number %s already exists (IMEI: %s)", device.SimNo, existingSim.IMEI)
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error":   "Device with this SIM number already exists",
			"existing_device": gin.H{
				"imei":   existingSim.IMEI,
				"sim_no": existingSim.SimNo,
			},
		})
		return
	}

	// Validate SIM operator
	if device.SimOperator != models.SimOperatorNcell && device.SimOperator != models.SimOperatorNtc {
		colors.PrintWarning("⚠️ Invalid SIM operator: %s", device.SimOperator)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":         false,
			"error":           "Invalid SIM operator",
			"provided":        string(device.SimOperator),
			"valid_operators": []string{string(models.SimOperatorNcell), string(models.SimOperatorNtc)},
		})
		return
	}

	// Set default protocol if not provided
	if device.Protocol == "" {
		device.Protocol = models.ProtocolGT06
		colors.PrintInfo("🔧 Set default protocol to GT06")
	}

	// Validate protocol
	if device.Protocol != models.ProtocolGT06 {
		colors.PrintWarning("⚠️ Invalid protocol: %s", device.Protocol)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":             false,
			"error":               "Invalid protocol",
			"provided":            string(device.Protocol),
			"supported_protocols": []string{string(models.ProtocolGT06)},
		})
		return
	}

	// Validate device model if provided
	if device.ModelID != nil {
		var deviceModel models.DeviceModel
		if err := db.GetDB().First(&deviceModel, *device.ModelID).Error; err != nil {
			colors.PrintWarning("⚠️ Invalid device model ID: %d", *device.ModelID)
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid device model",
				"message": "The specified device model does not exist",
			})
			return
		}
		colors.PrintInfo("🔧 Device model: %s (ID: %d)", deviceModel.Name, deviceModel.ID)
	}

	// Test database connection before saving
	if db.GetDB() == nil {
		colors.PrintError("❌ Database connection is nil")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection unavailable",
			"message": "Please try again later or contact administrator",
		})
		return
	}

	// Save the device to the database
	colors.PrintInfo("💾 Attempting to save device to database...")
	if err := db.GetDB().Create(&device).Error; err != nil {
		colors.PrintError("❌ Database error while creating device: %v", err)

		// Check for common database errors and provide user-friendly messages
		errorMessage := "Failed to create device due to a database error"

		if strings.Contains(err.Error(), "duplicate key") {
			if strings.Contains(err.Error(), "devices_imei_key") {
				errorMessage = "A device with this IMEI already exists"
			}
		} else if strings.Contains(err.Error(), "foreign key constraint") {
			errorMessage = "Cannot create device: reference constraint failed"

			if strings.Contains(err.Error(), "fk_vehicles_device") {
				errorMessage = "This IMEI is already associated with a vehicle"
			}
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   errorMessage,
			"details": err.Error(),
		})
		return
	}

	colors.PrintSuccess("✅ Device created successfully: ID=%d, IMEI=%s", device.ID, device.IMEI)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Device created successfully",
		"data":    device,
	})
}

// UpdateDevice updates an existing device
func (dc *DeviceController) UpdateDevice(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid device ID",
		})
		return
	}

	var device models.Device
	if err := db.GetDB().First(&device, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Device not found",
		})
		return
	}

	var updateData models.Device
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data",
		})
		return
	}

	// Don't allow IMEI updates for security reasons
	// If you want to allow IMEI updates, comment out the line below and uncomment the validation section
	updateData.IMEI = device.IMEI

	/*
		// Uncomment this section if you want to allow IMEI updates:
		// Validate new IMEI if it's being changed
		if updateData.IMEI != device.IMEI && updateData.IMEI != "" {
			// Check IMEI format (must be 16 digits)
			if len(updateData.IMEI) != 16 {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error": "IMEI must be exactly 16 digits",
				})
				return
			}

			// Validate IMEI contains only digits
			for _, char := range updateData.IMEI {
				if char < '0' || char > '9' {
					c.JSON(http.StatusBadRequest, gin.H{
						"success": false,
						"error": "IMEI must contain only digits",
					})
					return
				}
			}

			// Check if new IMEI already exists
			var existingDevice models.Device
			if err := db.GetDB().Where("imei = ? AND id != ?", updateData.IMEI, device.ID).First(&existingDevice).Error; err == nil {
				c.JSON(http.StatusConflict, gin.H{
					"success": false,
					"error": "Device with this IMEI already exists",
				})
				return
			}
		}
	*/

	if err := db.GetDB().Model(&device).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update device",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    device,
		"message": "Device updated successfully",
	})
}

// DeleteDevice deletes a device
func (dc *DeviceController) DeleteDevice(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	var device models.Device
	if err := db.GetDB().First(&device, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Device not found",
		})
		return
	}

	// Check if device has associated vehicles
	var vehicleCount int64
	db.GetDB().Model(&models.Vehicle{}).Where("imei = ?", device.IMEI).Count(&vehicleCount)
	if vehicleCount > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Cannot delete device with associated vehicles",
		})
		return
	}

	if err := db.GetDB().Unscoped().Delete(&device).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete device",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Device deleted successfully",
	})
}
