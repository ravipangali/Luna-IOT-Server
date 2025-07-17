package services

import (
	"encoding/json"
	"fmt"
	"log"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
)

type NotificationService struct{}

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

	// Log notification (simulated)
	log.Printf("Notification sent to user %d (%s): %s - %s", userID, user.Name, notification.Title, notification.Body)

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

	// Log multicast notification
	log.Printf("Multicast notification sent to %d users: %s - %s", len(users), notification.Title, notification.Body)

	return &NotificationServiceResponse{
		Success: true,
		Message: fmt.Sprintf("Multicast notification sent successfully to %d users", len(users)),
	}, nil
}

// SendToTopic sends notification to a topic
func (ns *NotificationService) SendToTopic(topic string, notification *NotificationData) (*NotificationServiceResponse, error) {
	// Log topic notification
	log.Printf("Topic notification sent to '%s': %s - %s", topic, notification.Title, notification.Body)

	return &NotificationServiceResponse{
		Success: true,
		Message: fmt.Sprintf("Topic notification sent successfully for topic '%s'", topic),
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
