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
	// No Firebase dependencies
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
	return &NotificationService{}
}

// SendToUser sends notification to a specific user (simulated)
func (ns *NotificationService) SendToUser(userID uint, notification *NotificationData) (*NotificationServiceResponse, error) {
	colors.PrintInfo("=== NOTIFICATION DEBUG INFO (SendToUser) ===")
	colors.PrintInfo("Firebase removed - simulating notification for user %d", userID)
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

	// Simulate successful notification
	log.Printf("Simulated notification sent to user %d: %s", userID, notification.Title)

	return &NotificationServiceResponse{
		Success: true,
		Message: "Notification simulated successfully (Firebase removed)",
	}, nil
}

// SendToMultipleUsers sends notification to multiple users (simulated)
func (ns *NotificationService) SendToMultipleUsers(userIDs []uint, notification *NotificationData) (*NotificationServiceResponse, error) {
	colors.PrintInfo("SendToMultipleUsers called with %d user IDs: %v", len(userIDs), userIDs)
	colors.PrintInfo("Notification data: Title='%s', Body='%s', Type='%s'",
		notification.Title, notification.Body, notification.Type)

	colors.PrintInfo("=== NOTIFICATION DEBUG INFO ===")
	colors.PrintInfo("Firebase removed - simulating notifications for %d users", len(userIDs))
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

	// Simulate successful notification
	log.Printf("Simulated notifications sent to %d users: %s", len(users), notification.Title)

	return &NotificationServiceResponse{
		Success: true,
		Message: fmt.Sprintf("Notifications simulated successfully for %d users (Firebase removed)", len(users)),
	}, nil
}

// SendToTopic sends notification to a topic (simulated)
func (ns *NotificationService) SendToTopic(topic string, notification *NotificationData) (*NotificationServiceResponse, error) {
	log.Printf("Simulated topic notification: %s", notification.Title)

	return &NotificationServiceResponse{
		Success: true,
		Message: fmt.Sprintf("Topic notification simulated successfully for topic '%s' (Firebase removed)", topic),
	}, nil
}

// SendToToken sends notification to a specific FCM token (simulated)
func (ns *NotificationService) sendToToken(token string, notification *NotificationData) (*NotificationServiceResponse, error) {
	log.Printf("Simulated token notification: %s", notification.Title)

	return &NotificationServiceResponse{
		Success: true,
		Message: "Token notification simulated successfully (Firebase removed)",
	}, nil
}

// SendToMultipleTokens sends notification to multiple FCM tokens (simulated)
func (ns *NotificationService) sendToMultipleTokens(tokens []string, notification *NotificationData) (*NotificationServiceResponse, error) {
	log.Printf("Simulated multicast notification to %d tokens: %s", len(tokens), notification.Title)

	return &NotificationServiceResponse{
		Success: true,
		Message: fmt.Sprintf("Multicast notifications simulated successfully for %d tokens (Firebase removed)", len(tokens)),
	}, nil
}

// convertDataToMap converts notification data to string map (kept for compatibility)
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

// UpdateUserFCMToken updates user's FCM token (simulated)
func (ns *NotificationService) UpdateUserFCMToken(userID uint, fcmToken string) error {
	database := db.GetDB()
	log.Printf("Simulated FCM token update for user %d", userID)
	return database.Model(&models.User{}).Where("id = ?", userID).Update("fcm_token", fcmToken).Error
}

// RemoveUserFCMToken removes user's FCM token (simulated)
func (ns *NotificationService) RemoveUserFCMToken(userID uint) error {
	database := db.GetDB()
	log.Printf("Simulated FCM token removal for user %d", userID)
	return database.Model(&models.User{}).Where("id = ?", userID).Update("fcm_token", "").Error
}

// SubscribeToTopic subscribes a user to a topic (simulated)
func (ns *NotificationService) SubscribeToTopic(userID uint, topic string) error {
	log.Printf("Simulated topic subscription for user %d to topic '%s'", userID, topic)
	return nil
}

// UnsubscribeFromTopic unsubscribes a user from a topic (simulated)
func (ns *NotificationService) UnsubscribeFromTopic(userID uint, topic string) error {
	log.Printf("Simulated topic unsubscription for user %d from topic '%s'", userID, topic)
	return nil
}
