package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"
)

type NotificationService struct {
	firebaseApp *firebase.App
	messaging   *messaging.Client
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
	ctx := context.Background()

	// Hardcoded service account JSON with real line breaks in private_key
	serviceAccount := []byte(`{
  "type": "service_account",
  "project_id": "luna-iot-5993f",
  "private_key_id": "7b2de23547167be850a4c997c7d8c53583377ce8",
  "private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCm3BG7delgI/u9\n2ObBHe+1lNyCInFddMu6DFs+8t0kSyCcLaJwpzsd0baa5HymtWuu5j9wlasAFAFl\nFdTUpxzvohw2tM1TTNzOBDfAr+/Ibut66KmTEA8WM2+Y4dGmORecq+25+w4l0fOX\n67h/fSpD1YMwnA30BJgdymzXCezNs4pgQPsrx0o/97dWaxRYrZ+Hhsea6GdYwdBo\ncdVKmL2vf2AUD/5ruJ/3N/zJaUNEs3R2UBEYik3TFOKUsSVtiiFXFoGxV0xW5YPR\n//6+ZOd/o20+OitTVlPojhb1EP5mmIduOI3RWO2/VHVtvIL0L+7VTPhb2QZPjnUt\n46llun23AgMBAAECggEAFQL4Bf/Octl/z+OfsIh726/H0hNS7I59QQu2nxynavSS\n7K0fxskoO+m2nEFSzmNgu1Yp0vGQkPfz8li8w50vmvVyUWk/GdLp/dSwzfDZubGj\nXDIuzbgN+PZigoFH4dilTTNRQkQyVIzgUdclv9gbxGL/RtW/5A8tYJhRpa/i7pwN\naHw8W6IMUKd9HuCqXOBXhH1GggMhCEJt2wm7PXzxgsKVa4zxPx82vI2Tuhgox+UE\nIipaK26GkvnDjPyRLyGNZk1f0ntYjv8TbDj3rmsSSfCbxgG+otWS8rlgTN44L7w9\nkBCLVjlslIA2x8qmcbzXK635dPvIqB3OnTo5ca3xYQKBgQDrY9k1V6mpELG5zCaF\nAqE8WN1AIwrT2A1cT1KgH9yr8MOX1n0enPeyFRiKMrAnuecyB72KEYVwFrS5m9XX\nwwc+Oa/jCwqgfanTKaf5SyjPzgds5G/fxUi4jB6epr4+6nqnxL0XCswBSTvK6qm0\n+qQJfGF1VJNRZq728CH1+jkrnwKBgQC1eCVc++7k3qsKbuTVjz+kMhdZ89rAgEmN\n1z9l6eWF1QYZ5wQS0trN9deN+sG0PjRsukR4VJw0HJxX/wlPvlIbWUmPaauaN+sb\naOfVkuoC+3N/bpwDfLd1wVg8KQ/lyTL9PvSQpLGwU17Qms+wqLA+gRDoX3nk2XOv\nctungtn26QKBgQCF1ZmUGKmgNJu4NfjYu2wNMcFqTAJF/JtsFrW10SfYouWymQM+\nuqSinhf7y2IY1Dw9V+VOcTPbTS2oMpBdQsgFeysj/g0mvwwlwZN9zFwB+vSB10g8\nhKEaPKDUN54Hi639YYDZbwwa1xamAtJG0hMeSZfn7BRuveFRCatlfcWvpQKBgCml\nKO3t4yUi9J2wVVOtTC2iUTmTfOAwkLC8dRAuXT4ZZQ0MtyKawRwDDzTGFy4GGIHb\nPVtgD3jmF/sZzElApBcipn8DAR6jNpFTweCBlrKYgij8eVFTjca4WEd2JO/W/Jyh\nlf6bzStp9pho7sDb9ZZiiD7Lqm2aebIJ6d7HaL4BAoGAHBS4o1ymFAS7GyF6Oc58\n9DH8lF/G3iAgZP6qvgt8OPeLk1Xvcd/BIHJf6P2VOYYGQwkrwBsYhL1Y5knEUUsi\nsWEW0HI+zKJbjieuFcKF2th6OYB/EK5k1Zyd2F4Ip3W8gCjyNOp7HhYQ+sv/YLCL\nzxcbvtbrDdQskKZGrFkWlsw=\n-----END PRIVATE KEY-----\n",
  "client_email": "firebase-adminsdk-fbsvc@luna-iot-5993f.iam.gserviceaccount.com",
  "client_id": "108105592493412976295",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/firebase-adminsdk-fbsvc%40luna-iot-5993f.iam.gserviceaccount.com"
}`)

	// Initialize Firebase app with credentials
	opt := option.WithCredentialsJSON(serviceAccount)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		colors.PrintError("Error initializing Firebase app: %v", err)
		return &NotificationService{}
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		colors.PrintError("Error getting Messaging client: %v", err)
		return &NotificationService{}
	}

	colors.PrintSuccess("Firebase initialized successfully")

	return &NotificationService{
		firebaseApp: app,
		messaging:   client,
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

	// Send notification using Firebase
	response, err := ns.sendToToken(user.FCMToken, notification)
	if err != nil {
		colors.PrintError("Failed to send notification to user %d: %v", userID, err)
		return &NotificationServiceResponse{
			Success: false,
			Message: "Failed to send notification",
			Error:   err.Error(),
		}, err
	}

	return response, nil
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

	// Send notification using Firebase
	response, err := ns.sendToMultipleTokens(tokens, notification)
	if err != nil {
		colors.PrintError("Failed to send notifications: %v", err)
		return &NotificationServiceResponse{
			Success: false,
			Message: "Failed to send notifications",
			Error:   err.Error(),
		}, err
	}

	return response, nil
}

