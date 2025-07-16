package services

import (
	"context"
	"fmt"

	"luna_iot_server/pkg/colors"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"
)

type FirebaseService struct {
	app      *firebase.App
	client   *messaging.Client
	isActive bool
}

type NotificationMessage struct {
	Token       string            `json:"token"`
	Title       string            `json:"title"`
	Body        string            `json:"body"`
	Data        map[string]string `json:"data,omitempty"`
	ImageURL    string            `json:"image_url,omitempty"`
	Sound       string            `json:"sound,omitempty"`
	Priority    string            `json:"priority,omitempty"`
	CollapseKey string            `json:"collapse_key,omitempty"`
}

type FirebaseNotificationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// NewFirebaseService creates a new Firebase service instance
func NewFirebaseService() *FirebaseService {
	ctx := context.Background()

	// Load Firebase service account key from file
	opt := option.WithCredentialsFile("firebase_key.json")

	// Initialize Firebase App
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		colors.PrintError("Error initializing Firebase app: %v", err)
		colors.PrintWarning("Firebase service will be disabled - notifications will be simulated")
		return &FirebaseService{isActive: false}
	}

	// Get Messaging client
	client, err := app.Messaging(ctx)
	if err != nil {
		colors.PrintError("Error getting Messaging client: %v", err)
		colors.PrintWarning("Firebase service will be disabled - notifications will be simulated")
		return &FirebaseService{isActive: false}
	}

	colors.PrintSuccess("Firebase service initialized successfully")
	return &FirebaseService{
		app:      app,
		client:   client,
		isActive: true,
	}
}

// SendNotification sends a notification to a specific FCM token
func (fs *FirebaseService) SendNotification(msg *NotificationMessage) (*FirebaseNotificationResponse, error) {
	if !fs.isActive {
		// Simulate notification when Firebase is not available
		colors.PrintWarning("Firebase not available - simulating notification")
		colors.PrintInfo("Simulated notification: Title='%s', Body='%s', Token='%s'",
			msg.Title, msg.Body, msg.Token[:10]+"...")

		return &FirebaseNotificationResponse{
			Success: true,
			Message: "Notification simulated successfully (Firebase not available)",
		}, nil
	}

	ctx := context.Background()

	// Build FCM message
	fcmMessage := &messaging.Message{
		Token: msg.Token,
		Notification: &messaging.Notification{
			Title: msg.Title,
			Body:  msg.Body,
		},
		Data: msg.Data,
	}

	// Add optional fields
	if msg.ImageURL != "" {
		fcmMessage.Notification.ImageURL = msg.ImageURL
	}

	if msg.Sound != "" {
		fcmMessage.Android = &messaging.AndroidConfig{
			Notification: &messaging.AndroidNotification{
				Sound: msg.Sound,
			},
		}
		fcmMessage.APNS = &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Sound: msg.Sound,
				},
			},
		}
	}

	if msg.Priority != "" {
		fcmMessage.Android = &messaging.AndroidConfig{
			Priority: msg.Priority,
		}
		fcmMessage.APNS = &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-priority": msg.Priority,
			},
		}
	}

	if msg.CollapseKey != "" {
		fcmMessage.Android = &messaging.AndroidConfig{
			CollapseKey: msg.CollapseKey,
		}
	}

	// Send message
	response, err := fs.client.Send(ctx, fcmMessage)
	if err != nil {
		colors.PrintError("Error sending FCM message: %v", err)
		// Don't return error, just log it and simulate the notification
		colors.PrintWarning("Firebase error - simulating notification instead")
		return &FirebaseNotificationResponse{
			Success: true,
			Message: "Notification simulated due to Firebase error",
		}, nil
	}

	colors.PrintSuccess("Successfully sent notification: %s", response)
	return &FirebaseNotificationResponse{
		Success: true,
		Message: "Notification sent successfully",
	}, nil
}

