package controllers

import (
	"net/http"
	"strconv"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/internal/services"
	"luna_iot_server/pkg/colors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NotificationManagementController struct {
	notificationService   *services.NotificationService
	notificationDBService *services.NotificationDBService
}

func NewNotificationManagementController() *NotificationManagementController {
	return &NotificationManagementController{
		notificationService:   services.NewNotificationService(),
		notificationDBService: services.NewNotificationDBService(),
	}
}

// CreateNotificationRequest represents the request for creating a notification
type CreateNotificationRequest struct {
	Title           string                 `json:"title" binding:"required"`
	Body            string                 `json:"body" binding:"required"`
	Type            string                 `json:"type"`
	ImageURL        string                 `json:"image_url"`
	ImageData       string                 `json:"image_data"` // File path for uploaded images
	Sound           string                 `json:"sound"`
	Priority        string                 `json:"priority"`
	Data            map[string]interface{} `json:"data"`
	UserIDs         []uint                 `json:"user_ids" binding:"required"`
	SendImmediately bool                   `json:"send_immediately"`
}

// UpdateNotificationRequest represents the request for updating a notification
type UpdateNotificationRequest struct {
	Title           string                 `json:"title" binding:"required"`
	Body            string                 `json:"body" binding:"required"`
	Type            string                 `json:"type"`
	ImageURL        string                 `json:"image_url"`
	ImageData       string                 `json:"image_data"` // File path for uploaded images
	Sound           string                 `json:"sound"`
	Priority        string                 `json:"priority"`
	Data            map[string]interface{} `json:"data"`
	UserIDs         []uint                 `json:"user_ids" binding:"required"`
	SendImmediately bool                   `json:"send_immediately"`
}

// GetNotifications retrieves all notifications with pagination
func (nmc *NotificationManagementController) GetNotifications(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	notifications, total, err := nmc.notificationDBService.GetNotifications(page, limit)
	if err != nil {
		colors.PrintError("Failed to get notifications: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve notifications",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notifications retrieved successfully",
		"data":    notifications,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

// GetNotification retrieves a specific notification by ID
func (nmc *NotificationManagementController) GetNotification(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid notification ID",
			"message": "Please provide a valid notification ID",
		})
		return
	}

	notification, err := nmc.notificationDBService.GetNotificationByID(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Notification not found",
				"message": "The requested notification does not exist",
			})
			return
		}
		colors.PrintError("Failed to get notification %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve notification",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notification retrieved successfully",
		"data":    notification,
	})
}

// CreateNotification creates a new notification
func (nmc *NotificationManagementController) CreateNotification(c *gin.Context) {
	var req CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		colors.PrintError("Invalid create notification request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	// Get current user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Unauthorized",
			"message": "User not authenticated",
		})
		return
	}
	user := userInterface.(*models.User)

	// Create notification request for database service
	dbReq := &services.CreateNotificationRequest{
		Title:     req.Title,
		Body:      req.Body,
		Type:      req.Type,
		ImageURL:  req.ImageData, // Use uploaded file URL as image_url for display
		ImageData: req.ImageData, // Also store in image_data
		Sound:     req.Sound,
		Priority:  req.Priority,
		Data:      req.Data,
		UserIDs:   req.UserIDs,
		CreatedBy: user.ID,
	}

	// Debug logging
	colors.PrintInfo("Creating notification with ImageURL: %s, ImageData: %s", dbReq.ImageURL, dbReq.ImageData)

	// Save notification to database
	response, err := nmc.notificationDBService.CreateNotification(dbReq)
	if err != nil {
		colors.PrintError("Failed to create notification: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create notification",
			"message": err.Error(),
		})
		return
	}

	if !response.Success {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   response.Error,
			"message": response.Message,
		})
		return
	}

	// If send immediately is requested, send the notification
	if req.SendImmediately {
		// Determine which image URL to use
		imageURL := req.ImageURL
		if req.ImageData != "" {
			imageURL = req.ImageData // Use uploaded image URL
		}

		notificationData := &services.NotificationData{
			Type:     req.Type,
			Title:    req.Title,
			Body:     req.Body,
			Data:     req.Data,
			ImageURL: imageURL,
			Sound:    req.Sound,
			Priority: req.Priority,
		}

		sendResponse, err := nmc.notificationService.SendToMultipleUsers(req.UserIDs, notificationData)
		if err != nil {
			colors.PrintError("Failed to send notification: %v", err)
			// Don't fail the request, just log the error
		} else if sendResponse.Success {
			// Mark notification as sent
			if err := nmc.notificationDBService.MarkNotificationAsSent(response.Data.ID); err != nil {
				colors.PrintError("Failed to mark notification as sent: %v", err)
			}
		}
	}

	colors.PrintSuccess("Notification created successfully with ID: %d", response.Data.ID)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Notification created successfully",
		"data":    response.Data,
	})
}

