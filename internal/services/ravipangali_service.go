package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"luna_iot_server/pkg/colors"
)

// RavipangaliService handles communication with the Ravipangali API
type RavipangaliService struct{}

// NewRavipangaliService creates a new Ravipangali service instance
func NewRavipangaliService() *RavipangaliService {
	return &RavipangaliService{}
}

// RavipangaliPayload represents the payload sent to Ravipangali API
type RavipangaliPayload struct {
	Email    string                 `json:"email"`
	Password string                 `json:"password"`
	Title    string                 `json:"title"`
	Body     string                 `json:"body"`
	Tokens   []string               `json:"tokens"`
	ImageURL string                 `json:"image_url,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Priority string                 `json:"priority"`
	Type     string                 `json:"type,omitempty"`  // Add notification type
	Sound    string                 `json:"sound,omitempty"` // Add notification sound
	// Add flag to send only data payload (no notification payload)
	DataOnly bool `json:"data_only,omitempty"`
}

// RavipangaliResponse represents the response from Ravipangali API
type RavipangaliResponse struct {
	Success         bool     `json:"success"`
	Message         string   `json:"message,omitempty"`
	Error           string   `json:"error,omitempty"`
	NotificationID  string   `json:"notification_id,omitempty"`
	TokensSent      int      `json:"tokens_sent,omitempty"`
	TokensDelivered int      `json:"tokens_delivered,omitempty"`
	TokensFailed    int      `json:"tokens_failed,omitempty"`
	Details         []Detail `json:"details,omitempty"`
}

// Detail represents individual token delivery details
type Detail struct {
	Token    string      `json:"token"`
	Success  bool        `json:"success"`
	Response interface{} `json:"response"`
}

// SendPushNotification sends a push notification via Ravipangali API
func (rs *RavipangaliService) SendPushNotification(
	title, body string,
	tokens []string,
	imageURL string,
	data map[string]interface{},
	priority string,
	notificationType string,
	sound string,
) (*RavipangaliResponse, error) {
	// Get configuration from environment variables
	appID := os.Getenv("RP_FIREBASE_APP_ID")
	email := os.Getenv("RP_ACCOUNT_EMAIL")
	password := os.Getenv("RP_ACCOUNT_PASSWORD")

	// Validate required configuration
	if appID == "" {
		return nil, fmt.Errorf("RP_FIREBASE_APP_ID environment variable is not set")
	}
	if email == "" {
		return nil, fmt.Errorf("RP_ACCOUNT_EMAIL environment variable is not set")
	}
	if password == "" {
		return nil, fmt.Errorf("RP_ACCOUNT_PASSWORD environment variable is not set")
	}

	// Validate required parameters
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if body == "" {
		return nil, fmt.Errorf("body is required")
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("at least one FCM token is required")
	}

	// Set default priority if not provided
	if priority == "" {
		priority = "normal"
	}

	// Construct the API endpoint
	endpoint := fmt.Sprintf("https://ravipangali.com.np/user/api/firebase/apps/%s/notifications/", appID)

	// Prepare the payload
	payload := RavipangaliPayload{
		Email:    email,
		Password: password,
		Title:    title,
		Body:     body,
		Tokens:   tokens,
		ImageURL: imageURL,
		Data:     data,
		Priority: priority,
		Type:     notificationType,
		Sound:    sound,
		DataOnly: true, // Send only data payload to prevent Firebase automatic display
	}

	// If DataOnly is true, include notification content in data payload
	if payload.DataOnly {
		if payload.Data == nil {
			payload.Data = make(map[string]interface{})
		}
		// Include notification content in data payload
		payload.Data["title"] = title
		payload.Data["body"] = body
		payload.Data["image_url"] = imageURL
		payload.Data["priority"] = priority
		payload.Data["type"] = notificationType
		payload.Data["sound"] = sound
		// Keep original data fields
		for key, value := range data {
			payload.Data[key] = value
		}
	}

	// Marshal payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		colors.PrintError("Failed to marshal Ravipangali payload: %v", err)
		return nil, fmt.Errorf("failed to prepare notification payload: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		colors.PrintError("Failed to create HTTP request: %v", err)
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Luna-IOT-Server/1.0")

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Log the request (without sensitive data)
	colors.PrintInfo("Sending push notification to Ravipangali API:")
	colors.PrintInfo("  Endpoint: %s", endpoint)
	colors.PrintInfo("  Title: %s", title)
	colors.PrintInfo("  Body: %s", body)
	colors.PrintInfo("  Tokens: %d", len(tokens))
	colors.PrintInfo("  Priority: %s", priority)
	if imageURL != "" {
		colors.PrintInfo("  Image URL: %s", imageURL)
	}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		colors.PrintError("Failed to send request to Ravipangali API: %v", err)
		return nil, fmt.Errorf("failed to send request to Ravipangali API: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	var response RavipangaliResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		colors.PrintError("Failed to decode Ravipangali API response: %v", err)
		return nil, fmt.Errorf("failed to decode API response: %v", err)
	}

	// Log the response
	colors.PrintInfo("Ravipangali API response:")
	colors.PrintInfo("  Status Code: %d", resp.StatusCode)
	colors.PrintInfo("  Success: %t", response.Success)
	colors.PrintInfo("  Message: %s", response.Message)
	if response.Error != "" {
		colors.PrintError("  Error: %s", response.Error)
	}
	if response.TokensSent > 0 {
		colors.PrintInfo("  Tokens Sent: %d", response.TokensSent)
	}
	if response.TokensDelivered > 0 {
		colors.PrintSuccess("  Tokens Delivered: %d", response.TokensDelivered)
	}
	if response.TokensFailed > 0 {
		colors.PrintError("  Tokens Failed: %d", response.TokensFailed)
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		colors.PrintError("Ravipangali API returned non-200 status code: %d", resp.StatusCode)
		return &response, fmt.Errorf("Ravipangali API returned status code: %d", resp.StatusCode)
	}

	return &response, nil
}

// GetUserFCMTokens retrieves FCM tokens for the given user IDs
func (rs *RavipangaliService) GetUserFCMTokens(userIDs []uint) ([]string, error) {
	// This function would typically query your database to get FCM tokens
	// For now, we'll implement a placeholder that should be replaced with actual database query
	// This should be implemented in the notification service or database service

	// Placeholder implementation - replace with actual database query
	var tokens []string
	// TODO: Implement database query to get FCM tokens for userIDs
	// Example:
	// var users []models.User
	// if err := db.GetDB().Where("id IN ?", userIDs).Find(&users).Error; err != nil {
	//     return nil, err
	// }
	// for _, user := range users {
	//     if user.FCMToken != "" {
	//         tokens = append(tokens, user.FCMToken)
	//     }
	// }

	return tokens, nil
}
