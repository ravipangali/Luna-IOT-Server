package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DeviceModelController handles device model related HTTP requests
type DeviceModelController struct{}

// NewDeviceModelController creates a new device model controller
func NewDeviceModelController() *DeviceModelController {
	return &DeviceModelController{}
}

// GetDeviceModels returns all device models
func (dmc *DeviceModelController) GetDeviceModels(c *gin.Context) {
	var deviceModels []models.DeviceModel

	// Check if devices should be included
	includeDevices := c.Query("include_devices") == "true"

	query := db.GetDB()
	if includeDevices {
		query = query.Preload("Devices")
	}

	if err := query.Find(&deviceModels).Error; err != nil {
		colors.PrintError("Failed to fetch device models: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch device models",
			"message": "Unable to retrieve device models from database",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    deviceModels,
		"count":   len(deviceModels),
		"message": "Device models retrieved successfully",
	})
}

// GetDeviceModel returns a single device model by ID
func (dmc *DeviceModelController) GetDeviceModel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid device model ID",
			"message": "Device model ID must be a valid number",
		})
		return
	}

	var deviceModel models.DeviceModel
	query := db.GetDB()

	// Check if devices should be included
	if c.Query("include_devices") == "true" {
		query = query.Preload("Devices")
	}

	if err := query.First(&deviceModel, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Device model not found",
				"message": "No device model found with the specified ID",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Database error",
				"message": "Failed to retrieve device model from database",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    deviceModel,
		"message": "Device model retrieved successfully",
	})
}

// CreateDeviceModel creates a new device model
func (dmc *DeviceModelController) CreateDeviceModel(c *gin.Context) {
	var deviceModel models.DeviceModel

	colors.PrintInfo("📥 Received device model creation request from %s", c.ClientIP())

	if err := c.ShouldBindJSON(&deviceModel); err != nil {
		colors.PrintError("❌ JSON binding failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid JSON format in request body",
			"message": "Please check your JSON syntax and required fields",
			"details": err.Error(),
		})
		return
	}

	colors.PrintInfo("📋 Creating device model: Name=%s", deviceModel.Name)

	// Validate required fields
	if strings.TrimSpace(deviceModel.Name) == "" {
		colors.PrintWarning("⚠️ Missing device model name")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Device model name is required",
			"message": "Please provide a valid device model name",
		})
		return
	}

	// Check if device model with this name already exists
	var existingModel models.DeviceModel
	if err := db.GetDB().Where("name = ?", deviceModel.Name).First(&existingModel).Error; err == nil {
		colors.PrintWarning("⚠️ Device model with name %s already exists (ID: %d)", deviceModel.Name, existingModel.ID)
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error":   "Device model with this name already exists",
			"message": "A device model with this name already exists",
			"existing_model": gin.H{
				"id":         existingModel.ID,
				"name":       existingModel.Name,
				"created_at": existingModel.CreatedAt,
			},
		})
		return
	}

	// Save the device model to the database
	colors.PrintInfo("💾 Attempting to save device model to database...")
	if err := db.GetDB().Create(&deviceModel).Error; err != nil {
		colors.PrintError("❌ Database error while creating device model: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create device model",
			"message": "Database error occurred while creating device model",
			"details": err.Error(),
		})
		return
	}

	colors.PrintSuccess("✅ Device model created successfully: ID=%d, Name=%s", deviceModel.ID, deviceModel.Name)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    deviceModel,
		"message": "Device model created successfully",
	})
}

// UpdateDeviceModel updates an existing device model
func (dmc *DeviceModelController) UpdateDeviceModel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid device model ID",
			"message": "Device model ID must be a valid number",
		})
		return
	}

	var deviceModel models.DeviceModel
	if err := db.GetDB().First(&deviceModel, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Device model not found",
				"message": "No device model found with the specified ID",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Database error",
				"message": "Failed to retrieve device model from database",
			})
		}
		return
	}

	var updateData models.DeviceModel
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data",
			"message": "Please check your JSON syntax and required fields",
			"details": err.Error(),
		})
		return
	}

	colors.PrintInfo("📝 Updating device model: ID=%d, Name=%s", deviceModel.ID, updateData.Name)

	// Validate name if it's being changed
	if strings.TrimSpace(updateData.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Device model name is required",
			"message": "Please provide a valid device model name",
		})
		return
	}

	// Check if another device model with this name already exists
	if updateData.Name != deviceModel.Name {
		var existingModel models.DeviceModel
		if err := db.GetDB().Where("name = ? AND id != ?", updateData.Name, deviceModel.ID).First(&existingModel).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error":   "Device model name already exists",
				"message": "A device model with this name already exists",
			})
			return
		}
	}

	if err := db.GetDB().Model(&deviceModel).Updates(updateData).Error; err != nil {
		colors.PrintError("❌ Failed to update device model in database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update device model",
			"message": "Database error occurred while updating device model",
		})
		return
	}

	colors.PrintSuccess("✅ Device model updated successfully: ID=%d", deviceModel.ID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    deviceModel,
		"message": "Device model updated successfully",
	})
}

// DeleteDeviceModel deletes a device model
func (dmc *DeviceModelController) DeleteDeviceModel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid device model ID",
			"message": "Device model ID must be a valid number",
		})
		return
	}

	var deviceModel models.DeviceModel
	if err := db.GetDB().First(&deviceModel, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Device model not found",
				"message": "No device model found with the specified ID",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Database error",
				"message": "Failed to retrieve device model from database",
			})
		}
		return
	}

	// Check if device model has associated devices
	var deviceCount int64
	db.GetDB().Model(&models.Device{}).Where("model_id = ?", deviceModel.ID).Count(&deviceCount)
	if deviceCount > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"success":      false,
			"error":        "Cannot delete device model with associated devices",
			"message":      "This device model is currently being used by devices and cannot be deleted",
			"device_count": deviceCount,
		})
		return
	}

	if err := db.GetDB().Delete(&deviceModel).Error; err != nil {
		colors.PrintError("❌ Failed to delete device model: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete device model",
			"message": "Database error occurred while deleting device model",
		})
		return
	}

	colors.PrintSuccess("✅ Device model deleted successfully: ID=%d, Name=%s", deviceModel.ID, deviceModel.Name)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Device model deleted successfully",
	})
}

// ForceDeleteDeviceModelsBackupData permanently deletes all soft-deleted device models
func (dmc *DeviceModelController) ForceDeleteDeviceModelsBackupData(c *gin.Context) {
	gormDB := db.GetDB()

	// Count records to be deleted for confirmation
	var deletedDeviceModels int64
	gormDB.Unscoped().Model(&models.DeviceModel{}).Where("deleted_at IS NOT NULL").Count(&deletedDeviceModels)

	if deletedDeviceModels == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success":       true,
			"message":       "No deleted device model backup data found to force delete",
			"deleted_count": 0,
		})
		return
	}

	// Perform the permanent deletion
	result := gormDB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.DeviceModel{})
	if result.Error != nil {
		colors.PrintError("Failed to force delete device models: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to force delete device model backup data"})
		return
	}

	colors.PrintSuccess("Force deleted %d device models permanently", deletedDeviceModels)

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "Device model backup data has been permanently removed",
		"deleted_count": deletedDeviceModels,
	})
}