// UpdateNotification updates an existing notification
func (nmc *NotificationManagementController) UpdateNotification(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid notification ID",
			"message": "Please provide a valid notification ID",
		})
		return
	}

	var req UpdateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		colors.PrintError("Invalid update notification request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	// Create update request for database service
	dbReq := &services.UpdateNotificationRequest{
		Title:     req.Title,
		Body:      req.Body,
		Type:      req.Type,
		ImageURL:  req.ImageData, // Use uploaded file URL as image_url for display
		ImageData: req.ImageData, // Also store in image_data
		Sound:     req.Sound,
		Priority:  req.Priority,
		Data:      req.Data,
		UserIDs:   req.UserIDs,
	}

	// Update notification in database
	response, err := nmc.notificationDBService.UpdateNotification(uint(id), dbReq)
	if err != nil {
		colors.PrintError("Failed to update notification %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update notification",
			"message": err.Error(),
		})
		return
	}

	if !response.Success {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   response.Error,
			"message": response.Message,
		})
		return
	}

	// If send immediately is requested, send the notification
	if req.SendImmediately {
		// Determine which image URL to use
		imageURL := req.ImageURL
		if req.ImageData != "" {
			imageURL = req.ImageData // Use uploaded image URL
		}

		notificationData := &services.NotificationData{
			Type:     req.Type,
			Title:    req.Title,
			Body:     req.Body,
			Data:     req.Data,
			ImageURL: imageURL,
			Sound:    req.Sound,
			Priority: req.Priority,
		}

		sendResponse, err := nmc.notificationService.SendToMultipleUsers(req.UserIDs, notificationData)
		if err != nil {
			colors.PrintError("Failed to send notification: %v", err)
			// Don't fail the request, just log the error
		} else if sendResponse.Success {
			// Mark notification as sent
			if err := nmc.notificationDBService.MarkNotificationAsSent(response.Data.ID); err != nil {
				colors.PrintError("Failed to mark notification as sent: %v", err)
			}
		}
	}

	colors.PrintSuccess("Notification updated successfully with ID: %d", response.Data.ID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notification updated successfully",
		"data":    response.Data,
	})
}

// DeleteNotification deletes a notification
func (nmc *NotificationManagementController) DeleteNotification(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid notification ID",
			"message": "Please provide a valid notification ID",
		})
		return
	}

	if err := nmc.notificationDBService.DeleteNotification(uint(id)); err != nil {
		colors.PrintError("Failed to delete notification %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete notification",
			"message": err.Error(),
		})
		return
	}

	colors.PrintSuccess("Notification deleted successfully with ID: %d", id)
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
		colors.PrintError("Invalid notification ID format: %s", idStr)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid notification ID",
			"message": "Please provide a valid notification ID",
		})
		return
	}

	colors.PrintInfo("Starting to send notification with ID: %d", id)

	// Get notification from database
	colors.PrintInfo("Fetching notification %d from database", id)
	notification, err := nmc.notificationDBService.GetNotificationByID(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			colors.PrintError("Notification %d not found in database", id)
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Notification not found",
				"message": "The requested notification does not exist",
			})
			return
		}
		colors.PrintError("Database error getting notification %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve notification",
			"message": err.Error(),
		})
		return
	}

	colors.PrintInfo("Successfully retrieved notification %d: Title='%s', Body='%s'",
		id, notification.Title, notification.Body)

	// Get user IDs for this notification
	colors.PrintInfo("Fetching user IDs for notification %d", id)
	var users []models.User
	if err := db.GetDB().Model(&notification).Association("Users").Find(&users); err != nil {
		colors.PrintError("Failed to get users for notification %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get notification users",
			"message": err.Error(),
		})
		return
	}

	// Extract user IDs from the users slice
	var userIDs []uint
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}

	colors.PrintInfo("Found %d users for notification %d: %v", len(userIDs), id, userIDs)

	if len(userIDs) == 0 {
		colors.PrintWarning("No users found for notification %d", id)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No users assigned",
			"message": "No users are assigned to this notification",
		})
		return
	}

	// Prepare notification data
	notificationData := &services.NotificationData{
		Type:     notification.Type,
		Title:    notification.Title,
		Body:     notification.Body,
		Data:     notification.GetDataMap(),
		ImageURL: notification.ImageData, // Use image_data as primary image URL
		Sound:    notification.Sound,
		Priority: notification.Priority,
	}

	// If image_data is not available, fallback to image_url
	if notification.ImageData == "" {
		notificationData.ImageURL = notification.ImageURL
	}

	colors.PrintInfo("Prepared notification data: Title='%s', Body='%s', Type='%s'",
		notificationData.Title, notificationData.Body, notificationData.Type)

	// Send notification
	colors.PrintInfo("Sending notification %d to %d users", id, len(userIDs))
	sendResponse, err := nmc.notificationService.SendToMultipleUsers(userIDs, notificationData)
	if err != nil {
		colors.PrintError("Failed to send notification %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to send notification",
			"message": err.Error(),
		})
		return
	}

	colors.PrintInfo("Notification send response: %+v", sendResponse)
	if !sendResponse.Success {
		colors.PrintError("Notification send failed: %s", sendResponse.Message)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   sendResponse.Error,
			"message": sendResponse.Message,
		})
		return
	}

	// Mark notification as sent
	colors.PrintInfo("Marking notification %d as sent", id)
	if err := nmc.notificationDBService.MarkNotificationAsSent(uint(id)); err != nil {
		colors.PrintError("Failed to mark notification as sent: %v", err)
	}

	colors.PrintSuccess("Notification sent successfully with ID: %d", id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notification sent successfully",
		"data":    sendResponse,
	})
}
