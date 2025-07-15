package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"luna_iot_server/config"
	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"

	"firebase.google.com/go/v4/messaging"
)

type NotificationService struct {
	messagingClient *messaging.Client
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
		messagingClient: config.GetMessagingClient(),
	}
}

// SendToUser sends notification to a specific user
func (ns *NotificationService) SendToUser(userID uint, notification *NotificationData) (*NotificationServiceResponse, error) {
	if !config.IsFirebaseEnabled() {
		log.Printf("Firebase not configured, returning success for user notification: %s", notification.Title)
		return &NotificationServiceResponse{
			Success: true,
			Message: "Firebase not configured - notification would be sent if Firebase was enabled",
		}, nil
	}

	// Get user's FCM token from database
	var user models.User
	database := db.GetDB()
	if err := database.First(&user, userID).Error; err != nil {
		log.Printf("Failed to fetch user %d for notification: %v", userID, err)
		return &NotificationServiceResponse{
			Success: false,
			Message: "User not found",
		}, err
	}

	if user.FCMToken == "" {
		log.Printf("User %d has no FCM token", userID)
		return &NotificationServiceResponse{
			Success: true,
			Message: "User has no FCM token - notification would be sent if user had FCM token",
		}, nil
	}

	return ns.sendToToken(user.FCMToken, notification)
}

// SendToMultipleUsers sends notification to multiple users
func (ns *NotificationService) SendToMultipleUsers(userIDs []uint, notification *NotificationData) (*NotificationServiceResponse, error) {
	colors.PrintInfo("SendToMultipleUsers called with %d user IDs: %v", len(userIDs), userIDs)
	colors.PrintInfo("Notification data: Title='%s', Body='%s', Type='%s'",
		notification.Title, notification.Body, notification.Type)

	if !config.IsFirebaseEnabled() {
		colors.PrintWarning("Firebase not configured, returning success for notification: %s", notification.Title)
		return &NotificationServiceResponse{
			Success: true,
			Message: "Firebase not configured - notification would be sent if Firebase was enabled",
		}, nil
	}

	colors.PrintInfo("Firebase is enabled, proceeding with notification send")

	// Get users' FCM tokens from database
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

	var tokens []string
	for _, user := range users {
		if user.FCMToken != "" {
			tokens = append(tokens, user.FCMToken)
			colors.PrintInfo("User %d has FCM token: %s", user.ID, user.FCMToken[:20]+"...")
		} else {
			colors.PrintInfo("User %d has no FCM token", user.ID)
		}
	}

	if len(tokens) == 0 {
		colors.PrintWarning("No valid FCM tokens found for users: %v", userIDs)
		return &NotificationServiceResponse{
			Success: true,
			Message: "No valid FCM tokens found - notification would be sent if users had FCM tokens",
		}, nil
	}

	colors.PrintInfo("Sending notification to %d FCM tokens", len(tokens))
	return ns.sendToMultipleTokens(tokens, notification)
}

// SendToTopic sends notification to a topic
func (ns *NotificationService) SendToTopic(topic string, notification *NotificationData) (*NotificationServiceResponse, error) {
	if !config.IsFirebaseEnabled() {
		log.Printf("Firebase not configured, returning success for topic notification: %s", notification.Title)
		return &NotificationServiceResponse{
			Success: true,
			Message: "Firebase not configured - notification would be sent if Firebase was enabled",
		}, nil
	}

	message := &messaging.Message{
		Topic: topic,
		Notification: &messaging.Notification{
			Title: notification.Title,
			Body:  notification.Body,
		},
		Data: ns.convertDataToMap(notification.Data),
		Android: &messaging.AndroidConfig{
			Priority: notification.Priority,
			Notification: &messaging.AndroidNotification{
				Sound: notification.Sound,
				Icon:  "ic_notification",
			},
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Sound: notification.Sound,
					Badge: func() *int { i := 1; return &i }(),
				},
			},
		},
	}

	response, err := ns.messagingClient.Send(context.Background(), message)
	if err != nil {
		log.Printf("Failed to send notification to topic %s: %v", topic, err)
		return &NotificationServiceResponse{
			Success: false,
			Message: "Failed to send notification",
			Error:   err.Error(),
		}, err
	}

	return &NotificationServiceResponse{
		Success: true,
		Message: fmt.Sprintf("Notification sent successfully. Message ID: %s", response),
	}, nil
}

// SendToToken sends notification to a specific FCM token
func (ns *NotificationService) sendToToken(token string, notification *NotificationData) (*NotificationServiceResponse, error) {
	message := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: notification.Title,
			Body:  notification.Body,
		},
		Data: ns.convertDataToMap(notification.Data),
		Android: &messaging.AndroidConfig{
			Priority: notification.Priority,
			Notification: &messaging.AndroidNotification{
				Sound: notification.Sound,
				Icon:  "ic_notification",
			},
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Sound: notification.Sound,
					Badge: func() *int { i := 1; return &i }(),
				},
			},
		},
	}

	response, err := ns.messagingClient.Send(context.Background(), message)
	if err != nil {
		log.Printf("Failed to send notification to token %s: %v", token, err)
		return &NotificationServiceResponse{
			Success: false,
			Message: "Failed to send notification",
			Error:   err.Error(),
		}, err
	}

	return &NotificationServiceResponse{
		Success: true,
		Message: fmt.Sprintf("Notification sent successfully. Message ID: %s", response),
	}, nil
}

// SendToMultipleTokens sends notification to multiple FCM tokens
func (ns *NotificationService) sendToMultipleTokens(tokens []string, notification *NotificationData) (*NotificationServiceResponse, error) {
	message := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: notification.Title,
			Body:  notification.Body,
		},
		Data: ns.convertDataToMap(notification.Data),
		Android: &messaging.AndroidConfig{
			Priority: notification.Priority,
			Notification: &messaging.AndroidNotification{
				Sound: notification.Sound,
				Icon:  "ic_notification",
			},
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Sound: notification.Sound,
					Badge: func() *int { i := 1; return &i }(),
				},
			},
		},
	}

	response, err := ns.messagingClient.SendMulticast(context.Background(), message)
	if err != nil {
		log.Printf("Failed to send multicast notification: %v", err)
		return &NotificationServiceResponse{
			Success: false,
			Message: "Failed to send notifications",
			Error:   err.Error(),
		}, err
	}

	return &NotificationServiceResponse{
		Success: true,
		Message: fmt.Sprintf("Notifications sent successfully. Success: %d, Failure: %d", response.SuccessCount, response.FailureCount),
	}, nil
}

// convertDataToMap converts notification data to string map for FCM
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
	if !config.IsFirebaseEnabled() {
		return fmt.Errorf("Firebase not configured")
	}

	var user models.User
	database := db.GetDB()
	if err := database.First(&user, userID).Error; err != nil {
		return err
	}

	if user.FCMToken == "" {
		return fmt.Errorf("User has no FCM token")
	}

	_, err := ns.messagingClient.SubscribeToTopic(context.Background(), []string{user.FCMToken}, topic)
	return err
}

// UnsubscribeFromTopic unsubscribes a user from a topic
func (ns *NotificationService) UnsubscribeFromTopic(userID uint, topic string) error {
	if !config.IsFirebaseEnabled() {
		return fmt.Errorf("Firebase not configured")
	}

	var user models.User
	database := db.GetDB()
	if err := database.First(&user, userID).Error; err != nil {
		return err
	}

	if user.FCMToken == "" {
		return fmt.Errorf("User has no FCM token")
	}

	_, err := ns.messagingClient.UnsubscribeFromTopic(context.Background(), []string{user.FCMToken}, topic)
	return err
}
