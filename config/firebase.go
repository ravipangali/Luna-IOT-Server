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

	// Use hardcoded credentials from your service account file
	// These are the exact credentials from your firebase-service-account.json
	credentials := map[string]interface{}{
		"type":                        "service_account",
		"project_id":                  "luna-iot-b5cdd",
		"private_key_id":              "db5d32a02cc822f776fb829c5ded8ab999d4d9d7",
		"private_key":                 "-----BEGIN PRIVATE KEY-----\nMIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCS6sKIxE3fzazE\nqH/i+DBGbVdAwBTLS2oIyGSUrERB93BdX1PYpykLyxg8tDGN37mUDsRc5aOhc/Ru\nT4gL2yYplw4GXkUEzZaNgmJdPjgsu/w9vmQEN+hUjtGTdgJ2AVilXujFJBxsw/FS\nTBEqLM5cDTbgAE7m5SRdgj/6+jkhOxKk0biPP0caQDSvxcv6LC/XL5oWdwrJ/pNg\nCpLPOY9Um6JPGIiQhdk5IYHGhgl+5IblaAzmmBcvJKXEjLX+GFgiw9rR1gn75fuS\ns4gotBO4XDF3W/UCl2bm0odsXEu0IkHCBFhFsu7LYXgHYNxBGYJ5gg18FwPLkEPr\nV6p3yyrBAgMBAAECggEAH8cel9WaIIgM0CbALrhPPNSGtG83sDdaMpchxlSymPAs\nAk5NxQV3J+FglzTEqTLUobVF/PAA4jnCC6AxRZs72HAfbPo0BJNxdp9WpmOAZBCv\nQS2u1YjAPJX9t98lLiAha/eo8odajJ4fUxU3+z7gzeFf1rjKWEAFCyLSsvcvp0Ob\nX0RLu6awJIz5is+PSI60btKC5+/jxxdIbldg6YSOmiPyW9rXVUfhk0Cjlg4jXIhm\nqs1CYtlD9Bsa6mcujJpSXqjMWysE4K48OXgcJAW6GIn6F20GyeoksYksZfgFvcbn\ncguhg9G72fMpAL1Nvo1Xjlu6fLpV//i/fCYw1BWJlwKBgQDE47Hs9lt2Fbq7p1Lp\nGNtXTxGboagivScWeIT5uKgu/pzszu/C2i0T986/RUXmm4DpNaAGqBwfWHEeqLCg\ne+xp6nDIDQI1PqTxuiRxQLRbpyafssGjln+yGh4L9HN5ZJLr2oqF2tgbRBqMusCq\nwmbycLKulEvIgOjhiX8KeHGTwwKBgQC/BleyrSLGZ0t0nMJYeKmGvZLjS8Wn/CwB\nUnj/Phv9D2sQoa7L1QszBHEAx2p+9ewa0OmaMXOjJttGpVpGQSa9LiUhTqZEOZTa\nf+0/sysx2HKNq5aoPAdn32homAU3TVqiadhDGXdHulSeNLNdFW2Z5CT713+fQzMs\nXl3ZW6KzKwKBgEVDUKFq3SwCYumG6Gzl+KuTPj+AtBRcdHa8ORNceZXmri/EcKYc\neIUwxQOWjAufIs9ntP8CfrosM8c0UsZyMe3ksn49zUwL2JzM/er1dz1S5QyDJwm0\ndQGjnHRaL5FB50mfXOHP5fxZjfl57TNlJjAdo041Dx/e8Y39/7ogOtxfAoGAIIga\n/VHg/zruLcDYlCqQbGLylgT8d1xJvjvmYUmZiKJMkHuIgiwZCSozeHd9mnuVJwf3\nEIxlbh6a71APrLFBwKwQJLj5Nds8j22D4PpJW+bJs3jKYoI+nKD+bfmdwcpJqiku\nbFb06mFAMeU1up+Al9mztrP/hwbxuxejEfY6IhsCgYAVstcBmPpWd6HHGgloDFoh\nYUTSWQB5pUWFwzQjPZMWTMBlI2snmMpECXpR1vWG71HPSD9lZo3lSJApi/Ht2pAb\n2lZ3seVPyU4n/D/MEht25PnQyJPk0NX8b9gvhgQ84DlRFAydcZVWiZqlHhDQtQqx\nNCTpfWr2m6ItzG4UfqnXpg==\n-----END PRIVATE KEY-----\n",
		"client_email":                "firebase-adminsdk-fbsvc@luna-iot-b5cdd.iam.gserviceaccount.com",
		"client_id":                   "116176236230697495561",
		"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
		"token_uri":                   "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url":        "https://www.googleapis.com/robot/v1/metadata/x509/firebase-adminsdk-fbsvc%40luna-iot-b5cdd.iam.gserviceaccount.com",
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
		ProjectID: "luna-iot-b5cdd",
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
