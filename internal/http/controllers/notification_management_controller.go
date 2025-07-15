package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

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

	notifications, total, err := nmc.notificationDBService.GetNotifications(page, limit)
	if err != nil {
		colors.PrintError("Failed to get notifications: %v", err)
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

	notification, err := nmc.notificationDBService.GetNotificationByID(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "Notification not found",
			})
			return
		}
		colors.PrintError("Failed to get notification: %v", err)
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

	// Create notification request for database service
	dbReq := &services.CreateNotificationRequest{
		Title:     req.Title,
		Body:      req.Body,
		Type:      req.Type,
		ImageURL:  req.ImageURL,
		Sound:     req.Sound,
		Priority:  req.Priority,
		Data:      req.Data,
		UserIDs:   req.UserIDs,
		CreatedBy: userID.(uint),
	}

	// Save notification to database
	response, err := nmc.notificationDBService.CreateNotification(dbReq)
	if err != nil {
		colors.PrintError("Failed to create notification: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": response.Message,
			"error":   response.Error,
		})
		return
	}

	// Send notification immediately if requested
	if req.SendImmediately {
		colors.PrintInfo("Sending notification immediately for ID: %d", response.Data.ID)

		// Convert data JSON to map for sending
		var data map[string]interface{}
		if response.Data.Data != "" {
			if err := json.Unmarshal([]byte(response.Data.Data), &data); err != nil {
				colors.PrintWarning("Failed to unmarshal notification data: %v", err)
				data = make(map[string]interface{})
			}
		}

		notificationData := &services.NotificationData{
			Type:     response.Data.Type,
			Title:    response.Data.Title,
			Body:     response.Data.Body,
			Data:     data,
			ImageURL: response.Data.ImageURL,
			Sound:    response.Data.Sound,
			Priority: response.Data.Priority,
		}

		sendResponse, sendErr := nmc.notificationService.SendToMultipleUsers(req.UserIDs, notificationData)
		if sendErr != nil {
			colors.PrintError("Failed to send notification: %v", sendErr)
			c.JSON(http.StatusOK, gin.H{
				"success":    true,
				"message":    "Notification created but failed to send immediately",
				"data":       response.Data,
				"send_error": sendErr.Error(),
			})
			return
		}

		if sendResponse.Success {
			// Mark notification as sent in database
			if markErr := nmc.notificationDBService.MarkNotificationAsSent(response.Data.ID); markErr != nil {
				colors.PrintWarning("Failed to mark notification as sent: %v", markErr)
			}

			c.JSON(http.StatusOK, gin.H{
				"success":  true,
				"message":  "Notification created and sent successfully",
				"data":     response.Data,
				"response": sendResponse,
			})
			return
		} else {
			c.JSON(http.StatusOK, gin.H{
				"success":    true,
				"message":    "Notification created but failed to send",
				"data":       response.Data,
				"send_error": sendResponse.Message,
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
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

	// Update notification request for database service
	dbReq := &services.UpdateNotificationRequest{
		Title:    req.Title,
		Body:     req.Body,
		Type:     req.Type,
		ImageURL: req.ImageURL,
		Sound:    req.Sound,
		Priority: req.Priority,
		Data:     req.Data,
		UserIDs:  req.UserIDs,
	}

	// Update notification in database
	response, err := nmc.notificationDBService.UpdateNotification(uint(id), dbReq)
	if err != nil {
		colors.PrintError("Failed to update notification: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": response.Message,
			"error":   response.Error,
		})
		return
	}

	// Send notification immediately if requested
	if req.SendImmediately {
		colors.PrintInfo("Sending updated notification immediately for ID: %d", response.Data.ID)

		// Convert data JSON to map for sending
		var data map[string]interface{}
		if response.Data.Data != "" {
			if err := json.Unmarshal([]byte(response.Data.Data), &data); err != nil {
				colors.PrintWarning("Failed to unmarshal notification data: %v", err)
				data = make(map[string]interface{})
			}
		}

		notificationData := &services.NotificationData{
			Type:     response.Data.Type,
			Title:    response.Data.Title,
			Body:     response.Data.Body,
			Data:     data,
			ImageURL: response.Data.ImageURL,
			Sound:    response.Data.Sound,
			Priority: response.Data.Priority,
		}

		sendResponse, sendErr := nmc.notificationService.SendToMultipleUsers(req.UserIDs, notificationData)
		if sendErr != nil {
			colors.PrintError("Failed to send updated notification: %v", sendErr)
			c.JSON(http.StatusOK, gin.H{
				"success":    true,
				"message":    "Notification updated but failed to send immediately",
				"data":       response.Data,
				"send_error": sendErr.Error(),
			})
			return
		}

		if sendResponse.Success {
			// Mark notification as sent in database
			if markErr := nmc.notificationDBService.MarkNotificationAsSent(response.Data.ID); markErr != nil {
				colors.PrintWarning("Failed to mark updated notification as sent: %v", markErr)
			}

			c.JSON(http.StatusOK, gin.H{
				"success":  true,
				"message":  "Notification updated and sent successfully",
				"data":     response.Data,
				"response": sendResponse,
			})
			return
		} else {
			c.JSON(http.StatusOK, gin.H{
				"success":    true,
				"message":    "Notification updated but failed to send",
				"data":       response.Data,
				"send_error": sendResponse.Message,
			})
			return
		}
	}

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
			"message": "Invalid notification ID",
		})
		return
	}

	err = nmc.notificationDBService.DeleteNotification(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "Notification not found",
			})
			return
		}
		colors.PrintError("Failed to delete notification: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to delete notification",
			"error":   err.Error(),
		})
		return
	}

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

	// Get notification from database
	notification, err := nmc.notificationDBService.GetNotificationByID(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "Notification not found",
			})
			return
		}
		colors.PrintError("Failed to get notification: %v", err)
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
			colors.PrintWarning("Failed to unmarshal notification data: %v", err)
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
		colors.PrintError("Failed to send notification: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to send notification",
			"error":   err.Error(),
		})
		return
	}

	if response.Success {
		// Mark notification as sent in database
		if markErr := nmc.notificationDBService.MarkNotificationAsSent(notification.ID); markErr != nil {
			colors.PrintWarning("Failed to mark notification as sent: %v", markErr)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "Notification sent successfully",
		"response": response,
	})
}
