package controllers

import (
	"net/http"

	"luna_iot_server/internal/services"

	"github.com/gin-gonic/gin"
)

type TestNotificationController struct {
	notificationService *services.NotificationService
}

func NewTestNotificationController() *TestNotificationController {
	return &TestNotificationController{
		notificationService: services.NewNotificationService(),
	}
}

// TestNotificationRequest represents the request body for testing notifications
type TestNotificationRequest struct {
	Title string                 `json:"title" binding:"required"`
	Body  string                 `json:"body" binding:"required"`
	Data  map[string]interface{} `json:"data,omitempty"`
}

// SendTestNotification sends a test notification to the current user
func (tnc *TestNotificationController) SendTestNotification(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "User not authenticated",
		})
		return
	}
	userID := userIDInterface.(uint)

	var req TestNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request body",
			"error":   err.Error(),
		})
		return
	}

	// Send notification to the current user
	response, err := tnc.notificationService.SendToUser(userID, &services.NotificationData{
		Type:  "test_notification",
		Title: req.Title,
		Body:  req.Body,
		Data:  req.Data,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to send notification",
			"error":   err.Error(),
		})
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": response.Message,
		})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": response.Message,
		})
	}
}

// SendTestTopicNotification sends a test notification to a topic
func (tnc *TestNotificationController) SendTestTopicNotification(c *gin.Context) {
	topic := c.Param("topic")
	if topic == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Topic parameter is required",
		})
		return
	}

	var req TestNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request body",
			"error":   err.Error(),
		})
		return
	}

	// Send notification to the topic
	response, err := tnc.notificationService.SendToTopic(topic, &services.NotificationData{
		Type:  "test_notification",
		Title: req.Title,
		Body:  req.Body,
		Data:  req.Data,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to send notification",
			"error":   err.Error(),
		})
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": response.Message,
		})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": response.Message,
		})
	}
}
