package controllers

import (
	"net/http"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"

	"github.com/gin-gonic/gin"
)

// VehicleController handles vehicle-related HTTP requests
type VehicleController struct{}

// NewVehicleController creates a new vehicle controller
func NewVehicleController() *VehicleController {
	return &VehicleController{}
}

// GetVehicles returns all vehicles with their associated devices
func (vc *VehicleController) GetVehicles(c *gin.Context) {
	var vehicles []models.Vehicle

	if err := db.GetDB().Preload("Device").Find(&vehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch vehicles",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    vehicles,
		"count":   len(vehicles),
		"message": "Vehicles retrieved successfully",
	})
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

	c.JSON(http.StatusOK, gin.H{
		"data":    vehicle,
		"message": "Vehicle retrieved successfully",
	})
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
	var vehicle models.Vehicle

	if err := c.ShouldBindJSON(&vehicle); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
		})
		return
	}

	// Validate IMEI length
	if len(vehicle.IMEI) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "IMEI must be exactly 16 digits",
		})
		return
	}

	// Check if device exists
	var device models.Device
	if err := db.GetDB().Where("imei = ?", vehicle.IMEI).First(&device).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device with this IMEI does not exist",
		})
		return
	}

	// Check if vehicle with same registration number already exists
	var existingVehicle models.Vehicle
	if err := db.GetDB().Where("reg_no = ?", vehicle.RegNo).First(&existingVehicle).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Vehicle with this registration number already exists",
		})
		return
	}

	// Check if this IMEI is already assigned to another vehicle
	if err := db.GetDB().Where("imei = ?", vehicle.IMEI).First(&existingVehicle).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "This device is already assigned to another vehicle",
		})
		return
	}

	if err := db.GetDB().Create(&vehicle).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create vehicle",
		})
		return
	}

	// Load the device relationship
	db.GetDB().Preload("Device").Where("imei = ?", vehicle.IMEI).First(&vehicle)

	c.JSON(http.StatusCreated, gin.H{
		"data":    vehicle,
		"message": "Vehicle created successfully",
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

	// Load the device relationship
	db.GetDB().Preload("Device").Where("imei = ?", vehicle.IMEI).First(&vehicle)

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
	if err := db.GetDB().Preload("Device").Where("vehicle_type = ?", vehicleType).Find(&vehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch vehicles",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    vehicles,
		"count":   len(vehicles),
		"message": "Vehicles retrieved successfully",
	})
}
