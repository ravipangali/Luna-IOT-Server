package main

import (
	"luna_iot_server/internal/services"
	"luna_iot_server/pkg/colors"
)

// Example of how to use Firebase service dynamically in your controllers

func ExampleSendNotificationToUser() {
	// Initialize Firebase service
	firebaseService := services.NewFirebaseService()

	// Example: Send notification to a specific user
	userToken := "user_fcm_token_here"
	
	notification := &services.NotificationMessage{
		Token:    userToken,
		Title:    "Vehicle Alert",
		Body:     "Your vehicle has been started",
		Data:     map[string]string{"vehicle_id": "123", "action": "started"},
		Priority: "high",
		Sound:    "default",
	}

	response, err := firebaseService.SendNotification(notification)
	if err != nil {
		colors.PrintError("Failed to send notification: %v", err)
		return
	}

	if response.Success {
		colors.PrintSuccess("Notification sent: %s", response.Message)
	} else {
		colors.PrintError("Notification failed: %s", response.Message)
	}
}

func ExampleSendToMultipleUsers() {
	// Initialize Firebase service
	firebaseService := services.NewFirebaseService()

	// Example: Send to multiple users
	userTokens := []string{
		"cSMQwCvFT8yZEP_mpqPsu_:APA91bFDpU_GXdsXXxamS8TpGjCybOEXqDzLyEo38z8W7nEfrgvspbe4RU8hAJZ5T7t7ectX76SaIbAxtcKpvSUk1_NwRSZFAzdN1T_UHo-jnw7vnHAW89o",
	}

	response, err := firebaseService.SendToMultipleTokens(
		userTokens,
		"System Maintenance",
		"Scheduled maintenance will begin in 30 minutes",
		map[string]string{"maintenance_type": "scheduled", "duration": "2_hours"},
	)
	if err != nil {
		colors.PrintError("Failed to send multicast: %v", err)
		return
	}

	if response.Success {
		colors.PrintSuccess("Multicast sent: %s", response.Message)
	} else {
		colors.PrintError("Multicast failed: %s", response.Message)
	}
}

func ExampleSendToTopic() {
	// Initialize Firebase service
	firebaseService := services.NewFirebaseService()

	// Example: Send to a topic (all users subscribed to "alerts")
	response, err := firebaseService.SendToTopic(
		"alerts",
		"Emergency Alert",
		"All vehicles return to base immediately",
		map[string]string{"alert_type": "emergency", "priority": "critical"},
	)
	if err != nil {
		colors.PrintError("Failed to send topic notification: %v", err)
		return
	}

	if response.Success {
		colors.PrintSuccess("Topic notification sent: %s", response.Message)
	} else {
		colors.PrintError("Topic notification failed: %s", response.Message)
	}
}

func ExampleSubscribeToTopic() {
	// Initialize Firebase service
	firebaseService := services.NewFirebaseService()

	// Example: Subscribe users to a topic
	userTokens := []string{"user1_token", "user2_token", "user3_token"}
	
	err := firebaseService.SubscribeToTopic(userTokens, "vehicle_updates")
	if err != nil {
		colors.PrintError("Failed to subscribe to topic: %v", err)
		return
	}

	colors.PrintSuccess("Users subscribed to vehicle_updates topic")
}

func ExampleUnsubscribeFromTopic() {
	// Initialize Firebase service
	firebaseService := services.NewFirebaseService()

	// Example: Unsubscribe users from a topic
	userTokens := []string{"user1_token", "user2_token"}
	
	err := firebaseService.UnsubscribeFromTopic(userTokens, "old_topic")
	if err != nil {
		colors.PrintError("Failed to unsubscribe from topic: %v", err)
		return
	}

	colors.PrintSuccess("Users unsubscribed from old_topic")
}

// Example of how to integrate with your existing notification service
func ExampleWithNotificationService() {
	// Initialize the notification service (which uses Firebase internally)
	notificationService := services.NewNotificationService()

	// Example: Send notification to a user by ID
	userID := uint(1)
	notificationData := &services.NotificationData{
		Title:    "Vehicle Status Update",
		Body:     "Your vehicle is now online",
		Type:     "vehicle_status",
		Data:     map[string]interface{}{"vehicle_id": "123", "status": "online"},
		ImageURL: "https://example.com/vehicle_icon.png",
		Sound:    "notification.wav",
		Priority: "normal",
	}

	response, err := notificationService.SendToUser(userID, notificationData)
	if err != nil {
		colors.PrintError("Failed to send notification: %v", err)
		return
	}

	if response.Success {
		colors.PrintSuccess("User notification sent: %s", response.Message)
	} else {
		colors.PrintError("User notification failed: %s", response.Message)
	}
}

// Example of dynamic notification based on events
func ExampleDynamicNotification(event string, vehicleID string, userTokens []string) {
	firebaseService := services.NewFirebaseService()

	var title, body string
	var data map[string]string

	switch event {
	case "vehicle_started":
		title = "Vehicle Started"
		body = "Your vehicle has been started remotely"
		data = map[string]string{"event": "started", "vehicle_id": vehicleID}
	case "vehicle_stopped":
		title = "Vehicle Stopped"
		body = "Your vehicle has been stopped"
		data = map[string]string{"event": "stopped", "vehicle_id": vehicleID}
	case "low_battery":
		title = "Low Battery Alert"
		body = "Vehicle battery is running low"
		data = map[string]string{"event": "low_battery", "vehicle_id": vehicleID, "battery_level": "15%"}
	case "overspeed":
		title = "Overspeed Alert"
		body = "Vehicle is exceeding speed limit"
		data = map[string]string{"event": "overspeed", "vehicle_id": vehicleID, "speed": "85km/h"}
	default:
		title = "Vehicle Alert"
		body = "Vehicle status has changed"
		data = map[string]string{"event": event, "vehicle_id": vehicleID}
	}

	response, err := firebaseService.SendToMultipleTokens(userTokens, title, body, data)
	if err != nil {
		colors.PrintError("Failed to send dynamic notification: %v", err)
		return
	}

	if response.Success {
		colors.PrintSuccess("Dynamic notification sent: %s", response.Message)
	} else {
		colors.PrintError("Dynamic notification failed: %s", response.Message)
	}
} 