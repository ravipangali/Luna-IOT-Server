package controllers

import (
	"net/http"
	"strconv"
	"time"

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

// SendNotification sends a notification immediately via Ravipangali API
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

	colors.PrintInfo("Starting to send notification with ID: %d via Ravipangali API", id)

	// Use the new backend-driven approach
	sendResponse, err := nmc.notificationService.SendNotificationByID(uint(id))
	if err != nil {
		colors.PrintError("Failed to send notification %d via Ravipangali: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to send notification",
			"message": err.Error(),
		})
		return
	}

	if !sendResponse.Success {
		colors.PrintError("Notification send failed: %s", sendResponse.Message)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   sendResponse.Error,
			"message": sendResponse.Message,
		})
		return
	}

	colors.PrintSuccess("Notification sent successfully with ID: %d via Ravipangali API", id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notification sent successfully via Ravipangali API",
		"data":    sendResponse,
	})
}

// SendNotificationToDevice sends notification directly to device tokens via Ravipangali
func (nmc *NotificationManagementController) SendNotificationToDevice(c *gin.Context) {
	var req struct {
		Title    string                 `json:"title" binding:"required"`
		Body     string                 `json:"body" binding:"required"`
		Tokens   []string               `json:"tokens" binding:"required"`
		ImageURL string                 `json:"image_url,omitempty"`
		Data     map[string]interface{} `json:"data,omitempty"`
		Priority string                 `json:"priority,omitempty"`
		Type     string                 `json:"type,omitempty"`
		Sound    string                 `json:"sound,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		colors.PrintError("Invalid send notification to device request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	// Validate required fields
	if len(req.Tokens) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No tokens provided",
			"message": "At least one FCM token is required",
		})
		return
	}

	// Set default priority if not provided
	if req.Priority == "" {
		req.Priority = "normal"
	}

	// Create Ravipangali service and send notification
	ravipangaliService := services.NewRavipangaliService()
	response, err := ravipangaliService.SendPushNotification(
		req.Title,
		req.Body,
		req.Tokens,
		req.ImageURL,
		req.Data,
		req.Priority,
		req.Type,
		req.Sound,
	)

	if err != nil {
		colors.PrintError("Failed to send notification to devices: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to send notification",
			"message": err.Error(),
		})
		return
	}

	if !response.Success {
		colors.PrintError("Ravipangali API returned failure: %s", response.Error)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   response.Error,
			"message": response.Message,
		})
		return
	}

	colors.PrintSuccess("Notification sent to %d devices via Ravipangali API", len(req.Tokens))
	c.JSON(http.StatusOK, gin.H{
		"success":          response.Success,
		"message":          "Notification sent successfully",
		"notification_id":  response.NotificationID,
		"tokens_sent":      response.TokensSent,
		"tokens_delivered": response.TokensDelivered,
		"tokens_failed":    response.TokensFailed,
		"details":          response.Details,
	})
}

// TestAlarmNotification sends a test alarm notification
func (nmc *NotificationManagementController) TestAlarmNotification(c *gin.Context) {
	var req struct {
		Title  string   `json:"title" binding:"required"`
		Body   string   `json:"body" binding:"required"`
		Tokens []string `json:"tokens" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		colors.PrintError("Invalid test alarm notification request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	// Create Ravipangali service and send alarm notification
	ravipangaliService := services.NewRavipangaliService()
	response, err := ravipangaliService.SendPushNotification(
		req.Title,
		req.Body,
		req.Tokens,
		"", // No image for test
		map[string]interface{}{
			"test_alarm": true,
			"timestamp":  time.Now().Unix(),
		},
		"urgent", // Force urgent priority
		"alarm",  // Force alarm type
		"alarm",  // Force alarm sound
	)

	if err != nil {
		colors.PrintError("Failed to send test alarm notification: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to send alarm notification",
			"message": err.Error(),
		})
		return
	}

	if !response.Success {
		colors.PrintError("Ravipangali API returned failure: %s", response.Error)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   response.Error,
			"message": response.Message,
		})
		return
	}

	colors.PrintSuccess("Test alarm notification sent to %d devices", len(req.Tokens))
	c.JSON(http.StatusOK, gin.H{
		"success":          response.Success,
		"message":          "Test alarm notification sent successfully",
		"notification_id":  response.NotificationID,
		"tokens_sent":      response.TokensSent,
		"tokens_delivered": response.TokensDelivered,
		"tokens_failed":    response.TokensFailed,
		"details":          response.Details,
	})
}
