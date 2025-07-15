package main

import (
	"context"
	"encoding/json"
	"luna_iot_server/config"
	"luna_iot_server/pkg/colors"
	"time"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

func main() {
	colors.PrintBanner()
	colors.PrintHeader("Firebase OAuth2 Test")

	// Test OAuth2 authentication
	colors.PrintInfo("Testing Firebase OAuth2 authentication...")

	config := config.GetFirebaseConfig()

	// Create credentials with OAuth2
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
		colors.PrintError("Failed to marshal credentials: %v", err)
		return
	}

	// Initialize Firebase with OAuth2
	opt := option.WithCredentialsJSON(credentialsJSON)
	app, err := firebase.NewApp(context.Background(), &firebase.Config{
		ProjectID: config.ProjectID,
	}, opt)

	if err != nil {
		colors.PrintError("Failed to create Firebase app: %v", err)
		return
	}

	colors.PrintSuccess("Firebase app created successfully")

	// Initialize messaging client
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messaging, err := app.Messaging(ctx)
	if err != nil {
		colors.PrintError("Failed to create messaging client: %v", err)
		return
	}

	colors.PrintSuccess("Messaging client created successfully")

	// Test with a fake token to check OAuth2 authentication
	message := &messaging.Message{
		Token: "fake-token-for-oauth2-test",
		Notification: &messaging.Notification{
			Title: "OAuth2 Test",
			Body:  "Testing OAuth2 authentication",
		},
	}

	_, err = messaging.Send(ctx, message)
	if err != nil {
		if err.Error() == "The registration token is not a valid FCM registration token" {
			colors.PrintSuccess("✅ OAuth2 authentication successful! (Expected error for invalid token)")
			colors.PrintInfo("Your Firebase configuration is working correctly with OAuth2")
		} else {
			colors.PrintError("❌ OAuth2 authentication failed: %v", err)
		}
	} else {
		colors.PrintSuccess("✅ OAuth2 authentication successful!")
	}
}
