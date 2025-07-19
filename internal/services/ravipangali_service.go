package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	// Add alarm-specific fields
	IsAlarm    bool `json:"is_alarm,omitempty"`   // Flag for alarm notifications
	Urgent     bool `json:"urgent,omitempty"`     // Flag for urgent notifications
	Persistent bool `json:"persistent,omitempty"` // Flag for persistent notifications
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

// SendPushNotification sends push notification via Ravipangali API
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

	// Validate FCM tokens before sending
	var validTokens []string
	var invalidTokens []string

	for i, token := range tokens {
		// FCM tokens should be at least 100 characters and contain only valid characters
		if len(token) < 100 {
			invalidTokens = append(invalidTokens, fmt.Sprintf("Token %d: too short (%d chars)", i+1, len(token)))
			continue
		}

		// Check for valid FCM token format (should contain only alphanumeric and some special chars)
		valid := true
		for _, char := range token {
			if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
				(char >= '0' && char <= '9') || char == ':' || char == '_' || char == '-') {
				invalidTokens = append(invalidTokens, fmt.Sprintf("Token %d: invalid character '%c'", i+1, char))
				valid = false
				break
			}
		}

		if valid {
			validTokens = append(validTokens, token)
		}
	}

	// Log token validation results
	colors.PrintInfo("FCM Token Validation Results:")
	colors.PrintInfo("  Total tokens: %d", len(tokens))
	colors.PrintInfo("  Valid tokens: %d", len(validTokens))
	colors.PrintInfo("  Invalid tokens: %d", len(invalidTokens))

	if len(invalidTokens) > 0 {
		for _, invalid := range invalidTokens {
			colors.PrintWarning("  %s", invalid)
		}
	}

	if len(validTokens) == 0 {
		return nil, fmt.Errorf("no valid FCM tokens found")
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
		Tokens:   validTokens, // Use only valid tokens
		ImageURL: imageURL,
		Data:     data,
		Priority: priority,
		Type:     notificationType,
		Sound:    sound,
		DataOnly: false, // Changed from true to false to allow Firebase to display notifications
	}

	// Handle alarm notifications specially
	if notificationType == "alarm" {
		payload.IsAlarm = true
		payload.Urgent = true
		payload.Persistent = true
		payload.Priority = "urgent" // Force urgent priority for alarms
		payload.Sound = "alarm"     // Force alarm sound for alarms
	} else if notificationType == "alert" {
		payload.Urgent = true
		payload.Priority = "high" // Force high priority for alerts
	}

	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		colors.PrintError("Failed to marshal payload to JSON: %v", err)
		return nil, fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Log the request details
	colors.PrintInfo("Sending push notification to Ravipangali API:")
	colors.PrintInfo("  Endpoint: %s", endpoint)
	colors.PrintInfo("  Title: %s", title)
	colors.PrintInfo("  Body: %s", body)
	colors.PrintInfo("  Tokens: %d", len(validTokens))
	colors.PrintInfo("  Priority: %s", priority)
	colors.PrintInfo("  Type: %s", notificationType)
	colors.PrintInfo("  Sound: %s", sound)
	colors.PrintInfo("  DataOnly: %t", payload.DataOnly)

	// Create HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		colors.PrintError("Failed to create HTTP request: %v", err)
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		colors.PrintError("Failed to send request to Ravipangali API: %v", err)
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		colors.PrintError("Failed to read response body: %v", err)
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Log the response
	colors.PrintInfo("Ravipangali API response:")
	colors.PrintInfo("  Status Code: %d", resp.StatusCode)
	colors.PrintInfo("  Response Body: %s", string(bodyBytes))

	// Parse response
	var response RavipangaliResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		colors.PrintError("Failed to parse Ravipangali API response: %v", err)
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Log detailed results
	colors.PrintInfo("  Success: %t", response.Success)
	colors.PrintInfo("  Message: %s", response.Message)
	colors.PrintInfo("  Tokens Sent: %d", response.TokensSent)
	colors.PrintInfo("  Tokens Delivered: %d", response.TokensDelivered)
	colors.PrintInfo("  Tokens Failed: %d", response.TokensFailed)

	// If there are failed tokens, log them
	if response.TokensFailed > 0 {
		colors.PrintWarning("❌   Tokens Failed: %d", response.TokensFailed)
		if len(response.Details) > 0 {
			for _, detail := range response.Details {
				if !detail.Success {
					colors.PrintWarning("    Failed token: %s", detail.Token[:20]+"...")
					colors.PrintWarning("    Response: %v", detail.Response)
				}
			}
		}
	}

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
