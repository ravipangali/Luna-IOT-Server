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

// UserVehicleController handles user-vehicle relationship operations
type UserVehicleController struct{}

// NewUserVehicleController creates a new user-vehicle controller
func NewUserVehicleController() *UserVehicleController {
	return &UserVehicleController{}
}

// AssignVehicleRequest represents the request to assign vehicles to a user
type AssignVehicleRequest struct {
	UserID      uint                     `json:"user_id" binding:"required"`
	VehicleID   string                   `json:"vehicle_id" binding:"required,len=16"` // IMEI
	Permissions AssignVehiclePermissions `json:"permissions" binding:"required"`
	ExpiresAt   *time.Time               `json:"expires_at,omitempty"`
	Notes       string                   `json:"notes,omitempty"`
}

// AssignVehiclePermissions represents the permissions structure
type AssignVehiclePermissions struct {
	AllAccess     bool `json:"all_access"`
	LiveTracking  bool `json:"live_tracking"`
	History       bool `json:"history"`
	Report        bool `json:"report"`
	VehicleEdit   bool `json:"vehicle_edit"`
	Notification  bool `json:"notification"`
	ShareTracking bool `json:"share_tracking"`
}

// BulkAssignRequest represents the request to assign multiple vehicles to a user
type BulkAssignRequest struct {
	UserID      uint                     `json:"user_id" binding:"required"`
	VehicleIDs  []string                 `json:"vehicle_ids" binding:"required"`
	Permissions AssignVehiclePermissions `json:"permissions" binding:"required"`
	ExpiresAt   *time.Time               `json:"expires_at,omitempty"`
	Notes       string                   `json:"notes,omitempty"`
}

// AssignVehicleToUser assigns a vehicle to a user with specific permissions
func (uvc *UserVehicleController) AssignVehicleToUser(c *gin.Context) {
	var req AssignVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Get the current user (who is granting access)
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Unauthorized",
		})
		return
	}
	grantedBy := currentUser.(*models.User).ID

	// Verify the target user exists
	var user models.User
	if err := db.GetDB().First(&user, req.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "User not found",
		})
		return
	}

	// Verify the vehicle exists
	var vehicle models.Vehicle
	if err := db.GetDB().Where("imei = ?", req.VehicleID).First(&vehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Vehicle not found",
		})
		return
	}

	// Check if assignment already exists
	var existingAccess models.UserVehicle
	if err := db.GetDB().Where("user_id = ? AND vehicle_id = ?", req.UserID, req.VehicleID).First(&existingAccess).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error":   "User already has access to this vehicle",
			"data":    existingAccess,
		})
		return
	}

	// Create new user-vehicle relationship
	userVehicle := models.UserVehicle{
		UserID:        req.UserID,
		VehicleID:     req.VehicleID,
		AllAccess:     req.Permissions.AllAccess,
		LiveTracking:  req.Permissions.LiveTracking,
		History:       req.Permissions.History,
		Report:        req.Permissions.Report,
		VehicleEdit:   req.Permissions.VehicleEdit,
		Notification:  req.Permissions.Notification,
		ShareTracking: req.Permissions.ShareTracking,
		GrantedBy:     grantedBy,
		GrantedAt:     time.Now(),
		ExpiresAt:     req.ExpiresAt,
		IsActive:      true,
		Notes:         req.Notes,
	}

	if err := db.GetDB().Create(&userVehicle).Error; err != nil {
		colors.PrintError("Failed to assign vehicle to user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to assign vehicle to user",
		})
		return
	}

	// Load relationships
	db.GetDB().Preload("User").Preload("Vehicle").Preload("GrantedByUser").First(&userVehicle, userVehicle.ID)

	colors.PrintSuccess("Vehicle %s assigned to user %s by user %d", req.VehicleID, user.Email, grantedBy)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Vehicle assigned to user successfully",
		"data":    userVehicle,
	})
}

