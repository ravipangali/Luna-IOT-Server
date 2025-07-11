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

type PopupController struct{}

func NewPopupController() *PopupController {
	return &PopupController{}
}

// GetPopups returns all popups
func (pc *PopupController) GetPopups(c *gin.Context) {
	var popups []models.Popup
	query := db.GetDB()

	// by default show active popups
	if c.Query("all") != "true" {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Find(&popups).Error; err != nil {
		colors.PrintError("Failed to fetch popups: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch popups",
			"message": "Unable to retrieve popups from database",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    popups,
		"count":   len(popups),
		"message": "Popups retrieved successfully",
	})
}

// GetPopup returns a single popup by ID
func (pc *PopupController) GetPopup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid popup ID",
			"message": "Popup ID must be a valid number",
		})
		return
	}

	var popup models.Popup
	if err := db.GetDB().First(&popup, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Popup not found",
				"message": "No popup found with the specified ID",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Database error",
				"message": "Failed to retrieve popup from database",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    popup,
		"message": "Popup retrieved successfully",
	})
}

// CreatePopup creates a new popup
func (pc *PopupController) CreatePopup(c *gin.Context) {
	var popup models.Popup

	if err := c.ShouldBindJSON(&popup); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid JSON format in request body",
			"message": "Please check your JSON syntax and required fields",
			"details": err.Error(),
		})
		return
	}

	// Image validation
	if popup.Image != "" {
		if !strings.HasPrefix(popup.Image, "data:image/") {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid image format",
				"message": "Image must be a valid base64 data URL",
			})
			return
		}
		if len(popup.Image) > 7*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Image too large",
				"message": "Image size must be less than 5MB",
			})
			return
		}
		colors.PrintInfo("Popup image included in request (size: %d bytes)", len(popup.Image))
	}

	if err := db.GetDB().Create(&popup).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create popup",
			"message": "Database error occurred while creating popup",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    popup,
		"message": "Popup created successfully",
	})
}

// UpdatePopup updates an existing popup
func (pc *PopupController) UpdatePopup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid popup ID"})
		return
	}

	var existingPopup models.Popup
	if err := db.GetDB().First(&existingPopup, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Popup not found"})
		return
	}

	var updateData models.Popup
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Image validation
	if updateData.Image != "" && updateData.Image != existingPopup.Image {
		if !strings.HasPrefix(updateData.Image, "data:image/") {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid image format",
				"message": "Image must be a valid base64 data URL",
			})
			return
		}
		if len(updateData.Image) > 7*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Image too large",
				"message": "Image size must be less than 5MB",
			})
			return
		}
		colors.PrintInfo("Popup image updated (size: %d bytes)", len(updateData.Image))
	}

	// Use map to update to allow setting boolean to false
	updatePayload := map[string]interface{}{
		"title":     updateData.Title,
		"is_active": updateData.IsActive,
		"image":     updateData.Image,
	}

	if err := db.GetDB().Model(&existingPopup).Updates(updatePayload).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update popup"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": existingPopup, "message": "Popup updated successfully"})
}

// DeletePopup deletes a popup
func (pc *PopupController) DeletePopup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid popup ID"})
		return
	}

	if err := db.GetDB().Unscoped().Delete(&models.Popup{}, uint(id)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete popup"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Popup deleted successfully", "success": true})
}

// GetPopupImage returns a popup's image
func (pc *PopupController) GetPopupImage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid popup ID",
			"message": "Popup ID must be a valid number",
		})
		return
	}

	var popup models.Popup
	if err := db.GetDB().Select("id, image").First(&popup, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Popup not found",
			"message": "No popup found with the specified ID",
		})
		return
	}

	if popup.Image == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Popup has no image",
			"message": "This popup does not have an image",
		})
		return
	}

	// Return the base64 image data
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"image":   popup.Image,
	})
}

// DeletePopupImage removes a popup's image
func (pc *PopupController) DeletePopupImage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid popup ID",
			"message": "Popup ID must be a valid number",
		})
		return
	}

	var popup models.Popup
	if err := db.GetDB().First(&popup, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Popup not found",
			"message": "No popup found with the specified ID",
		})
		return
	}

	// Clear the image field
	if err := db.GetDB().Model(&popup).Update("image", "").Error; err != nil {
		colors.PrintError("Failed to delete popup image: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete popup image",
			"message": "Database error occurred while deleting popup image",
		})
		return
	}

	colors.PrintSuccess("Popup image deleted successfully for popup ID: %d", popup.ID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Popup image deleted successfully",
	})
}

// GetActivePopups returns only active popups for regular users (no admin required)
func (pc *PopupController) GetActivePopups(c *gin.Context) {
	var popups []models.Popup

	if err := db.GetDB().Where("is_active = ?", true).Find(&popups).Error; err != nil {
		colors.PrintError("Failed to fetch active popups: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch active popups",
			"message": "Unable to retrieve active popups from database",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    popups,
		"count":   len(popups),
		"message": "Active popups retrieved successfully",
	})
}
