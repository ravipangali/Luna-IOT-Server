package controllers

import (
	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/internal/services"
	"luna_iot_server/pkg/colors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TestNotificationController handles test notification requests
type TestNotificationController struct {
	notificationService *services.NotificationService
}

// NewTestNotificationController creates a new test notification controller
func NewTestNotificationController() *TestNotificationController {
	return &TestNotificationController{
		notificationService: services.NewNotificationService(),
	}
}

// TestNotificationRequest represents the request body for test notifications
type TestNotificationRequest struct {
	Title string                 `json:"title" binding:"required"`
	Body  string                 `json:"body" binding:"required"`
	Data  map[string]interface{} `json:"data,omitempty"`
}

// SendTestNotification sends a test notification to all users
func (tnc *TestNotificationController) SendTestNotification(c *gin.Context) {
	var req TestNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	// Get all users with FCM tokens
	var userIDs []uint
	if err := db.GetDB().Model(&models.User{}).Where("fcm_token != ''").Pluck("id", &userIDs).Error; err != nil {
		colors.PrintError("Failed to get users with FCM tokens: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get users",
			"message": err.Error(),
		})
		return
	}

	if len(userIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No users with FCM tokens",
			"message": "No users have registered FCM tokens for notifications",
		})
		return
	}

	// Prepare notification data
	notificationData := &services.NotificationData{
		Type:  "test_notification",
		Title: req.Title,
		Body:  req.Body,
		Data:  req.Data,
	}

	// Send test notification
	response, err := tnc.notificationService.SendToMultipleUsers(userIDs, notificationData)
	if err != nil {
		colors.PrintError("Failed to send test notification: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to send notification",
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

	colors.PrintSuccess("Test notification sent successfully to %d users", len(userIDs))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Test notification sent successfully",
		"data":    response,
	})
}

// SendTestTopicNotification sends a test notification to a topic
func (tnc *TestNotificationController) SendTestTopicNotification(c *gin.Context) {
	var req TestNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	topic := c.Param("topic")
	if topic == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Topic required",
			"message": "Please provide a topic name",
		})
		return
	}

	// Prepare notification data
	notificationData := &services.NotificationData{
		Type:  "test_topic_notification",
		Title: req.Title,
		Body:  req.Body,
		Data:  req.Data,
	}

	// Send test topic notification
	response, err := tnc.notificationService.SendToTopic(topic, notificationData)
	if err != nil {
		colors.PrintError("Failed to send test topic notification: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to send notification",
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

	colors.PrintSuccess("Test topic notification sent successfully to topic: %s", topic)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Test topic notification sent successfully",
		"data":    response,
	})
}
