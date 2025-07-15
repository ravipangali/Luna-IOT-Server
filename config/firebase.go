package config

import (
	"context"
	"encoding/json"
	"log"
	"os"

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

func GetFirebaseConfig() *FirebaseConfig {
	return &FirebaseConfig{
		ProjectID:     getEnv("FIREBASE_PROJECT_ID", ""),
		PrivateKeyID:  getEnv("FIREBASE_PRIVATE_KEY_ID", ""),
		PrivateKey:    getEnv("FIREBASE_PRIVATE_KEY", ""),
		ClientEmail:   getEnv("FIREBASE_CLIENT_EMAIL", ""),
		ClientID:      getEnv("FIREBASE_CLIENT_ID", ""),
		AuthURI:       getEnv("FIREBASE_AUTH_URI", "https://accounts.google.com/o/oauth2/auth"),
		TokenURI:      getEnv("FIREBASE_TOKEN_URI", "https://oauth2.googleapis.com/token"),
		AuthProvider:  getEnv("FIREBASE_AUTH_PROVIDER", "https://www.googleapis.com/oauth2/v1/certs"),
		ClientCertURL: getEnv("FIREBASE_CLIENT_CERT_URL", "https://www.googleapis.com/robot/v1/metadata/x509/firebase-adminsdk"),
	}
}

func InitializeFirebase() error {
	// Try to read from service account file first
	serviceAccountPath := "firebase-service-account.json"

	// Check if service account file exists
	if _, err := os.Stat(serviceAccountPath); err == nil {
		// Use service account file
		opt := option.WithCredentialsFile(serviceAccountPath)
		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			log.Printf("Failed to initialize Firebase with service account file: %v", err)
			return err
		}

		firebaseApp = app

		// Initialize messaging client
		messaging, err := app.Messaging(context.Background())
		if err != nil {
			return err
		}

		messagingClient = messaging
		log.Println("Firebase initialized successfully using service account file")
		return nil
	}

	// Fallback to environment variables
	config := GetFirebaseConfig()

	if config.ProjectID == "" {
		log.Println("Firebase not configured, push notifications will be disabled")
		return nil
	}

	// Create Firebase credentials
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

	// Convert credentials to JSON bytes
	credentialsJSON, err := json.Marshal(credentials)
	if err != nil {
		return err
	}

	// Initialize Firebase app
	opt := option.WithCredentialsJSON(credentialsJSON)
	app, err := firebase.NewApp(context.Background(), &firebase.Config{
		ProjectID: config.ProjectID,
	}, opt)

	if err != nil {
		return err
	}

	firebaseApp = app

	// Initialize messaging client
	messaging, err := app.Messaging(context.Background())
	if err != nil {
		return err
	}

	messagingClient = messaging
	log.Println("Firebase initialized successfully using environment variables")
	return nil
}

func GetMessagingClient() *messaging.Client {
	return messagingClient
}

func IsFirebaseEnabled() bool {
	return messagingClient != nil
}
