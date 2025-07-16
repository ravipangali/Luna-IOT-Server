# Firebase Notification Service Usage

This document explains how to use the new Firebase notification service dynamically in your Luna IoT application.

## Overview

The new Firebase service (`internal/services/firebase_service.go`) provides a clean, reusable interface for sending notifications using Firebase Cloud Messaging (FCM). It's based on the working `firebase.go` code but made dynamic and reusable.

## Basic Usage

### 1. Initialize Firebase Service

```go
import "luna_iot_server/internal/services"

// Initialize the Firebase service
firebaseService := services.NewFirebaseService()

// Check if Firebase is active
if !firebaseService.IsActive() {
    // Handle Firebase not being available
    return
}
```

### 2. Send Notification to Single Token

```go
notification := &services.NotificationMessage{
    Token:       "user_fcm_token_here",
    Title:       "Vehicle Alert",
    Body:        "Your vehicle has been started",
    Data:        map[string]string{"vehicle_id": "123", "action": "started"},
    ImageURL:    "https://example.com/icon.png",
    Sound:       "default",
    Priority:    "high",
    CollapseKey: "vehicle_alert",
}

response, err := firebaseService.SendNotification(notification)
if err != nil {
    // Handle error
    return
}

if response.Success {
    // Notification sent successfully
    fmt.Printf("Success: %s\n", response.Message)
} else {
    // Notification failed
    fmt.Printf("Failed: %s\n", response.Message)
}
```

### 3. Send to Multiple Tokens

```go
tokens := []string{"token1", "token2", "token3"}

response, err := firebaseService.SendToMultipleTokens(
    tokens,
    "System Alert",
    "Maintenance scheduled for tomorrow",
    map[string]string{"maintenance_type": "scheduled"},
)
```

### 4. Send to Topic

```go
response, err := firebaseService.SendToTopic(
    "alerts",
    "Emergency Alert",
    "All vehicles return to base immediately",
    map[string]string{"alert_type": "emergency"},
)
```

### 5. Subscribe/Unsubscribe from Topics

```go
// Subscribe users to a topic
tokens := []string{"user1_token", "user2_token"}
err := firebaseService.SubscribeToTopic(tokens, "vehicle_updates")

// Unsubscribe users from a topic
err := firebaseService.UnsubscribeFromTopic(tokens, "old_topic")
```

## Integration with Existing Notification Service

The existing `NotificationService` has been updated to use the new Firebase service internally:

```go
// Initialize notification service
notificationService := services.NewNotificationService()

// Send to user by ID (automatically fetches FCM token from database)
notificationData := &services.NotificationData{
    Title:    "Vehicle Status Update",
    Body:     "Your vehicle is now online",
    Type:     "vehicle_status",
    Data:     map[string]interface{}{"vehicle_id": "123"},
    ImageURL: "https://example.com/icon.png",
    Sound:    "notification.wav",
    Priority: "normal",
}

response, err := notificationService.SendToUser(userID, notificationData)
```

## Dynamic Notification Examples

### Vehicle Events

```go
func SendVehicleEventNotification(event string, vehicleID string, userTokens []string) {
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
        data = map[string]string{"event": "low_battery", "vehicle_id": vehicleID}
    case "overspeed":
        title = "Overspeed Alert"
        body = "Vehicle is exceeding speed limit"
        data = map[string]string{"event": "overspeed", "vehicle_id": vehicleID}
    }
    
    response, err := firebaseService.SendToMultipleTokens(userTokens, title, body, data)
    if err != nil {
        // Handle error
        return
    }
    
    if response.Success {
        fmt.Printf("Notification sent: %s\n", response.Message)
    }
}
```

### System Alerts

```go
func SendSystemAlert(alertType string, message string, userTokens []string) {
    firebaseService := services.NewFirebaseService()
    
    var priority string
    switch alertType {
    case "emergency":
        priority = "high"
    case "warning":
        priority = "normal"
    case "info":
        priority = "low"
    }
    
    notification := &services.NotificationMessage{
        Token:    userTokens[0], // Send to first user as example
        Title:    fmt.Sprintf("%s Alert", strings.Title(alertType)),
        Body:     message,
        Data:     map[string]string{"alert_type": alertType},
        Priority: priority,
        Sound:    "alert.wav",
    }
    
    response, err := firebaseService.SendNotification(notification)
    if err != nil {
        // Handle error
        return
    }
    
    if response.Success {
        fmt.Printf("System alert sent: %s\n", response.Message)
    }
}
```

## Testing

Run the test file to verify Firebase connectivity:

```bash
cd luna_iot_server
go run cmd/test-firebase/main.go
```

## Configuration

The Firebase service uses the `firebase_key.json` file for authentication. Make sure this file is present in the project root.

## Error Handling

The service provides detailed error messages and success/failure responses:

```go
response, err := firebaseService.SendNotification(notification)
if err != nil {
    // Network or Firebase API error
    fmt.Printf("Error: %v\n", err)
    return
}

if !response.Success {
    // Firebase returned an error (invalid token, etc.)
    fmt.Printf("Firebase error: %s\n", response.Error)
    return
}

// Success
fmt.Printf("Success: %s\n", response.Message)
```

## Features

- ✅ Dynamic token-based notifications
- ✅ Multicast notifications to multiple tokens
- ✅ Topic-based notifications
- ✅ Topic subscription management
- ✅ Rich notification data (images, sounds, priority)
- ✅ Error handling and status reporting
- ✅ Connection testing
- ✅ Integration with existing notification service

The new Firebase service is now ready to use dynamically throughout your application! 