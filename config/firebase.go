package config

import (
	"context"
	"encoding/json"
	"luna_iot_server/pkg/colors"
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
		ProjectID:     getEnv("FIREBASE_PROJECT_ID", "luna-iot-b5cdd"),
		PrivateKeyID:  getEnv("FIREBASE_PRIVATE_KEY_ID", "499080667fce39b655a13c75ffe715ff94185f8d"),
		PrivateKey:    getEnv("FIREBASE_PRIVATE_KEY", "-----BEGIN PRIVATE KEY-----\nMIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQDgoxhyhYJUyBRn\nbwRk1oD8BkBw9bd9lh+q2b1IbaYfhhaLsx6P5IPkfF3YgMnAgwI0ojnrMl1Gl903\n7p59T1eGQnbwl4pVn3CI+OsF5GgqSMP2XPLahQy6uz0pQRG2LF/3u+64GsDb2nMJ\nlKsX5v+/n3gJgJbg/mvP3O7V6yycyedEUzFRs6KS/PyD/rQ7s2UZAZSZm7uMB8OC\nHIMdKdQPL3cseJxLNgeOgNivGN0V62brMYcSvaRmrl3cpbPyh58DVG5rLFIUfFoa\nVZ9EOqRg6BUNO9R1RL1waN8V0PJuIEUkrz/C33VpE6G7LuzX9ZXaKekVILPZoRdl\nvYNp+i39AgMBAAECggEAEdCAiu1EkZwP/8eBlf+e2TdW5/GZYur2liWDY2ZZFmBO\nktCaW8RX1ohewHsu9/nlWsg2U9ns8Y3vyLsMAMyiGJWJYJLoyqaEyNzSQprdAEsN\nMCrDeNa0h9OufRvzn1Qm98eeUSWQZgvaZwgBU4Y86Cgc35M98LGsjVHF83TQvZqg\nk1TS51cwvBhavsHgQPn84Hqb55lmgXdCcgU2qps80g+f71joDYifwiyS6nDFnmj3\n6pm6zDhmkqZVJMeg0CyGMD85aSFwDmuNAsi53eZlwwks50eLec0kCaO9/ElxF/6x\neKSd8/tzGaKobpYCCkDYlgiu4eSyi9TThD2Fm0FpnwKBgQD3UjhLsW8hbwLyXr05\nr68EU9sk+nd3j1+WZlo6KR1Y2dV3XNAwvrJ5OsKV8/CQmxV/p9crB0gRBdjVPk25\nUvm0PVax/DqP6s99ZRxSwphA6OFgnXmqd4WKLZDFsIgZytV2OaH8R4Uqb9LnEI6h\n5J+NvuqXxZ6q3ePiUQJ48wycUwKBgQDohRiLtJYMrzmVUFbZuO6R0PjsiQuGHD3n\ns8pocmQd6XrS6laIC++vEqT5E83+ggcR20ls4MxmcvXOhwbwGNnoHW9xBi29ERxa\nM5WQNn7bXfP1+rGFw2qsp4KkpFOPecFGwIyRGt6G/kTq5Wqi9XCjVDV5s7JD1BgR\nc1mk3VdCbwKBgHYq+azo1TFDSkQlkgHK+DN4IX/UkFo2zbQdqUSaumPmiMDkPrDb\nnIih07E0AaAGCUqaFguACiXgBk802owOojJFEHQwEIcM6SB/u/2q7nYtDupLs4MI\nYmy4ArEB/LVeHYnEVaolPfIdxcYTOiMOClH+gzYK/RmktSpADI9fiYnzAoGAMaG/\nVIrOgJSigPmuIDk2S0/E4pB6Mj0zBZM+AD9ymWPuALlekRmjJsafCj+s98d/hNM/\nAAuX9cJSL6xo0bUsRjyKPiDogHP3jlV2dlr7hw2t9nJ1lCzbR1FWNJiS8Yw2skiF\neK+4ki4SPeWMdo5XZbWi2IB/67SJEqiBmQxaBOcCgYBdEQE7EXYIZINLvJ52le9a\nLn2AxiSJTVWF4+6elclceF9mzQ9Bb8ZzhsybJfCSdHgxP37vHZKSIJHE6Znb4uhG\nuopTxNfsDYUjoLCusT/MpjZHF1pr+g1bW2bF4tpPDDpj5gEwLpwjbahV4oyveGAP\n4YuXI1tGM82qPSZH/nCaKA==\n-----END PRIVATE KEY-----\n"),
		ClientEmail:   getEnv("FIREBASE_CLIENT_EMAIL", "firebase-adminsdk-fbsvc@luna-iot-b5cdd.iam.gserviceaccount.com"),
		ClientID:      getEnv("FIREBASE_CLIENT_ID", "116176236230697495561"),
		AuthURI:       getEnv("FIREBASE_AUTH_URI", "https://accounts.google.com/o/oauth2/auth"),
		TokenURI:      getEnv("FIREBASE_TOKEN_URI", "https://oauth2.googleapis.com/token"),
		AuthProvider:  getEnv("FIREBASE_AUTH_PROVIDER_X509_CERT_URL", "https://www.googleapis.com/oauth2/v1/certs"),
		ClientCertURL: getEnv("FIREBASE_CLIENT_X509_CERT_URL", "https://www.googleapis.com/robot/v1/metadata/x509/firebase-adminsdk-fbsvc%40luna-iot-b5cdd.iam.gserviceaccount.com"),
	}
}