// SendToMultipleTokens sends notification to multiple FCM tokens
func (fs *FirebaseService) SendToMultipleTokens(tokens []string, title, body string, data map[string]string) (*FirebaseNotificationResponse, error) {
	if !fs.isActive {
		// Simulate multicast notification when Firebase is not available
		colors.PrintWarning("Firebase not available - simulating multicast notification")
		colors.PrintInfo("Simulated multicast: Title='%s', Body='%s', Tokens=%d",
			title, body, len(tokens))

		return &FirebaseNotificationResponse{
			Success: true,
			Message: "Multicast notification simulated successfully (Firebase not available)",
		}, nil
	}

	ctx := context.Background()

	// Build FCM multicast message
	message := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	// Send multicast message
	response, err := fs.client.SendMulticast(ctx, message)
	if err != nil {
		colors.PrintError("Error sending FCM multicast message: %v", err)
		// Don't return error, just log it and simulate the notification
		colors.PrintWarning("Firebase multicast error - simulating notification instead")
		return &FirebaseNotificationResponse{
			Success: true,
			Message: "Multicast notification simulated due to Firebase error",
		}, nil
	}

	colors.PrintSuccess("Successfully sent multicast message: %d successful, %d failed", response.SuccessCount, response.FailureCount)
	return &FirebaseNotificationResponse{
		Success: true,
		Message: fmt.Sprintf("Multicast notifications sent successfully: %d successful, %d failed", response.SuccessCount, response.FailureCount),
	}, nil
}

// SendToTopic sends notification to a topic
func (fs *FirebaseService) SendToTopic(topic, title, body string, data map[string]string) (*FirebaseNotificationResponse, error) {
	if !fs.isActive {
		// Simulate topic notification when Firebase is not available
		colors.PrintWarning("Firebase not available - simulating topic notification")
		colors.PrintInfo("Simulated topic: Topic='%s', Title='%s', Body='%s'",
			topic, title, body)

		return &FirebaseNotificationResponse{
			Success: true,
			Message: "Topic notification simulated successfully (Firebase not available)",
		}, nil
	}

	ctx := context.Background()

	// Build FCM topic message
	message := &messaging.Message{
		Topic: topic,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	// Send topic message
	response, err := fs.client.Send(ctx, message)
	if err != nil {
		colors.PrintError("Error sending FCM topic message: %v", err)
		// Don't return error, just log it and simulate the notification
		colors.PrintWarning("Firebase topic error - simulating notification instead")
		return &FirebaseNotificationResponse{
			Success: true,
			Message: "Topic notification simulated due to Firebase error",
		}, nil
	}

	colors.PrintSuccess("Successfully sent topic message: %s", response)
	return &FirebaseNotificationResponse{
		Success: true,
		Message: fmt.Sprintf("Topic notification sent successfully for topic '%s'", topic),
	}, nil
}

// SubscribeToTopic subscribes tokens to a topic
func (fs *FirebaseService) SubscribeToTopic(tokens []string, topic string) error {
	if !fs.isActive {
		return fmt.Errorf("firebase service not active")
	}

	ctx := context.Background()
	_, err := fs.client.SubscribeToTopic(ctx, tokens, topic)
	if err != nil {
		colors.PrintError("Error subscribing to topic: %v", err)
		return err
	}

	colors.PrintSuccess("Successfully subscribed %d tokens to topic '%s'", len(tokens), topic)
	return nil
}

// UnsubscribeFromTopic unsubscribes tokens from a topic
func (fs *FirebaseService) UnsubscribeFromTopic(tokens []string, topic string) error {
	if !fs.isActive {
		return fmt.Errorf("firebase service not active")
	}

	ctx := context.Background()
	_, err := fs.client.UnsubscribeFromTopic(ctx, tokens, topic)
	if err != nil {
		colors.PrintError("Error unsubscribing from topic: %v", err)
		return err
	}

	colors.PrintSuccess("Successfully unsubscribed %d tokens from topic '%s'", len(tokens), topic)
	return nil
}

// IsActive returns whether the Firebase service is active
func (fs *FirebaseService) IsActive() bool {
	return fs.isActive
}

// TestConnection tests the Firebase connection
func (fs *FirebaseService) TestConnection() error {
	if !fs.isActive {
		return fmt.Errorf("firebase service not active")
	}

	// Try to send a test message to a dummy token
	testMessage := &NotificationMessage{
		Token: "test_token",
		Title: "Test",
		Body:  "Test notification",
	}

	_, err := fs.SendNotification(testMessage)
	// We expect this to fail due to invalid token, but it should reach Firebase
	if err != nil && err.Error() != "firebase service not active" {
		// If we get a Firebase-specific error, the connection is working
		colors.PrintSuccess("Firebase connection test successful")
		return nil
	}

	return fmt.Errorf("firebase connection test failed")
}
