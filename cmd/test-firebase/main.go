package main

import (
	"luna_iot_server/internal/services"
	"luna_iot_server/pkg/colors"
)

func main() {
	colors.PrintBanner()
	colors.PrintInfo("Testing Firebase Notification Service")

	// Initialize Firebase service
	firebaseService := services.NewFirebaseService()

	// Test if Firebase is active
	if !firebaseService.IsActive() {
		colors.PrintError("Firebase service is not active")
		return
	}

	colors.PrintSuccess("Firebase service is active")

	// Test connection
	if err := firebaseService.TestConnection(); err != nil {
		colors.PrintError("Firebase connection test failed: %v", err)
		return
	}

	// Example: Send notification to a specific token
	// Replace with actual FCM token
	testToken := "cSMQwCvFT8yZEP_mpqPsu_:APA91bFDpU_GXdsXXxamS8TpGjCybOEXqDzLyEo38z8W7nEfrgvspbe4RU8hAJZ5T7t7ectX76SaIbAxtcKpvSUk1_NwRSZFAzdN1T_UHo-jnw7vnHAW89o"

	notification := &services.NotificationMessage{
		Token:       testToken,
		Title:       "Dynamic Notification Test",
		Body:        "This is a test notification sent dynamically!",
		Data:        map[string]string{"test_key": "test_value", "timestamp": "2024"},
		ImageURL:    "",
		Sound:       "default",
		Priority:    "high",
		CollapseKey: "test_notification",
	}

	colors.PrintInfo("Sending test notification...")
	response, err := firebaseService.SendNotification(notification)
	if err != nil {
		colors.PrintError("Failed to send notification: %v", err)
		return
	}

	if response.Success {
		colors.PrintSuccess("Notification sent successfully: %s", response.Message)
	} else {
		colors.PrintError("Notification failed: %s", response.Message)
	}

	// Example: Send to multiple tokens
	tokens := []string{testToken}
	colors.PrintInfo("Sending multicast notification...")
	multicastResponse, err := firebaseService.SendToMultipleTokens(tokens, "Multicast Test", "Testing multiple tokens", map[string]string{"type": "multicast"})
	if err != nil {
		colors.PrintError("Failed to send multicast notification: %v", err)
		return
	}

	if multicastResponse.Success {
		colors.PrintSuccess("Multicast notification sent: %s", multicastResponse.Message)
	} else {
		colors.PrintError("Multicast notification failed: %s", multicastResponse.Message)
	}

	// Example: Send to topic
	colors.PrintInfo("Sending topic notification...")
	topicResponse, err := firebaseService.SendToTopic("test_topic", "Topic Test", "Testing topic notifications", map[string]string{"type": "topic"})
	if err != nil {
		colors.PrintError("Failed to send topic notification: %v", err)
		return
	}

	if topicResponse.Success {
		colors.PrintSuccess("Topic notification sent: %s", topicResponse.Message)
	} else {
		colors.PrintError("Topic notification failed: %s", topicResponse.Message)
	}

	colors.PrintSuccess("Firebase service test completed successfully!")
}