// BulkAssignVehiclesToUser assigns multiple vehicles to a user
func (uvc *UserVehicleController) BulkAssignVehiclesToUser(c *gin.Context) {
	var req BulkAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Get the current user (who is granting access)
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Unauthorized",
		})
		return
	}
	grantedBy := currentUser.(*models.User).ID

	// Verify the target user exists
	var user models.User
	if err := db.GetDB().First(&user, req.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "User not found",
		})
		return
	}

	var results []models.UserVehicle
	var errors []string

	for _, vehicleID := range req.VehicleIDs {
		// Verify the vehicle exists
		var vehicle models.Vehicle
		if err := db.GetDB().Where("imei = ?", vehicleID).First(&vehicle).Error; err != nil {
			errors = append(errors, "Vehicle "+vehicleID+" not found")
			continue
		}

		// Check if assignment already exists
		var existingAccess models.UserVehicle
		if err := db.GetDB().Where("user_id = ? AND vehicle_id = ?", req.UserID, vehicleID).First(&existingAccess).Error; err == nil {
			errors = append(errors, "User already has access to vehicle "+vehicleID)
			continue
		}

		// Create new user-vehicle relationship
		userVehicle := models.UserVehicle{
			UserID:        req.UserID,
			VehicleID:     vehicleID,
			AllAccess:     req.Permissions.AllAccess,
			LiveTracking:  req.Permissions.LiveTracking,
			History:       req.Permissions.History,
			Report:        req.Permissions.Report,
			VehicleEdit:   req.Permissions.VehicleEdit,
			Notification:  req.Permissions.Notification,
			ShareTracking: req.Permissions.ShareTracking,
			GrantedBy:     grantedBy,
			GrantedAt:     time.Now(),
			ExpiresAt:     req.ExpiresAt,
			IsActive:      true,
			Notes:         req.Notes,
		}

		if err := db.GetDB().Create(&userVehicle).Error; err != nil {
			errors = append(errors, "Failed to assign vehicle "+vehicleID+": "+err.Error())
			continue
		}

		// Load relationships
		db.GetDB().Preload("User").Preload("Vehicle").Preload("GrantedByUser").First(&userVehicle, userVehicle.ID)
		results = append(results, userVehicle)
	}

	colors.PrintSuccess("Bulk assigned %d vehicles to user %s", len(results), user.Email)
	c.JSON(http.StatusCreated, gin.H{
		"success":        true,
		"message":        "Bulk vehicle assignment completed",
		"assigned_count": len(results),
		"total_count":    len(req.VehicleIDs),
		"error_count":    len(errors),
		"data":           results,
		"errors":         errors,
	})
}

// UpdateVehiclePermissions updates permissions for a user-vehicle relationship
func (uvc *UserVehicleController) UpdateVehiclePermissions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid access ID",
		})
		return
	}

	var req AssignVehiclePermissions
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	var userVehicle models.UserVehicle
	if err := db.GetDB().First(&userVehicle, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Access record not found",
		})
		return
	}

	// Update permissions
	userVehicle.AllAccess = req.AllAccess
	userVehicle.LiveTracking = req.LiveTracking
	userVehicle.History = req.History
	userVehicle.Report = req.Report
	userVehicle.VehicleEdit = req.VehicleEdit
	userVehicle.Notification = req.Notification
	userVehicle.ShareTracking = req.ShareTracking

	if err := db.GetDB().Save(&userVehicle).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update permissions",
		})
		return
	}

	// Load relationships
	db.GetDB().Preload("User").Preload("Vehicle").Preload("GrantedByUser").First(&userVehicle, userVehicle.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Permissions updated successfully",
		"data":    userVehicle,
	})
}

// RevokeVehicleAccess revokes a user's access to a vehicle
func (uvc *UserVehicleController) RevokeVehicleAccess(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid access ID",
		})
		return
	}

	var userVehicle models.UserVehicle
	if err := db.GetDB().First(&userVehicle, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Access record not found",
		})
		return
	}

	if err := db.GetDB().Delete(&userVehicle).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to revoke access",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Vehicle access revoked successfully",
	})
}

// GetUserVehicleAccess returns all vehicle access for a user
func (uvc *UserVehicleController) GetUserVehicleAccess(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid user ID",
		})
		return
	}

	var accesses []models.UserVehicle
	if err := db.GetDB().Where("user_id = ?", uint(userID)).
		Preload("User").Preload("Vehicle").Preload("GrantedByUser").
		Find(&accesses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch user vehicle access",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    accesses,
		"count":   len(accesses),
		"message": "User vehicle access retrieved successfully",
	})
}

// GetVehicleUserAccess returns all user access for a vehicle
func (uvc *UserVehicleController) GetVehicleUserAccess(c *gin.Context) {
	vehicleID := c.Param("vehicle_id")
	if len(vehicleID) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid vehicle ID (IMEI)",
		})
		return
	}

	var accesses []models.UserVehicle
	if err := db.GetDB().Where("vehicle_id = ?", vehicleID).
		Preload("User").Preload("Vehicle").Preload("GrantedByUser").
		Find(&accesses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch vehicle user access",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    accesses,
		"count":   len(accesses),
		"message": "Vehicle user access retrieved successfully",
	})
}

// SetMainUserRequest represents the request to set a main user
type SetMainUserRequest struct {
	UserAccessID uint `json:"user_access_id" binding:"required"`
}

// SetMainUser sets a user as the main user for a vehicle
func (uvc *UserVehicleController) SetMainUser(c *gin.Context) {
	vehicleID := c.Param("vehicle_id")
	if len(vehicleID) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid vehicle ID (IMEI)",
		})
		return
	}

	var req SetMainUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Start a transaction
	tx := db.GetDB().Begin()

	// First, remove main user status from all users for this vehicle
	if err := tx.Model(&models.UserVehicle{}).Where("vehicle_id = ?", vehicleID).Update("is_main_user", false).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update main user status",
		})
		return
	}

	// Now, set the new main user
	var userVehicle models.UserVehicle
	if err := tx.First(&userVehicle, req.UserAccessID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "User access record not found",
		})
		return
	}

	if userVehicle.VehicleID != vehicleID {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "User access record does not belong to this vehicle",
		})
		return
	}

	userVehicle.IsMainUser = true
	if err := tx.Save(&userVehicle).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to set new main user",
		})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Transaction failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Main user updated successfully",
		"data":    userVehicle,
	})
}
