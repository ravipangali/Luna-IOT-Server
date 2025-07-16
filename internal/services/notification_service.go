package services

import (
	"encoding/json"
	"fmt"
	"log"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"
)

type NotificationService struct {
	firebaseService *FirebaseService
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
	// Initialize Firebase service
	firebaseService := NewFirebaseService()

	return &NotificationService{
		firebaseService: firebaseService,
	}
}

// SendToUser sends notification to a specific user
func (ns *NotificationService) SendToUser(userID uint, notification *NotificationData) (*NotificationServiceResponse, error) {
	colors.PrintInfo("=== NOTIFICATION DEBUG INFO (SendToUser) ===")
	colors.PrintInfo("Sending notification to user %d", userID)
	colors.PrintInfo("Notification: Title='%s', Body='%s'", notification.Title, notification.Body)
	colors.PrintInfo("=============================================")

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
		colors.PrintWarning("User %d has no FCM token", userID)
		return &NotificationServiceResponse{
			Success: false,
			Message: "User has no FCM token",
		}, nil
	}

	// Send notification using Firebase service
	msg := &NotificationMessage{
		Token:    user.FCMToken,
		Title:    notification.Title,
		Body:     notification.Body,
		Data:     ns.convertDataToMap(notification.Data),
		ImageURL: notification.ImageURL,
		Sound:    notification.Sound,
		Priority: notification.Priority,
	}

	response, err := ns.firebaseService.SendNotification(msg)
	if err != nil {
		colors.PrintError("Failed to send notification to user %d: %v", userID, err)
		return &NotificationServiceResponse{
			Success: false,
			Message: "Failed to send notification",
			Error:   err.Error(),
		}, err
	}

	return &NotificationServiceResponse{
		Success: response.Success,
		Message: response.Message,
		Error:   response.Error,
	}, nil
}

// SendToMultipleUsers sends notification to multiple users
func (ns *NotificationService) SendToMultipleUsers(userIDs []uint, notification *NotificationData) (*NotificationServiceResponse, error) {
	colors.PrintInfo("SendToMultipleUsers called with %d user IDs: %v", len(userIDs), userIDs)
	colors.PrintInfo("Notification data: Title='%s', Body='%s', Type='%s'",
		notification.Title, notification.Body, notification.Type)

	colors.PrintInfo("=== NOTIFICATION DEBUG INFO ===")
	colors.PrintInfo("Sending notifications to %d users", len(userIDs))
	colors.PrintInfo("===============================")

	// Get users from database
	var users []models.User
	database := db.GetDB()
	if err := database.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		colors.PrintError("Failed to fetch users for notification: %v", err)
		return &NotificationServiceResponse{
			Success: false,
			Message: "Failed to fetch users",
		}, err
	}

	colors.PrintInfo("Found %d users in database", len(users))

	// Collect FCM tokens
	var tokens []string
	for _, user := range users {
		if user.FCMToken != "" {
			tokens = append(tokens, user.FCMToken)
		}
	}

	if len(tokens) == 0 {
		colors.PrintWarning("No FCM tokens found for users")
		return &NotificationServiceResponse{
			Success: false,
			Message: "No FCM tokens found for users",
		}, nil
	}

	// Send notification using Firebase service
	response, err := ns.firebaseService.SendToMultipleTokens(tokens, notification.Title, notification.Body, ns.convertDataToMap(notification.Data))
	if err != nil {
		colors.PrintError("Failed to send notifications: %v", err)
		return &NotificationServiceResponse{
			Success: false,
			Message: "Failed to send notifications",
			Error:   err.Error(),
		}, err
	}

	return &NotificationServiceResponse{
		Success: response.Success,
		Message: response.Message,
		Error:   response.Error,
	}, nil
}

// SendToTopic sends notification to a topic
func (ns *NotificationService) SendToTopic(topic string, notification *NotificationData) (*NotificationServiceResponse, error) {
	// Send notification using Firebase service
	response, err := ns.firebaseService.SendToTopic(topic, notification.Title, notification.Body, ns.convertDataToMap(notification.Data))
	if err != nil {
		colors.PrintError("Error sending FCM topic message: %v", err)
		return &NotificationServiceResponse{
			Success: false,
			Message: "Failed to send topic notification",
			Error:   err.Error(),
		}, err
	}

	return &NotificationServiceResponse{
		Success: response.Success,
		Message: response.Message,
		Error:   response.Error,
	}, nil
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
	colors.PrintInfo("Updating FCM token for user %d", userID)
	return database.Model(&models.User{}).Where("id = ?", userID).Update("fcm_token", fcmToken).Error
}

// RemoveUserFCMToken removes user's FCM token
func (ns *NotificationService) RemoveUserFCMToken(userID uint) error {
	database := db.GetDB()
	colors.PrintInfo("Removing FCM token for user %d", userID)
	return database.Model(&models.User{}).Where("id = ?", userID).Update("fcm_token", "").Error
}

// SubscribeToTopic subscribes a user to a topic
func (ns *NotificationService) SubscribeToTopic(userID uint, topic string) error {
	// Get user's FCM token
	var user models.User
	database := db.GetDB()
	if err := database.First(&user, userID).Error; err != nil {
		return err
	}

	if user.FCMToken == "" {
		return fmt.Errorf("user has no FCM token")
	}

	err := ns.firebaseService.SubscribeToTopic([]string{user.FCMToken}, topic)
	if err != nil {
		colors.PrintError("Error subscribing to topic: %v", err)
		return err
	}

	colors.PrintSuccess("User %d subscribed to topic '%s'", userID, topic)
	return nil
}

// UnsubscribeFromTopic unsubscribes a user from a topic
func (ns *NotificationService) UnsubscribeFromTopic(userID uint, topic string) error {
	// Get user's FCM token
	var user models.User
	database := db.GetDB()
	if err := database.First(&user, userID).Error; err != nil {
		return err
	}

	if user.FCMToken == "" {
		return fmt.Errorf("user has no FCM token")
	}

	err := ns.firebaseService.UnsubscribeFromTopic([]string{user.FCMToken}, topic)
	if err != nil {
		colors.PrintError("Error unsubscribing from topic: %v", err)
		return err
	}

	colors.PrintSuccess("User %d unsubscribed from topic '%s'", userID, topic)
	return nil
}
