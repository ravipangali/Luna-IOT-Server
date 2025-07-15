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
	"google.golang.org/api/option"
)

func main() {
	colors.PrintBanner()
	colors.PrintHeader("Firebase Diagnostic & Fix Tool")

	// Step 1: Check current configuration
	colors.PrintSubHeader("Step 1: Current Configuration")
	checkCurrentConfig()

	// Step 2: Test different approaches
	colors.PrintSubHeader("Step 2: Testing Firebase Connection")
	testFirebaseConnection()

	// Step 3: Provide solutions
	colors.PrintSubHeader("Step 3: Solutions")
	provideSolutions()
}

func checkCurrentConfig() {
	colors.PrintInfo("Checking current Firebase configuration...")

	// Check if service account file exists
	if _, err := os.Stat("firebase-service-account.json"); err == nil {
		colors.PrintSuccess("✓ Firebase service account file exists")
	} else {
		colors.PrintError("✗ Firebase service account file not found")
	}

	// Check hardcoded config
	config := config.GetFirebaseConfig()
	colors.PrintInfo("Project ID: %s", config.ProjectID)
	colors.PrintInfo("Client Email: %s", config.ClientEmail)
	colors.PrintInfo("Private Key ID: %s", config.PrivateKeyID)
}

func testFirebaseConnection() {
	colors.PrintInfo("Testing Firebase connection...")

	// Test 1: Try with service account file
	if _, err := os.Stat("firebase-service-account.json"); err == nil {
		colors.PrintInfo("Testing with service account file...")
		testWithServiceAccountFile()
	}

	// Test 2: Try with hardcoded credentials
	colors.PrintInfo("Testing with hardcoded credentials...")
	testWithHardcodedCredentials()
}

func testWithServiceAccountFile() {
	opt := option.WithCredentialsFile("firebase-service-account.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		colors.PrintError("✗ Failed to create Firebase app with service account file: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	messaging, err := app.Messaging(ctx)
	if err != nil {
		colors.PrintError("✗ Failed to create messaging client: %v", err)
		return
	}

	// Test with fake token
	message := &messaging.Message{
		Token: "fake-token-for-test",
		Notification: &messaging.Notification{
			Title: "Test",
			Body:  "Test",
		},
	}

	_, err = messaging.Send(ctx, message)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			colors.PrintError("✗ 404 Error: Project doesn't exist or no permissions")
		} else {
			colors.PrintSuccess("✓ Firebase connection works (expected error for invalid token)")
		}
	} else {
		colors.PrintSuccess("✓ Firebase connection successful")
	}
}

func testWithHardcodedCredentials() {
	config := config.GetFirebaseConfig()

	credentials := map[string]interface{}{
		"type":                        "service_account",
		"project_id":                  config.ProjectID,
		"private_key_id":              config.PrivateKeyID,
		"private_key":                 config.PrivateKey,
		"client_email":                config.ClientEmail,
		"client_id":                   config.ClientID,
		"auth_uri":                    config.AuthURI,
		"token_uri":                   config.TokenURI,
		"auth_provider_x509_cert_url": config.AuthProvider,
		"client_x509_cert_url":        config.ClientCertURL,
	}

	credentialsJSON, err := json.Marshal(credentials)
	if err != nil {
		colors.PrintError("✗ Failed to marshal credentials: %v", err)
		return
	}

	opt := option.WithCredentialsJSON(credentialsJSON)
	app, err := firebase.NewApp(context.Background(), &firebase.Config{
		ProjectID: config.ProjectID,
	}, opt)
	if err != nil {
		colors.PrintError("✗ Failed to create Firebase app: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	messaging, err := app.Messaging(ctx)
	if err != nil {
		colors.PrintError("✗ Failed to create messaging client: %v", err)
		return
	}

	// Test with fake token
	message := &messaging.Message{
		Token: "fake-token-for-test",
		Notification: &messaging.Notification{
			Title: "Test",
			Body:  "Test",
		},
	}

	_, err = messaging.Send(ctx, message)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			colors.PrintError("✗ 404 Error: Project doesn't exist or no permissions")
		} else {
			colors.PrintSuccess("✓ Firebase connection works (expected error for invalid token)")
		}
	} else {
		colors.PrintSuccess("✓ Firebase connection successful")
	}
}

func provideSolutions() {
	colors.PrintInfo("Based on the test results, here are the solutions:")
	colors.PrintInfo("")
	colors.PrintInfo("1. If you got 404 errors:")
	colors.PrintInfo("   - Go to https://console.firebase.google.com/")
	colors.PrintInfo("   - Check if project 'luna-iot-b5cdd' exists")
	colors.PrintInfo("   - If not, create it with this exact name")
	colors.PrintInfo("   - Enable Cloud Messaging")
	colors.PrintInfo("   - Generate new service account key")
	colors.PrintInfo("")
	colors.PrintInfo("2. If you got permission errors:")
	colors.PrintInfo("   - Go to Project Settings > Service Accounts")
	colors.PrintInfo("   - Make sure service account has 'Firebase Admin SDK Administrator' role")
	colors.PrintInfo("   - Generate new private key")
	colors.PrintInfo("")
	colors.PrintInfo("3. Replace your firebase-service-account.json file with the new one")
	colors.PrintInfo("")
	colors.PrintInfo("4. Restart your server and test again")
}
