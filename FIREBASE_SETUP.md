# Firebase Setup Guide

## Problem
You're getting a 404 error when trying to send notifications. This indicates that the Firebase configuration is invalid or the project doesn't exist.

## Solution

### Step 1: Create a Firebase Project

1. Go to [Firebase Console](https://console.firebase.google.com/)
2. Click "Create a project" or select an existing project
3. Follow the setup wizard
4. Note down your **Project ID** (e.g., `luna-iot-b5cdd`)

### Step 2: Enable Cloud Messaging

1. In your Firebase project, go to **Project Settings** (gear icon)
2. Click on the **Cloud Messaging** tab
3. Make sure **Cloud Messaging API** is enabled
4. Note down your **Server key** (optional, for legacy FCM)

### Step 3: Create a Service Account

1. In **Project Settings**, go to the **Service accounts** tab
2. Click **Generate new private key**
3. Choose **Firebase Admin SDK**
4. Click **Generate key**
5. Download the JSON file

### Step 4: Update Your Configuration

1. Rename the downloaded JSON file to `firebase-service-account.json`
2. Place it in the root directory of your server (`luna_iot_server/`)
3. Make sure the file is in your `.gitignore` to keep it secure

### Step 5: Test the Configuration

Run the test script to verify your setup:

```bash
cd luna_iot_server
go run test_firebase.go
```

### Step 6: Test via API

Once the server is running, test the Firebase connection:

```bash
curl -X GET "http://localhost:8080/api/v1/test/notifications/firebase-test" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Alternative: Environment Variables

If you prefer to use environment variables instead of a service account file:

1. Add these to your `.env` file:
```env
FIREBASE_PROJECT_ID=your-project-id
FIREBASE_PRIVATE_KEY_ID=your-private-key-id
FIREBASE_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n"
FIREBASE_CLIENT_EMAIL=your-service-account@your-project.iam.gserviceaccount.com
FIREBASE_CLIENT_ID=your-client-id
FIREBASE_AUTH_URI=https://accounts.google.com/o/oauth2/auth
FIREBASE_TOKEN_URI=https://oauth2.googleapis.com/token
FIREBASE_AUTH_PROVIDER_X509_CERT_URL=https://www.googleapis.com/oauth2/v1/certs
FIREBASE_CLIENT_X509_CERT_URL=https://www.googleapis.com/robot/v1/metadata/x509/your-service-account%40your-project.iam.gserviceaccount.com
```

2. Get these values from the service account JSON file you downloaded

## Troubleshooting

### 404 Error
- **Cause**: Invalid project ID or service account credentials
- **Solution**: Verify your Firebase project exists and the service account has the correct permissions

### Permission Denied
- **Cause**: Service account doesn't have Cloud Messaging permissions
- **Solution**: Make sure the service account has the "Firebase Admin SDK Administrator" role

### Network Issues
- **Cause**: Firewall or network restrictions
- **Solution**: Ensure your server can access `fcm.googleapis.com`

### Invalid Token
- **Cause**: FCM tokens are invalid or expired
- **Solution**: This is normal for test tokens. Real tokens from mobile apps should work.

## Security Notes

1. **Never commit** your `firebase-service-account.json` file to version control
2. **Rotate keys** regularly for production environments
3. **Use environment variables** in production for better security
4. **Restrict service account permissions** to only what's needed

## API Endpoints

Once configured, you can use these endpoints:

- `POST /api/v1/admin/notifications/send` - Send to specific users
- `POST /api/v1/admin/notifications/send-to-topic` - Send to topic
- `GET /api/v1/test/notifications/firebase-test` - Test Firebase connection

## Example Request

```bash
curl -X POST "http://localhost:8080/api/v1/admin/notifications/send" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "user_ids": [1, 2, 3],
    "title": "Test Notification",
    "body": "This is a test notification",
    "type": "system_notification"
  }'
``` 