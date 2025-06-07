package controllers

import (
	"net/http"
	"strconv"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"

	"github.com/gin-gonic/gin"
)

// DeviceController handles device-related HTTP requests
type DeviceController struct{}

// NewDeviceController creates a new device controller
func NewDeviceController() *DeviceController {
	return &DeviceController{}
}

// GetDevices returns all devices with their associated vehicles
func (dc *DeviceController) GetDevices(c *gin.Context) {
	var devices []models.Device

	if err := db.GetDB().Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch devices",
		})
		return
	}

	// Manually load vehicles for each device
	for i := range devices {
		var vehicles []models.Vehicle
		db.GetDB().Where("imei = ?", devices[i].IMEI).Find(&vehicles)
		// Add vehicles data to response (we'll include this in a separate field)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    devices,
		"count":   len(devices),
		"message": "Devices retrieved successfully",
	})
}

// GetDevice returns a single device by ID
func (dc *DeviceController) GetDevice(c *gin.Context) {
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

	c.JSON(http.StatusOK, gin.H{
		"data":    device,
		"message": "Device retrieved successfully",
	})
}

// GetDeviceByIMEI returns a device by IMEI
func (dc *DeviceController) GetDeviceByIMEI(c *gin.Context) {
	imei := c.Param("imei")
	if len(imei) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid IMEI format",
		})
		return
	}

	var device models.Device
	if err := db.GetDB().Where("imei = ?", imei).First(&device).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Device not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    device,
		"message": "Device retrieved successfully",
	})
}

// CreateDevice creates a new device
func (dc *DeviceController) CreateDevice(c *gin.Context) {
	var device models.Device

	if err := c.ShouldBindJSON(&device); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
		})
		return
	}

	// Validate IMEI length
	if len(device.IMEI) != 15 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "IMEI must be exactly 15 digits",
		})
		return
	}

	// Check if device with same IMEI already exists
	var existingDevice models.Device
	if err := db.GetDB().Where("imei = ?", device.IMEI).First(&existingDevice).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Device with this IMEI already exists",
		})
		return
	}

	if err := db.GetDB().Create(&device).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create device",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":    device,
		"message": "Device created successfully",
	})
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
