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
	colors.PrintInfo("Starting Firebase initialization with OAuth2 authentication...")

	// Try multiple possible paths for the service account file
	possiblePaths := []string{
		"firebase-service-account.json",                            // Current directory
		"/home/luna/Luna-IOT-Server/firebase-service-account.json", // Absolute path
		"./firebase-service-account.json",                          // Explicit current directory
	}

	var serviceAccountPath string
	var fileExists bool

	colors.PrintInfo("Searching for Firebase service account file...")
	for _, path := range possiblePaths {
		colors.PrintInfo("Checking path: %s", path)
		if _, err := os.Stat(path); err == nil {
			serviceAccountPath = path
			fileExists = true
			colors.PrintSuccess("Found Firebase service account file at: %s", path)
			break
		} else {
			colors.PrintInfo("Path not found: %s (error: %v)", path, err)
		}
	}

	if !fileExists {
		colors.PrintError("Firebase service account file not found in any of these locations:")
		for _, path := range possiblePaths {
			colors.PrintError("  - %s", path)
		}
		colors.PrintWarning("Firebase initialization failed, push notifications will be disabled")
		firebaseInitialized = false
		return nil
	}

	// Read and validate the service account file
	colors.PrintInfo("Reading service account file: %s", serviceAccountPath)
	fileBytes, err := os.ReadFile(serviceAccountPath)
	if err != nil {
		colors.PrintError("Failed to read service account file: %v", err)
		colors.PrintWarning("Firebase initialization failed, push notifications will be disabled")
		firebaseInitialized = false
		return nil
	}

	colors.PrintInfo("Service account file read successfully, size: %d bytes", len(fileBytes))

	// Parse the JSON to validate it
	var serviceAccount map[string]interface{}
	if err := json.Unmarshal(fileBytes, &serviceAccount); err != nil {
		colors.PrintError("Failed to parse service account JSON: %v", err)
		colors.PrintWarning("Firebase initialization failed, push notifications will be disabled")
		firebaseInitialized = false
		return nil
	}

	// Validate required fields
	projectID, ok := serviceAccount["project_id"].(string)
	if !ok || projectID == "" {
		colors.PrintError("Invalid project_id in service account file")
		colors.PrintWarning("Firebase initialization failed, push notifications will be disabled")
		firebaseInitialized = false
		return nil
	}

	clientEmail, ok := serviceAccount["client_email"].(string)
	if !ok || clientEmail == "" {
		colors.PrintError("Invalid client_email in service account file")
		colors.PrintWarning("Firebase initialization failed, push notifications will be disabled")
		firebaseInitialized = false
		return nil
	}

	colors.PrintSuccess("Service account file validation passed:")
	colors.PrintInfo("  Project ID: %s", projectID)
	colors.PrintInfo("  Client Email: %s", clientEmail)

	// Use service account file with OAuth2
	colors.PrintInfo("Initializing Firebase with service account file...")
	opt := option.WithCredentialsFile(serviceAccountPath)
	app, err := firebase.NewApp(context.Background(), &firebase.Config{
		ProjectID: projectID,
	}, opt)
	if err != nil {
		colors.PrintError("Failed to initialize Firebase with service account file: %v", err)
		colors.PrintWarning("Firebase initialization failed, push notifications will be disabled")
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
		colors.PrintWarning("Firebase messaging client failed, push notifications will be disabled")
		firebaseInitialized = false
		return nil // Don't return error, just disable Firebase
	}

	messagingClient = messaging
	firebaseInitialized = true
	colors.PrintSuccess("Firebase initialized successfully using service account file with OAuth2")
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
