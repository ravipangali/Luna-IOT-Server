package main

import (
	"context"
	"encoding/json"
	"luna_iot_server/config"
	"luna_iot_server/pkg/colors"
	"os"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

func main() {
	colors.PrintBanner()
	colors.PrintHeader("Firebase Configuration Test")

	// Test 1: Check if service account file exists
	colors.PrintSubHeader("Test 1: Service Account File")
	serviceAccountPath := "firebase-service-account.json"
	if _, err := os.Stat(serviceAccountPath); err == nil {
		colors.PrintSuccess("✓ Service account file found: %s", serviceAccountPath)
	} else {
		colors.PrintWarning("✗ Service account file not found: %s", serviceAccountPath)
		colors.PrintInfo("   Error: %v", err)
	}

	// Test 2: Try to initialize Firebase
	colors.PrintSubHeader("Test 2: Firebase Initialization")

	// Try with service account file first
	if _, err := os.Stat(serviceAccountPath); err == nil {
		colors.PrintInfo("Attempting to initialize Firebase with service account file...")

		opt := option.WithCredentialsFile(serviceAccountPath)
		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			colors.PrintError("✗ Failed to initialize Firebase app: %v", err)
		} else {
			colors.PrintSuccess("✓ Firebase app created successfully")

			// Test messaging client
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			messaging, err := app.Messaging(ctx)
			if err != nil {
				colors.PrintError("✗ Failed to initialize messaging client: %v", err)
			} else {
				colors.PrintSuccess("✓ Messaging client initialized successfully")

				// Test 3: Test Firebase connection
				colors.PrintSubHeader("Test 3: Firebase Connection Test")
				testFirebaseConnection(messaging)
			}
		}
	} else {
		// Try with hardcoded credentials
		colors.PrintInfo("Attempting to initialize Firebase with hardcoded credentials...")

		firebaseConfig := config.GetFirebaseConfig()
		credentials := map[string]interface{}{
			"type":                        "service_account",
			"project_id":                  firebaseConfig.ProjectID,
			"private_key_id":              firebaseConfig.PrivateKeyID,
			"private_key":                 firebaseConfig.PrivateKey,
			"client_email":                firebaseConfig.ClientEmail,
			"client_id":                   firebaseConfig.ClientID,
			"auth_uri":                    firebaseConfig.AuthURI,
			"token_uri":                   firebaseConfig.TokenURI,
			"auth_provider_x509_cert_url": firebaseConfig.AuthProvider,
			"client_x509_cert_url":        firebaseConfig.ClientCertURL,
		}

		credentialsJSON, err := json.Marshal(credentials)
		if err != nil {
			colors.PrintError("✗ Failed to marshal credentials: %v", err)
		} else {
			opt := option.WithCredentialsJSON(credentialsJSON)
			app, err := firebase.NewApp(context.Background(), &firebase.Config{
				ProjectID: firebaseConfig.ProjectID,
			}, opt)

			if err != nil {
				colors.PrintError("✗ Failed to initialize Firebase app: %v", err)
			} else {
				colors.PrintSuccess("✓ Firebase app created successfully")

				// Test messaging client
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				messaging, err := app.Messaging(ctx)
				if err != nil {
					colors.PrintError("✗ Failed to initialize messaging client: %v", err)
				} else {
					colors.PrintSuccess("✓ Messaging client initialized successfully")

					// Test 3: Test Firebase connection
					colors.PrintSubHeader("Test 3: Firebase Connection Test")
					testFirebaseConnection(messaging)
				}
			}
		}
	}

	colors.PrintSubHeader("Test Summary")
	colors.PrintInfo("If you see 404 errors, it means:")
	colors.PrintInfo("1. The Firebase project doesn't exist")
	colors.PrintInfo("2. The service account credentials are invalid")
	colors.PrintInfo("3. The service account doesn't have the required permissions")
	colors.PrintInfo("")
	colors.PrintInfo("To fix this:")
	colors.PrintInfo("1. Go to https://console.firebase.google.com/")
	colors.PrintInfo("2. Create or select your project")
	colors.PrintInfo("3. Go to Project Settings > Service Accounts")
	colors.PrintInfo("4. Generate a new private key")
	colors.PrintInfo("5. Replace the firebase-service-account.json file")
}

func testFirebaseConnection(client *messaging.Client) {
	colors.PrintInfo("Testing Firebase connection by sending a test message...")

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
		if strings.Contains(err.Error(), "404") {
			colors.PrintError("✗ Firebase configuration error: Invalid project or credentials")
			colors.PrintError("   Error: %v", err)
			colors.PrintInfo("   This indicates the Firebase project or service account is invalid")
		} else if strings.Contains(err.Error(), "InvalidRegistration") || strings.Contains(err.Error(), "not a valid FCM registration token") {
			colors.PrintSuccess("✓ Firebase connection test passed!")
			colors.PrintInfo("   The error is expected (invalid token) and indicates Firebase is working")
			colors.PrintInfo("   Your Firebase configuration is correct!")
		} else {
			colors.PrintWarning("? Unexpected error: %v", err)
			colors.PrintInfo("   This might indicate a network or configuration issue")
		}
	} else {
		colors.PrintSuccess("✓ Firebase connection test passed!")
	}
}
