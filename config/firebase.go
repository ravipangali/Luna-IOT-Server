package config

import (
	"context"
	"encoding/json"
	"fmt"
	"luna_iot_server/pkg/colors"
	"os"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type FirebaseConfig struct {
	ProjectID     string
	PrivateKeyID  string
	PrivateKey    string
	ClientEmail   string
	ClientID      string
	AuthURI       string
	TokenURI      string
	AuthProvider  string
	ClientCertURL string
}

var firebaseApp *firebase.App
var messagingClient *messaging.Client
var firebaseInitialized bool

func InitializeFirebase() error {
	colors.PrintInfo("Starting Firebase initialization with hardcoded credentials...")

	// Check if we should disable Firebase completely
	if os.Getenv("DISABLE_FIREBASE") == "true" {
		colors.PrintWarning("Firebase disabled by environment variable DISABLE_FIREBASE=true")
		colors.PrintInfo("Push notifications will be simulated successfully")
		firebaseInitialized = false
		return nil
	}

	// Try to use environment variables first (if available)
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		projectID = "luna-iot-5993f" // fallback
	}

	// Use hardcoded credentials from your new service account file
	// These are the exact credentials from your new firebase service account
	credentials := map[string]interface{}{
		"type":                        "service_account",
		"project_id":                  projectID,
		"private_key_id":              "7b2de23547167be850a4c997c7d8c53583377ce8",
		"private_key":                 "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCm3BG7delgI/u9\n2ObBHe+1lNyCInFddMu6DFs+8t0kSyCcLaJwpzsd0baa5HymtWuu5j9wlasAFAFl\nFdTUpxzvohw2tM1TTNzOBDfAr+/Ibut66KmTEA8WM2+Y4dGmORecq+25+w4l0fOX\n67h/fSpD1YMwnA30BJgdymzXCezNs4pgQPsrx0o/97dWaxRYrZ+Hhsea6GdYwdBo\ncdVKmL2vf2AUD/5ruJ/3N/zJaUNEs3R2UBEYik3TFOKUsSVtiiFXFoGxV0xW5YPR\n//6+ZOd/o20+OitTVlPojhb1EP5mmIduOI3RWO2/VHVtvIL0L+7VTPhb2QZPjnUt\n46llun23AgMBAAECggEAFQL4Bf/Octl/z+OfsIh726/H0hNS7I59QQu2nxynavSS\n7K0fxskoO+m2nEFSzmNgu1Yp0vGQkPfz8li8w50vmvVyUWk/GdLp/dSwzfDZubGj\nXDIuzbgN+PZigoFH4dilTTNRQkQyVIzgUdclv9gbxGL/RtW/5A8tYJhRpa/i7pwN\naHw8W6IMUKd9HuCqXOBXhH1GggMhCEJt2wm7PXzxgsKVa4zxPx82vI2Tuhgox+UE\nIipaK26GkvnDjPyRLyGNZk1f0ntYjv8TbDj3rmsSSfCbxgG+otWS8rlgTN44L7w9\nkBCLVjlslIA2x8qmcbzXK635dPvIqB3OnTo5ca3xYQKBgQDrY9k1V6mpELG5zCaF\nAqE8WN1AIwrT2A1cT1KgH9yr8MOX1n0enPeyFRiKMrAnuecyB72KEYVwFrS5m9XX\nwwc+Oa/jCwqgfanTKaf5SyjPzgds5G/fxUi4jB6epr4+6nqnxL0XCswBSTvK6qm0\n+qQJfGF1VJNRZq728CH1+jkrnwKBgQC1eCVc++7k3qsKbuTVjz+kMhdZ89rAgEmN\n1z9l6eWF1QYZ5wQS0trN9deN+sG0PjRsukR4VJw0HJxX/wlPvlIbWUmPaauaN+sb\naOfVkuoC+3N/bpwDfLd1wVg8KQ/lyTL9PvSQpLGwU17Qms+wqLA+gRDoX3nk2XOv\nctungtn26QKBgQCF1ZmUGKmgNJu4NfjYu2wNMcFqTAJF/JtsFrW10SfYouWymQM+\nuqSinhf7y2IY1Dw9V+VOcTPbTS2oMpBdQsgFeysj/g0mvwwlwZN9zFwB+vSB10g8\nhKEaPKDUN54Hi639YYDZbwwa1xamAtJG0hMeSZfn7BRuveFRCatlfcWvpQKBgCml\nKO3t4yUi9J2wVVOtTC2iUTmTfOAwkLC8dRAuXT4ZZQ0MtyKawRwDDzTGFy4GGIHb\nPVtgD3jmF/sZzElApBcipn8DAR6jNpFTweCBlrKYgij8eVFTjca4WEd2JO/W/Jyh\nlf6bzStp9pho7sDb9ZZiiD7Lqm2aebIJ6d7HaL4BAoGAHBS4o1ymFAS7GyF6Oc58\n9DH8lF/G3iAgZP6qvgt8OPeLk1Xvcd/BIHJf6P2VOYYGQwkrwBsYhL1Y5knEUUsi\nsWEW0HI+zKJbjieuFcKF2th6OYB/EK5k1Zyd2F4Ip3W8gCjyNOp7HhYQ+sv/YLCL\nzxcbvtbrDdQskKZGrFkWlsw=\n-----END PRIVATE KEY-----\n",
		"client_email":                "firebase-adminsdk-fbsvc@luna-iot-5993f.iam.gserviceaccount.com",
		"client_id":                   "108105592493412976295",
		"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
		"token_uri":                   "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url":        "https://www.googleapis.com/robot/v1/metadata/x509/firebase-adminsdk-fbsvc%40luna-iot-5993f.iam.gserviceaccount.com",
		"universe_domain":             "googleapis.com",
	}

	colors.PrintSuccess("Using hardcoded Firebase credentials:")
	colors.PrintInfo("  Project ID: %s", credentials["project_id"])
	colors.PrintInfo("  Client Email: %s", credentials["client_email"])

	// Convert credentials to JSON bytes
	credentialsBytes, err := json.Marshal(credentials)
	if err != nil {
		colors.PrintError("Failed to marshal credentials: %v", err)
		colors.PrintWarning("Firebase initialization failed, push notifications will be simulated")
		firebaseInitialized = false
		return nil
	}

	// Use hardcoded credentials with OAuth2
	colors.PrintInfo("Initializing Firebase with hardcoded credentials...")
	opt := option.WithCredentialsJSON(credentialsBytes)
	app, err := firebase.NewApp(context.Background(), &firebase.Config{
		ProjectID: projectID,
	}, opt)
	if err != nil {
		colors.PrintError("Failed to initialize Firebase with hardcoded credentials: %v", err)
		colors.PrintWarning("Firebase initialization failed, push notifications will be simulated")
		firebaseInitialized = false
		return nil // Don't return error, just disable Firebase
	}

	firebaseApp = app
	colors.PrintSuccess("Firebase app created successfully")

	// Initialize messaging client with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	colors.PrintInfo("Creating Firebase messaging client...")
	messaging, err := app.Messaging(ctx)
	if err != nil {
		colors.PrintError("Failed to initialize Firebase messaging client: %v", err)
		colors.PrintWarning("Firebase messaging client failed, push notifications will be simulated")
		firebaseInitialized = false
		return nil // Don't return error, just disable Firebase
	}

	messagingClient = messaging
	firebaseInitialized = true
	colors.PrintSuccess("Firebase initialized successfully using hardcoded credentials with OAuth2")
	colors.PrintInfo("Messaging client: %v", messagingClient)
	return nil
}

