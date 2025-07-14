package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NotificationManagementController struct {
	notificationService *services.NotificationService
}

func NewNotificationManagementController() *NotificationManagementController {
	return &NotificationManagementController{
		notificationService: services.NewNotificationService(),
	}
}

// CreateNotificationRequest represents the request body for creating notifications
type CreateNotificationRequest struct {
	Title           string                 `json:"title" binding:"required"`
	Body            string                 `json:"body" binding:"required"`
	Type            string                 `json:"type"`
	ImageURL        string                 `json:"image_url"`
	Sound           string                 `json:"sound"`
	Priority        string                 `json:"priority"`
	Data            map[string]interface{} `json:"data"`
	UserIDs         []uint                 `json:"user_ids" binding:"required"`
	SendImmediately bool                   `json:"send_immediately"`
}

// UpdateNotificationRequest represents the request body for updating notifications
type UpdateNotificationRequest struct {
	Title           string                 `json:"title" binding:"required"`
	Body            string                 `json:"body" binding:"required"`
	Type            string                 `json:"type"`
	ImageURL        string                 `json:"image_url"`
	Sound           string                 `json:"sound"`
	Priority        string                 `json:"priority"`
	Data            map[string]interface{} `json:"data"`
	UserIDs         []uint                 `json:"user_ids" binding:"required"`
	SendImmediately bool                   `json:"send_immediately"`
}

// GetNotifications gets all notifications with pagination
func (nmc *NotificationManagementController) GetNotifications(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	var notifications []models.Notification
	var total int64

	database := db.GetDB()

	// Count total
	if err := database.Model(&models.Notification{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to count notifications",
			"error":   err.Error(),
		})
		return
	}

	// Get notifications with creator and users
	if err := database.
		Preload("Creator").
		Preload("Users").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to fetch notifications",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notifications fetched successfully",
		"data":    notifications,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (int(total) + limit - 1) / limit,
		},
	})
}

// GetNotification gets a specific notification by ID
func (nmc *NotificationManagementController) GetNotification(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid notification ID",
		})
		return
	}

	var notification models.Notification
	database := db.GetDB()

	if err := database.
		Preload("Creator").
		Preload("Users").
		First(&notification, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "Notification not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to fetch notification",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notification fetched successfully",
		"data":    notification,
	})
}

// CreateNotification creates a new notification
func (nmc *NotificationManagementController) CreateNotification(c *gin.Context) {
	var req CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request body",
			"error":   err.Error(),
		})
		return
	}

	// Get current user from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "User not authenticated",
		})
		return
	}

	database := db.GetDB()

	// Convert data map to JSON string
	dataJSON := ""
	if req.Data != nil {
		if dataBytes, err := json.Marshal(req.Data); err == nil {
			dataJSON = string(dataBytes)
		}
	}

	// Create notification
	notification := models.Notification{
		Title:     req.Title,
		Body:      req.Body,
		Type:      req.Type,
		ImageURL:  req.ImageURL,
		Sound:     req.Sound,
		Priority:  req.Priority,
		Data:      dataJSON,
		CreatedBy: userID.(uint),
	}

	// Start transaction
	tx := database.Begin()

	// Save notification
	if err := tx.Create(&notification).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create notification",
			"error":   err.Error(),
		})
		return
	}

	// Associate users with notification
	var notificationUsers []models.NotificationUser
	for _, userID := range req.UserIDs {
		notificationUsers = append(notificationUsers, models.NotificationUser{
			NotificationID: notification.ID,
			UserID:         userID,
		})
	}

	if err := tx.Create(&notificationUsers).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to associate users with notification",
			"error":   err.Error(),
		})
		return
	}

	// Send notification immediately if requested
	if req.SendImmediately {
		notificationData := &services.NotificationData{
			Type:     req.Type,
			Title:    req.Title,
			Body:     req.Body,
			Data:     req.Data,
			ImageURL: req.ImageURL,
			Sound:    req.Sound,
			Priority: req.Priority,
		}

		response, err := nmc.notificationService.SendToMultipleUsers(req.UserIDs, notificationData)
		if err != nil {
			// Don't fail the creation, just log the error
			c.JSON(http.StatusOK, gin.H{
				"success":    true,
				"message":    "Notification created but failed to send immediately",
				"data":       notification,
				"send_error": err.Error(),
			})
			tx.Commit()
			return
		}

		if response.Success {
			// Update notification as sent
			now := time.Now()
			notification.IsSent = true
			notification.SentAt = &now
			tx.Save(&notification)

			// Update notification users as sent
			for i := range notificationUsers {
				notificationUsers[i].IsSent = true
				notificationUsers[i].SentAt = &now
			}
			tx.Save(&notificationUsers)
		}
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notification created successfully",
		"data":    notification,
	})
}