// SendToTopic sends notification to a topic
func (ns *NotificationService) SendToTopic(topic string, notification *NotificationData) (*NotificationServiceResponse, error) {
	ctx := context.Background()

	// Define FCM message for topic
	message := &messaging.Message{
		Topic: topic,
		Notification: &messaging.Notification{
			Title: notification.Title,
			Body:  notification.Body,
		},
		Data: ns.convertDataToMap(notification.Data),
	}

	// Send message
	response, err := ns.messaging.Send(ctx, message)
	if err != nil {
		colors.PrintError("Error sending FCM topic message: %v", err)
		return &NotificationServiceResponse{
			Success: false,
			Message: "Failed to send topic notification",
			Error:   err.Error(),
		}, err
	}

	colors.PrintSuccess("Successfully sent topic message: %s", response)
	return &NotificationServiceResponse{
		Success: true,
		Message: fmt.Sprintf("Topic notification sent successfully for topic '%s'", topic),
	}, nil
}

// sendToToken sends notification to a specific FCM token
func (ns *NotificationService) sendToToken(token string, notification *NotificationData) (*NotificationServiceResponse, error) {
	ctx := context.Background()

	// Define FCM message
	message := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: notification.Title,
			Body:  notification.Body,
		},
		Data: ns.convertDataToMap(notification.Data),
	}

	// Send message
	response, err := ns.messaging.Send(ctx, message)
	if err != nil {
		colors.PrintError("Error sending FCM message: %v", err)
		return &NotificationServiceResponse{
			Success: false,
			Message: "Failed to send notification",
			Error:   err.Error(),
		}, err
	}

	colors.PrintSuccess("Successfully sent message: %s", response)
	return &NotificationServiceResponse{
		Success: true,
		Message: "Notification sent successfully",
	}, nil
}

// sendToMultipleTokens sends notification to multiple FCM tokens
func (ns *NotificationService) sendToMultipleTokens(tokens []string, notification *NotificationData) (*NotificationServiceResponse, error) {
	ctx := context.Background()

	// Define FCM message for multicast
	message := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: notification.Title,
			Body:  notification.Body,
		},
		Data: ns.convertDataToMap(notification.Data),
	}

	// Send message
	response, err := ns.messaging.SendMulticast(ctx, message)
	if err != nil {
		colors.PrintError("Error sending FCM multicast message: %v", err)
		return &NotificationServiceResponse{
			Success: false,
			Message: "Failed to send multicast notification",
			Error:   err.Error(),
		}, err
	}

	colors.PrintSuccess("Successfully sent multicast message: %d successful, %d failed", response.SuccessCount, response.FailureCount)
	return &NotificationServiceResponse{
		Success: true,
		Message: fmt.Sprintf("Multicast notifications sent successfully: %d successful, %d failed", response.SuccessCount, response.FailureCount),
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

	ctx := context.Background()
	_, err := ns.messaging.SubscribeToTopic(ctx, []string{user.FCMToken}, topic)
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

	ctx := context.Background()
	_, err := ns.messaging.UnsubscribeFromTopic(ctx, []string{user.FCMToken}, topic)
	if err != nil {
		colors.PrintError("Error unsubscribing from topic: %v", err)
		return err
	}

	colors.PrintSuccess("User %d unsubscribed from topic '%s'", userID, topic)
	return nil
}