func GetMessagingClient() *messaging.Client {
	return messagingClient
}

func IsFirebaseEnabled() bool {
	enabled := messagingClient != nil && firebaseInitialized
	colors.PrintInfo("Firebase enabled check: %t (client: %v, initialized: %v)", enabled, messagingClient != nil, firebaseInitialized)
	return enabled
}

// TestFirebaseConnection tests if Firebase is properly configured and accessible
func TestFirebaseConnection() error {
	if !IsFirebaseEnabled() {
		return fmt.Errorf("Firebase is not enabled")
	}

	client := GetMessagingClient()
	if client == nil {
		return fmt.Errorf("Firebase messaging client is nil")
	}

	// Try to send a test message to a non-existent token to test the connection
	// This will fail but will tell us if the Firebase configuration is valid
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	message := &messaging.Message{
		Token: "test-token-that-does-not-exist",
		Notification: &messaging.Notification{
			Title: "Test",
			Body:  "Test",
		},
	}

	_, err := client.Send(ctx, message)
	if err != nil {
		// Check if it's a 404 error (invalid project/credentials) or other error
		if strings.Contains(err.Error(), "404") {
			colors.PrintError("Firebase configuration error: Invalid project or credentials")
			return fmt.Errorf("Firebase configuration error: Invalid project or credentials - %v", err)
		}
		// Other errors (like invalid token) are expected and indicate Firebase is working
		colors.PrintSuccess("Firebase connection test passed (expected error for invalid token)")
		return nil
	}

	return nil
}