func InitializeFirebase() error {
	colors.PrintInfo("Starting Firebase initialization...")

	// Try to read from service account file first
	serviceAccountPath := "firebase-service-account.json"

	// Check if service account file exists
	if _, err := os.Stat(serviceAccountPath); err == nil {
		colors.PrintInfo("Found Firebase service account file, using it for initialization")
		// Use service account file
		opt := option.WithCredentialsFile(serviceAccountPath)
		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			colors.PrintError("Failed to initialize Firebase with service account file: %v", err)
			colors.PrintWarning("Firebase initialization failed, push notifications will be disabled")
			return nil // Don't return error, just disable Firebase
		}

		firebaseApp = app

		// Initialize messaging client
		messaging, err := app.Messaging(context.Background())
		if err != nil {
			colors.PrintError("Failed to initialize Firebase messaging client: %v", err)
			colors.PrintWarning("Firebase messaging client failed, push notifications will be disabled")
			return nil // Don't return error, just disable Firebase
		}

		messagingClient = messaging
		colors.PrintSuccess("Firebase initialized successfully using service account file")
		colors.PrintInfo("Messaging client: %v", messagingClient)
		return nil
	}

	colors.PrintInfo("No Firebase service account file found, trying environment variables")

	// Fallback to environment variables
	config := GetFirebaseConfig()

	colors.PrintInfo("Firebase config check:")
	colors.PrintInfo("  ProjectID: %s", config.ProjectID)
	colors.PrintInfo("  ClientEmail: %s", config.ClientEmail)
	colors.PrintInfo("  PrivateKeyID: %s", config.PrivateKeyID)
	colors.PrintInfo("  PrivateKey length: %d", len(config.PrivateKey))

	if config.ProjectID == "" {
		colors.PrintWarning("Firebase not configured, push notifications will be disabled")
		return nil
	}

	colors.PrintInfo("Firebase config found: ProjectID=%s, ClientEmail=%s",
		config.ProjectID, config.ClientEmail)

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
		colors.PrintError("Failed to marshal Firebase credentials: %v", err)
		colors.PrintWarning("Firebase initialization failed, push notifications will be disabled")
		return nil // Don't return error, just disable Firebase
	}

	colors.PrintInfo("Firebase credentials JSON created successfully")

	// Initialize Firebase app
	opt := option.WithCredentialsJSON(credentialsJSON)
	app, err := firebase.NewApp(context.Background(), &firebase.Config{
		ProjectID: config.ProjectID,
	}, opt)

	if err != nil {
		colors.PrintError("Failed to create Firebase app: %v", err)
		colors.PrintWarning("Firebase app creation failed, push notifications will be disabled")
		return nil // Don't return error, just disable Firebase
	}

	firebaseApp = app
	colors.PrintInfo("Firebase app created successfully")

	// Initialize messaging client
	messaging, err := app.Messaging(context.Background())
	if err != nil {
		colors.PrintError("Failed to initialize Firebase messaging client: %v", err)
		colors.PrintWarning("Firebase messaging client failed, push notifications will be disabled")
		return nil // Don't return error, just disable Firebase
	}

	messagingClient = messaging
	colors.PrintSuccess("Firebase initialized successfully using environment variables")
	colors.PrintInfo("Messaging client: %v", messagingClient)
	return nil
}

func GetMessagingClient() *messaging.Client {
	return messagingClient
}

func IsFirebaseEnabled() bool {
	enabled := messagingClient != nil
	colors.PrintInfo("Firebase enabled check: %t", enabled)
	return enabled
}
