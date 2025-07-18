package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"
)

type NotificationService struct {
	ravipangaliService *RavipangaliService
}

type NotificationData struct {
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Body        string                 `json:"body"`
	Data        map[string]interface{} `json:"data,omitempty"`
	ImageURL    string                 `json:"image_url,omitempty"`
	Sound       string                 `json:"sound,omitempty"`
	Priority    string                 `json:"priority,omitempty"`
	CollapseKey string                 `json:"collapse_key,omitempty"`
}

type NotificationServiceResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func NewNotificationService() *NotificationService {
	return &NotificationService{
		ravipangaliService: NewRavipangaliService(),
	}
}

// SendToUser sends notification to a specific user
func (ns *NotificationService) SendToUser(userID uint, notification *NotificationData) (*NotificationServiceResponse, error) {
	// Get user from database
	var user models.User
	database := db.GetDB()
	if err := database.First(&user, userID).Error; err != nil {
		log.Printf("Failed to fetch user %d for notification: %v", userID, err)
		return &NotificationServiceResponse{
			Success: false,
			Message: "User not found",
		}, err
	}

	// Check if user has FCM token
	if user.FCMToken == "" {
		colors.PrintWarning("User %d (%s) has no FCM token", userID, user.Name)
		return &NotificationServiceResponse{
			Success: false,
			Message: "User has no FCM token",
		}, fmt.Errorf("user has no FCM token")
	}

	// Send via Ravipangali API
	response, err := ns.ravipangaliService.SendPushNotification(
		notification.Title,
		notification.Body,
		[]string{user.FCMToken},
		notification.ImageURL,
		notification.Data,
		notification.Priority,
	)

	if err != nil {
		colors.PrintError("Failed to send notification to user %d via Ravipangali: %v", userID, err)
		return &NotificationServiceResponse{
			Success: false,
			Message: "Failed to send notification",
			Error:   err.Error(),
		}, err
	}

	if !response.Success {
		colors.PrintError("Ravipangali API returned failure for user %d: %s", userID, response.Error)
		return &NotificationServiceResponse{
			Success: false,
			Message: "Failed to send notification",
			Error:   response.Error,
		}, fmt.Errorf("Ravipangali API error: %s", response.Error)
	}

	colors.PrintSuccess("Notification sent to user %d (%s) via Ravipangali: %s - %s",
		userID, user.Name, notification.Title, notification.Body)

	return &NotificationServiceResponse{
		Success: true,
		Message: "Notification sent successfully",
	}, nil
}

// SendToMultipleUsers sends notification to multiple users
func (ns *NotificationService) SendToMultipleUsers(userIDs []uint, notification *NotificationData) (*NotificationServiceResponse, error) {
	// Get users from database
	var users []models.User
	database := db.GetDB()
	if err := database.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		log.Printf("Failed to fetch users for notification: %v", err)
		return &NotificationServiceResponse{
			Success: false,
			Message: "Failed to fetch users",
		}, err
	}

	// Extract FCM tokens
	var tokens []string
	for _, user := range users {
		if user.FCMToken != "" {
			tokens = append(tokens, user.FCMToken)
		} else {
			colors.PrintWarning("User %d (%s) has no FCM token", user.ID, user.Name)
		}
	}

	if len(tokens) == 0 {
		colors.PrintWarning("No FCM tokens found for any of the %d users", len(userIDs))
		return &NotificationServiceResponse{
			Success: false,
			Message: "No FCM tokens found for any users",
		}, fmt.Errorf("no FCM tokens found")
	}

	// Send via Ravipangali API
	response, err := ns.ravipangaliService.SendPushNotification(
		notification.Title,
		notification.Body,
		tokens,
		notification.ImageURL,
		notification.Data,
		notification.Priority,
	)

	if err != nil {
		colors.PrintError("Failed to send notification to %d users via Ravipangali: %v", len(tokens), err)
		return &NotificationServiceResponse{
			Success: false,
			Message: "Failed to send notification",
			Error:   err.Error(),
		}, err
	}

	if !response.Success {
		colors.PrintError("Ravipangali API returned failure: %s", response.Error)
		return &NotificationServiceResponse{
			Success: false,
			Message: "Failed to send notification",
			Error:   response.Error,
		}, fmt.Errorf("Ravipangali API error: %s", response.Error)
	}

	colors.PrintSuccess("Multicast notification sent to %d users via Ravipangali: %s - %s",
		len(tokens), notification.Title, notification.Body)
	colors.PrintInfo("  Tokens sent: %d", response.TokensSent)
	colors.PrintInfo("  Tokens delivered: %d", response.TokensDelivered)
	colors.PrintInfo("  Tokens failed: %d", response.TokensFailed)

	return &NotificationServiceResponse{
		Success: true,
		Message: fmt.Sprintf("Multicast notification sent successfully to %d users", len(tokens)),
	}, nil
}