// UpdateNotification updates an existing notification
func (nmc *NotificationManagementController) UpdateNotification(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid notification ID",
		})
		return
	}

	var req UpdateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request body",
			"error":   err.Error(),
		})
		return
	}

	database := db.GetDB()

	// Check if notification exists
	var notification models.Notification
	if err := database.First(&notification, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "Notification not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to fetch notification",
			"error":   err.Error(),
		})
		return
	}

	// Convert data map to JSON string
	dataJSON := ""
	if req.Data != nil {
		if dataBytes, err := json.Marshal(req.Data); err == nil {
			dataJSON = string(dataBytes)
		}
	}

	// Start transaction
	tx := database.Begin()

	// Update notification
	notification.Title = req.Title
	notification.Body = req.Body
	notification.Type = req.Type
	notification.ImageURL = req.ImageURL
	notification.Sound = req.Sound
	notification.Priority = req.Priority
	notification.Data = dataJSON

	if err := tx.Save(&notification).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to update notification",
			"error":   err.Error(),
		})
		return
	}

	// Delete existing user associations
	if err := tx.Where("notification_id = ?", notification.ID).Delete(&models.NotificationUser{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to update user associations",
			"error":   err.Error(),
		})
		return
	}

	// Create new user associations
	var notificationUsers []models.NotificationUser
	for _, userID := range req.UserIDs {
		notificationUsers = append(notificationUsers, models.NotificationUser{
			NotificationID: notification.ID,
			UserID:         userID,
		})
	}

	if err := tx.Create(&notificationUsers).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to associate users with notification",
			"error":   err.Error(),
		})
		return
	}

	// Send notification immediately if requested
	if req.SendImmediately {
		notificationData := &services.NotificationData{
			Type:     req.Type,
			Title:    req.Title,
			Body:     req.Body,
			Data:     req.Data,
			ImageURL: req.ImageURL,
			Sound:    req.Sound,
			Priority: req.Priority,
		}

		response, err := nmc.notificationService.SendToMultipleUsers(req.UserIDs, notificationData)
		if err != nil {
			// Don't fail the update, just log the error
			c.JSON(http.StatusOK, gin.H{
				"success":    true,
				"message":    "Notification updated but failed to send immediately",
				"data":       notification,
				"send_error": err.Error(),
			})
			tx.Commit()
			return
		}

		if response.Success {
			// Update notification as sent
			now := time.Now()
			notification.IsSent = true
			notification.SentAt = &now
			tx.Save(&notification)

			// Update notification users as sent
			for i := range notificationUsers {
				notificationUsers[i].IsSent = true
				notificationUsers[i].SentAt = &now
			}
			tx.Save(&notificationUsers)
		}
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notification updated successfully",
		"data":    notification,
	})
}

// DeleteNotification deletes a notification
func (nmc *NotificationManagementController) DeleteNotification(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid notification ID",
		})
		return
	}

	database := db.GetDB()

	// Check if notification exists
	var notification models.Notification
	if err := database.First(&notification, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "Notification not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to fetch notification",
			"error":   err.Error(),
		})
		return
	}

	// Start transaction
	tx := database.Begin()

	// Delete notification users first (due to foreign key constraint)
	if err := tx.Where("notification_id = ?", id).Delete(&models.NotificationUser{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to delete notification associations",
			"error":   err.Error(),
		})
		return
	}

	// Delete notification
	if err := tx.Delete(&notification).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to delete notification",
			"error":   err.Error(),
		})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notification deleted successfully",
	})
}

// SendNotification sends a notification immediately
func (nmc *NotificationManagementController) SendNotification(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid notification ID",
		})
		return
	}

	database := db.GetDB()

	// Get notification with users
	var notification models.Notification
	if err := database.
		Preload("Users").
		First(&notification, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "Notification not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to fetch notification",
			"error":   err.Error(),
		})
		return
	}

	// Get user IDs
	var userIDs []uint
	for _, user := range notification.Users {
		userIDs = append(userIDs, user.ID)
	}

	if len(userIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "No users associated with this notification",
		})
		return
	}

	// Convert data JSON to map
	var data map[string]interface{}
	if notification.Data != "" {
		if err := json.Unmarshal([]byte(notification.Data), &data); err != nil {
			data = make(map[string]interface{})
		}
	}

	// Send notification
	notificationData := &services.NotificationData{
		Type:     notification.Type,
		Title:    notification.Title,
		Body:     notification.Body,
		Data:     data,
		ImageURL: notification.ImageURL,
		Sound:    notification.Sound,
		Priority: notification.Priority,
	}

	response, err := nmc.notificationService.SendToMultipleUsers(userIDs, notificationData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to send notification",
			"error":   err.Error(),
		})
		return
	}

	if response.Success {
		// Update notification as sent
		now := time.Now()
		notification.IsSent = true
		notification.SentAt = &now
		database.Save(&notification)

		// Update notification users as sent
		var notificationUsers []models.NotificationUser
		database.Where("notification_id = ?", notification.ID).Find(&notificationUsers)

		for i := range notificationUsers {
			notificationUsers[i].IsSent = true
			notificationUsers[i].SentAt = &now
		}
		database.Save(&notificationUsers)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "Notification sent successfully",
		"response": response,
	})
}
