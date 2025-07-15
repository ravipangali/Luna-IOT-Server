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

func GetFirebaseConfig() *FirebaseConfig {
	return &FirebaseConfig{
		ProjectID:     "luna-iot-b5cdd",
		PrivateKeyID:  "499080667fce39b655a13c75ffe715ff94185f8d",
		PrivateKey:    "-----BEGIN PRIVATE KEY-----\nMIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQDgoxhyhYJUyBRn\nbwRk1oD8BkBw9bd9lh+q2b1IbaYfhhaLsx6P5IPkfF3YgMnAgwI0ojnrMl1Gl903\n7p59T1eGQnbwl4pVn3CI+OsF5GgqSMP2XPLahQy6uz0pQRG2LF/3u+64GsDb2nMJ\nlKsX5v+/n3gJgJbg/mvP3O7V6yycyedEUzFRs6KS/PyD/rQ7s2UZAZSZm7uMB8OC\nHIMdKdQPL3cseJxLNgeOgNivGN0V62brMYcSvaRmrl3cpbPyh58DVG5rLFIUfFoa\nVZ9EOqRg6BUNO9R1RL1waN8V0PJuIEUkrz/C33VpE6G7LuzX9ZXaKekVILPZoRdl\nvYNp+i39AgMBAAECggEAEdCAiu1EkZwP/8eBlf+e2TdW5/GZYur2liWDY2ZZFmBO\nktCaW8RX1ohewHsu9/nlWsg2U9ns8Y3vyLsMAMyiGJWJYJLoyqaEyNzSQprdAEsN\nMCrDeNa0h9OufRvzn1Qm98eeUSWQZgvaZwgBU4Y86Cgc35M98LGsjVHF83TQvZqg\nk1TS51cwvBhavsHgQPn84Hqb55lmgXdCcgU2qps80g+f71joDYifwiyS6nDFnmj3\n6pm6zDhmkqZVJMeg0CyGMD85aSFwDmuNAsi53eZlwwks50eLec0kCaO9/ElxF/6x\neKSd8/tzGaKobpYCCkDYlgiu4eSyi9TThD2Fm0FpnwKBgQD3UjhLsW8hbwLyXr05\nr68EU9sk+nd3j1+WZlo6KR1Y2dV3XNAwvrJ5OsKV8/CQmxV/p9crB0gRBdjVPk25\nUvm0PVax/DqP6s99ZRxSwphA6OFgnXmqd4WKLZDFsIgZytV2OaH8R4Uqb9LnEI6h\n5J+NvuqXxZ6q3ePiUQJ48wycUwKBgQDohRiLtJYMrzmVUFbZuO6R0PjsiQuGHD3n\ns8pocmQd6XrS6laIC++vEqT5E83+ggcR20ls4MxmcvXOhwbwGNnoHW9xBi29ERxa\nM5WQNn7bXfP1+rGFw2qsp4KkpFOPecFGwIyRGt6G/kTq5Wqi9XCjVDV5s7JD1BgR\nc1mk3VdCbwKBgHYq+azo1TFDSkQlkgHK+DN4IX/UkFo2zbQdqUSaumPmiMDkPrDb\nnIih07E0AaAGCUqaFguACiXgBk802owOojJFEHQwEIcM6SB/u/2q7nYtDupLs4MI\nYmy4ArEB/LVeHYnEVaolPfIdxcYTOiMOClH+gzYK/RmktSpADI9fiYnzAoGAMaG/\nVIrOgJSigPmuIDk2S0/E4pB6Mj0zBZM+AD9ymWPuALlekRmjJsafCj+s98d/hNM/\nAAuX9cJSL6xo0bUsRjyKPiDogHP3jlV2dlr7hw2t9nJ1lCzbR1FWNJiS8Yw2skiF\neK+4ki4SPeWMdo5XZbWi2IB/67SJEqiBmQxaBOcCgYBdEQE7EXYIZINLvJ52le9a\nLn2AxiSJTVWF4+6elclceF9mzQ9Bb8ZzhsybJfCSdHgxP37vHZKSIJHE6Znb4uhG\nuopTxNfsDYUjoLCusT/MpjZHF1pr+g1bW2bF4tpPDDpj5gEwLpwjbahV4oyveGAP\n4YuXI1tGM82qPSZH/nCaKA==\n-----END PRIVATE KEY-----\n",
		ClientEmail:   "firebase-adminsdk-fbsvc@luna-iot-b5cdd.iam.gserviceaccount.com",
		ClientID:      "116176236230697495561",
		AuthURI:       "https://accounts.google.com/o/oauth2/auth",
		TokenURI:      "https://oauth2.googleapis.com/token",
		AuthProvider:  "https://www.googleapis.com/oauth2/v1/certs",
		ClientCertURL: "https://www.googleapis.com/robot/v1/metadata/x509/firebase-adminsdk-fbsvc%40luna-iot-b5cdd.iam.gserviceaccount.com",
	}
}

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
