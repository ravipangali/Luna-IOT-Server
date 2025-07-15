package controllers

import (
	"net/http"
	"strconv"

	"luna_iot_server/config"
	"luna_iot_server/internal/services"
	"luna_iot_server/pkg/colors"

	"github.com/gin-gonic/gin"
)

type NotificationController struct {
	notificationService *services.NotificationService
}

func NewNotificationController() *NotificationController {
	return &NotificationController{
		notificationService: services.NewNotificationService(),
	}
}

// SendNotificationRequest represents the request body for sending notifications
type SendNotificationRequest struct {
	UserIDs  []uint                 `json:"user_ids" binding:"required"`
	Title    string                 `json:"title" binding:"required"`
	Body     string                 `json:"body" binding:"required"`
	Data     map[string]interface{} `json:"data,omitempty"`
	ImageURL string                 `json:"image_url,omitempty"`
	Sound    string                 `json:"sound,omitempty"`
	Priority string                 `json:"priority,omitempty"`
	Type     string                 `json:"type,omitempty"`
}

// SendToTopicRequest represents the request body for sending notifications to topics
type SendToTopicRequest struct {
	Topic    string                 `json:"topic" binding:"required"`
	Title    string                 `json:"title" binding:"required"`
	Body     string                 `json:"body" binding:"required"`
	Data     map[string]interface{} `json:"data,omitempty"`
	ImageURL string                 `json:"image_url,omitempty"`
	Sound    string                 `json:"sound,omitempty"`
	Priority string                 `json:"priority,omitempty"`
	Type     string                 `json:"type,omitempty"`
}

// UpdateFCMTokenRequest represents the request body for updating FCM token
type UpdateFCMTokenRequest struct {
	FCMToken string `json:"fcm_token" binding:"required"`
}

// SendNotification sends notification to specific users
func (nc *NotificationController) SendNotification(c *gin.Context) {
	var req SendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request body",
			"error":   err.Error(),
		})
		return
	}

	notification := &services.NotificationData{
		Type:     req.Type,
		Title:    req.Title,
		Body:     req.Body,
		Data:     req.Data,
		ImageURL: req.ImageURL,
		Sound:    req.Sound,
		Priority: req.Priority,
	}

	response, err := nc.notificationService.SendToMultipleUsers(req.UserIDs, notification)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to send notification",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": response.Success,
		"message": response.Message,
		"error":   response.Error,
	})
}

// SendToTopic sends notification to a topic
func (nc *NotificationController) SendToTopic(c *gin.Context) {
	var req SendToTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request body",
			"error":   err.Error(),
		})
		return
	}

	notification := &services.NotificationData{
		Type:     req.Type,
		Title:    req.Title,
		Body:     req.Body,
		Data:     req.Data,
		ImageURL: req.ImageURL,
		Sound:    req.Sound,
		Priority: req.Priority,
	}

	response, err := nc.notificationService.SendToTopic(req.Topic, notification)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to send notification",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": response.Success,
		"message": response.Message,
		"error":   response.Error,
	})
}

// UpdateFCMToken updates user's FCM token
func (nc *NotificationController) UpdateFCMToken(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "User not authenticated",
		})
		return
	}
	userID := userIDInterface.(uint)

	var req UpdateFCMTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request body",
			"error":   err.Error(),
		})
		return
	}

	err := nc.notificationService.UpdateUserFCMToken(userID, req.FCMToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to update FCM token",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "FCM token updated successfully",
	})
}

// RemoveFCMToken removes user's FCM token
func (nc *NotificationController) RemoveFCMToken(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "User not authenticated",
		})
		return
	}
	userID := userIDInterface.(uint)

	err := nc.notificationService.RemoveUserFCMToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to remove FCM token",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "FCM token removed successfully",
	})
}

// SubscribeToTopic subscribes user to a topic
func (nc *NotificationController) SubscribeToTopic(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "User not authenticated",
		})
		return
	}
	userID := userIDInterface.(uint)

	topic := c.Param("topic")
	if topic == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Topic is required",
		})
		return
	}

	err := nc.notificationService.SubscribeToTopic(userID, topic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to subscribe to topic",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Subscribed to topic successfully",
	})
}

// UnsubscribeFromTopic unsubscribes user from a topic
func (nc *NotificationController) UnsubscribeFromTopic(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "User not authenticated",
		})
		return
	}
	userID := userIDInterface.(uint)

	topic := c.Param("topic")
	if topic == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Topic is required",
		})
		return
	}

	err := nc.notificationService.UnsubscribeFromTopic(userID, topic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to unsubscribe from topic",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Unsubscribed from topic successfully",
	})
}

// SendToUser sends notification to a specific user (admin only)
func (nc *NotificationController) SendToUser(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid user ID",
		})
		return
	}

	var req struct {
		Title    string                 `json:"title" binding:"required"`
		Body     string                 `json:"body" binding:"required"`
		Data     map[string]interface{} `json:"data,omitempty"`
		ImageURL string                 `json:"image_url,omitempty"`
		Sound    string                 `json:"sound,omitempty"`
		Priority string                 `json:"priority,omitempty"`
		Type     string                 `json:"type,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request body",
			"error":   err.Error(),
		})
		return
	}

	notification := &services.NotificationData{
		Type:     req.Type,
		Title:    req.Title,
		Body:     req.Body,
		Data:     req.Data,
		ImageURL: req.ImageURL,
		Sound:    req.Sound,
		Priority: req.Priority,
	}

	response, err := nc.notificationService.SendToUser(uint(userID), notification)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to send notification",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": response.Success,
		"message": response.Message,
		"error":   response.Error,
	})
}

// TestFirebaseConnection tests Firebase configuration
func (nc *NotificationController) TestFirebaseConnection(c *gin.Context) {
	colors.PrintInfo("Testing Firebase connection...")

	// Test if Firebase is enabled
	if !config.IsFirebaseEnabled() {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Firebase is not enabled",
			"details": map[string]interface{}{
				"firebase_enabled": false,
				"messaging_client": config.GetMessagingClient() != nil,
			},
		})
		return
	}

	// Test Firebase connection
	err := config.TestFirebaseConnection()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Firebase connection test failed",
			"error":   err.Error(),
			"details": map[string]interface{}{
				"firebase_enabled": true,
				"messaging_client": config.GetMessagingClient() != nil,
				"connection_error": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Firebase connection test passed",
		"details": map[string]interface{}{
			"firebase_enabled":  true,
			"messaging_client":  config.GetMessagingClient() != nil,
			"connection_status": "OK",
		},
	})
}