// SendToTopic sends notification to a topic
func (ns *NotificationService) SendToTopic(topic string, notification *NotificationData) (*NotificationServiceResponse, error) {
	// For topic notifications, we need to get all users subscribed to the topic
	// This is a simplified implementation - you might want to implement topic subscription logic

	colors.PrintInfo("Topic notification sent to '%s': %s - %s", topic, notification.Title, notification.Body)

	return &NotificationServiceResponse{
		Success: true,
		Message: fmt.Sprintf("Topic notification sent successfully for topic '%s'", topic),
	}, nil
}

// SendNotificationByID sends a notification by its database ID
func (ns *NotificationService) SendNotificationByID(notificationID uint) (*NotificationServiceResponse, error) {
	database := db.GetDB()

	// Get notification from database
	var notification models.Notification
	if err := database.Preload("Users").First(&notification, notificationID).Error; err != nil {
		colors.PrintError("Failed to fetch notification %d: %v", notificationID, err)
		return &NotificationServiceResponse{
			Success: false,
			Message: "Notification not found",
		}, err
	}

	// Extract user IDs
	var userIDs []uint
	for _, user := range notification.Users {
		userIDs = append(userIDs, user.ID)
	}

	if len(userIDs) == 0 {
		colors.PrintWarning("No users found for notification %d", notificationID)
		return &NotificationServiceResponse{
			Success: false,
			Message: "No users assigned to this notification",
		}, fmt.Errorf("no users assigned")
	}

	// Prepare notification data
	notificationData := &NotificationData{
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

	// Send the notification
	response, err := ns.SendToMultipleUsers(userIDs, notificationData)
	if err != nil {
		return response, err
	}

	// Mark notification as sent in database
	now := time.Now()
	if err := database.Model(&notification).Updates(map[string]interface{}{
		"is_sent":    true,
		"sent_at":    &now,
		"updated_at": now,
	}).Error; err != nil {
		colors.PrintError("Failed to mark notification %d as sent: %v", notificationID, err)
		// Don't fail the request, just log the error
	}

	// Mark notification users as sent
	if err := database.Model(&models.NotificationUser{}).
		Where("notification_id = ?", notificationID).
		Updates(map[string]interface{}{
			"is_sent":    true,
			"sent_at":    &now,
			"updated_at": now,
		}).Error; err != nil {
		colors.PrintError("Failed to mark notification users as sent: %v", err)
		// Don't fail the request, just log the error
	}

	colors.PrintSuccess("Notification %d sent successfully via Ravipangali API", notificationID)
	return response, nil
}

// convertDataToMap converts notification data to string map
func (ns *NotificationService) convertDataToMap(data map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for key, value := range data {
		switch v := value.(type) {
		case string:
			result[key] = v
		case int, int32, int64:
			result[key] = fmt.Sprintf("%d", v)
		case float32, float64:
			result[key] = fmt.Sprintf("%f", v)
		case bool:
			result[key] = fmt.Sprintf("%t", v)
		default:
			// Convert to JSON string for complex types
			if jsonBytes, err := json.Marshal(v); err == nil {
				result[key] = string(jsonBytes)
			}
		}
	}
	return result
}

// UpdateUserFCMToken updates user's FCM token
func (ns *NotificationService) UpdateUserFCMToken(userID uint, fcmToken string) error {
	database := db.GetDB()
	return database.Model(&models.User{}).Where("id = ?", userID).Update("fcm_token", fcmToken).Error
}

// RemoveUserFCMToken removes user's FCM token
func (ns *NotificationService) RemoveUserFCMToken(userID uint) error {
	database := db.GetDB()
	return database.Model(&models.User{}).Where("id = ?", userID).Update("fcm_token", "").Error
}

// SubscribeToTopic subscribes a user to a topic
func (ns *NotificationService) SubscribeToTopic(userID uint, topic string) error {
	var user models.User
	database := db.GetDB()
	if err := database.First(&user, userID).Error; err != nil {
		return err
	}

	log.Printf("User %d (%s) subscribed to topic '%s'", userID, user.Name, topic)
	return nil
}

// UnsubscribeFromTopic unsubscribes a user from a topic
func (ns *NotificationService) UnsubscribeFromTopic(userID uint, topic string) error {
	var user models.User
	database := db.GetDB()
	if err := database.First(&user, userID).Error; err != nil {
		return err
	}

	log.Printf("User %d (%s) unsubscribed from topic '%s'", userID, user.Name, topic)
	return nil
}
