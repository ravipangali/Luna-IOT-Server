# Firebase 404 Error - Quick Fix

## The Problem
You're getting a 404 error on `/batch` when sending notifications. This means your Firebase project `luna-iot-b5cdd` either:
1. **Doesn't exist** - The project ID is wrong
2. **Service account has no permissions** - The service account can't access the project
3. **Invalid credentials** - The private key or other credentials are wrong

## Quick Fix Steps

### Step 1: Check if Project Exists
1. Go to [Firebase Console](https://console.firebase.google.com/)
2. Look for project `luna-iot-b5cdd`
3. If it doesn't exist, create it with this exact name

### Step 2: Fix Service Account
1. In your Firebase project, go to **Project Settings** (gear icon)
2. Click on **Service accounts** tab
3. Click **Generate new private key**
4. Choose **Firebase Admin SDK**
5. Download the JSON file

### Step 3: Replace Credentials
1. Replace your `firebase-service-account.json` file with the new one
2. Or update the hardcoded credentials in `config/firebase.go`

### Step 4: Test
1. Restart your server
2. Test sending a notification

## Alternative: Use Legacy Server Key

If the above doesn't work, try using the legacy server key approach:

1. In Firebase Console, go to **Project Settings** > **Cloud Messaging**
2. Copy the **Server key**
3. Update your code to use the legacy server key instead of service account

## Common Issues

1. **Project doesn't exist**: Create it with exact name `luna-iot-b5cdd`
2. **Wrong permissions**: Service account needs "Firebase Admin SDK Administrator" role
3. **Invalid credentials**: Generate new service account key
4. **Cloud Messaging not enabled**: Enable it in Firebase Console

## Test Command
Run this to test your Firebase connection:
```bash
go run cmd/test-notification/main.go
``` 