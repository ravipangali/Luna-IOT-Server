# Firebase Setup Guide

## Current Issue
The Firebase service account key appears to be invalid or expired. The error "Invalid JWT Signature" indicates authentication problems.

## Solution Steps

### 1. Get a New Firebase Service Account Key

1. Go to [Firebase Console](https://console.firebase.google.com/)
2. Select your project: `luna-iot-5993f`
3. Go to **Project Settings** (gear icon)
4. Click on **Service accounts** tab
5. Click **Generate new private key**
6. Download the JSON file
7. Replace the existing `firebase_key.json` with the new one

### 2. Alternative: Use Environment Variables

If you prefer to use environment variables instead of a file:

```bash
# Set Firebase credentials as environment variables
export FIREBASE_PROJECT_ID="luna-iot-5993f"
export FIREBASE_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n"
export FIREBASE_CLIENT_EMAIL="firebase-adminsdk-xxx@luna-iot-5993f.iam.gserviceaccount.com"
```

### 3. Test the Setup

Run the test to verify Firebase is working:

```bash
cd luna_iot_server
go run cmd/test-firebase/main.go
```

### 4. Fallback Mode

The application now has a fallback mode that simulates notifications when Firebase is not available. This ensures your application continues to work even if Firebase is down.

## Expected Output

When Firebase is working correctly, you should see:
```
✅ Firebase service initialized successfully
✅ Firebase service is active
✅ Firebase connection test successful
✅ Notification sent successfully: projects/luna-iot-5993f/messages/...
```

When Firebase is not available, you should see:
```
⚠️ Firebase not available - simulating notification
ℹ Simulated notification: Title='...', Body='...', Token='...'
✅ Notification simulated successfully (Firebase not available)
```

## Troubleshooting

1. **Invalid JWT Signature**: Get a new service account key
2. **Permission denied**: Ensure the service account has the "Firebase Admin" role
3. **Project not found**: Verify the project ID is correct
4. **Network issues**: Check your internet connection

The application will continue to work in simulation mode until Firebase is properly configured. 