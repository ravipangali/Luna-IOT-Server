package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"

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

	if err := db.GetDB().Find(&devices).Error; err != nil {
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
	if err := db.GetDB().First(&device, uint(id)).Error; err != nil {
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
	if err := db.GetDB().Where("imei = ?", imei).First(&device).Error; err != nil {
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

	if err := c.ShouldBindJSON(&device); err != nil {
		details := make(map[string]string)
		details["json_error"] = err.Error()
		details["suggestion"] = "Please check the JSON format and required fields"

		// Provide specific field requirements
		details["required_fields"] = "imei, sim_no, sim_operator, protocol"
		details["imei_format"] = "16 numeric digits"
		details["example"] = `{"imei": "1234567890123456", "sim_no": "9841234567", "sim_operator": "Ncell", "protocol": "GT06"}`

		dc.createErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST_DATA",
			"Request data is invalid or malformed", details)
		return
	}

	// Validate IMEI length
	if len(device.IMEI) != 16 {
		dc.createErrorResponse(c, http.StatusBadRequest, "INVALID_IMEI_LENGTH",
			"IMEI must be exactly 16 digits",
			map[string]string{
				"provided_imei":   device.IMEI,
				"provided_length": strconv.Itoa(len(device.IMEI)),
				"expected_length": "16",
				"suggestion":      "Please provide a valid 16-digit IMEI number",
			})
		return
	}

	// Validate IMEI contains only digits
	for _, char := range device.IMEI {
		if char < '0' || char > '9' {
			dc.createErrorResponse(c, http.StatusBadRequest, "INVALID_IMEI_CHARACTERS",
				"IMEI must contain only numeric digits",
				map[string]string{
					"provided_imei":     device.IMEI,
					"invalid_character": string(char),
					"suggestion":        "Please ensure IMEI contains only numbers 0-9",
				})
			return
		}
	}

	// Validate SIM number
	if device.SimNo == "" {
		dc.createErrorResponse(c, http.StatusBadRequest, "MISSING_SIM_NUMBER",
			"SIM number is required",
			map[string]string{
				"field":      "sim_no",
				"suggestion": "Please provide a valid SIM card number",
			})
		return
	}

	// Validate SIM operator
	if device.SimOperator == "" {
		dc.createErrorResponse(c, http.StatusBadRequest, "MISSING_SIM_OPERATOR",
			"SIM operator is required",
			map[string]string{
				"field":           "sim_operator",
				"valid_operators": "Ncell, NTC, Smart Cell, etc.",
				"suggestion":      "Please specify the SIM card operator",
			})
		return
	}

	// Validate protocol
	validProtocols := []models.Protocol{models.ProtocolGT06}
	isValidProtocol := false
	for _, protocol := range validProtocols {
		if device.Protocol == protocol {
			isValidProtocol = true
			break
		}
	}

	if !isValidProtocol {
		validProtocolStrings := make([]string, len(validProtocols))
		for i, p := range validProtocols {
			validProtocolStrings[i] = string(p)
		}

		dc.createErrorResponse(c, http.StatusBadRequest, "INVALID_PROTOCOL",
			"Protocol is not supported",
			map[string]string{
				"provided_protocol":   string(device.Protocol),
				"supported_protocols": strings.Join(validProtocolStrings, ", "),
				"suggestion":          "Please use one of the supported protocols",
			})
		return
	}

	// Check if device with same IMEI already exists
	var existingDevice models.Device
	if err := db.GetDB().Where("imei = ?", device.IMEI).First(&existingDevice).Error; err == nil {
		dc.createErrorResponse(c, http.StatusConflict, "DEVICE_ALREADY_EXISTS",
			"A device with this IMEI already exists in the system",
			map[string]string{
				"imei":               device.IMEI,
				"existing_device_id": strconv.Itoa(int(existingDevice.ID)),
				"suggestion":         "Please use a different IMEI or update the existing device",
			})
		return
	}

	// Check if SIM number already exists
	var existingSimDevice models.Device
	if err := db.GetDB().Where("sim_no = ?", device.SimNo).First(&existingSimDevice).Error; err == nil {
		dc.createErrorResponse(c, http.StatusConflict, "SIM_NUMBER_ALREADY_EXISTS",
			"A device with this SIM number already exists",
			map[string]string{
				"sim_no":               device.SimNo,
				"existing_device_imei": existingSimDevice.IMEI,
				"suggestion":           "Please use a different SIM number or update the existing device",
			})
		return
	}

	if err := db.GetDB().Create(&device).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "UNIQUE constraint") {
			dc.createErrorResponse(c, http.StatusConflict, "DUPLICATE_DEVICE_DATA",
				"Device with similar data already exists",
				map[string]string{
					"database_error": err.Error(),
					"suggestion":     "Please check IMEI and SIM number for duplicates",
				})
		} else {
			dc.createErrorResponse(c, http.StatusInternalServerError, "DATABASE_CREATE_ERROR",
				"Failed to create device in database",
				map[string]string{
					"database_error": err.Error(),
					"suggestion":     "Please try again or contact system administrator",
				})
		}
		return
	}

	dc.createSuccessResponse(c, http.StatusCreated, "Device created successfully", device, 0)
}

// UpdateDevice updates an existing device
func (dc *DeviceController) UpdateDevice(c *gin.Context) {
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

	var updateData models.Device
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
		})
		return
	}

	// Don't allow IMEI updates
	updateData.IMEI = device.IMEI

	if err := db.GetDB().Model(&device).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update device",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
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

	if err := db.GetDB().Delete(&device).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete device",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Device deleted successfully",
	})
}
