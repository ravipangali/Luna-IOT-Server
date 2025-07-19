# Vehicle Notification System

This document describes the vehicle notification system implemented in the Luna IoT Server.

## Overview

The vehicle notification system automatically sends push notifications to users when specific vehicle events occur. Notifications are sent via the Ravipangali API to users who have notification permissions for the respective vehicles.

## Notification Types

### 1. Ignition Notifications

**Ignition ON**
- **Trigger**: Vehicle ignition turns ON
- **Title**: `{Vehicle_Reg_No}: Ignition On`
- **Body**: 
  ```
  Your vehicle is turned ON
  Date: yyyy-mm-dd
  Time: hh:mm am/pm
  ```

**Ignition OFF**
- **Trigger**: Vehicle ignition turns OFF
- **Title**: `{Vehicle_Reg_No}: Ignition Off`
- **Body**: 
  ```
  Your vehicle is turned OFF
  Date: yyyy-mm-dd
  Time: hh:mm am/pm
  ```

### 2. Speed Notifications

**Overspeed**
- **Trigger**: Vehicle speed exceeds the overspeed limit configured in the vehicle table
- **Title**: `{Vehicle_Reg_No}: Vehicle is Overspeed`
- **Body**: 
  ```
  Your vehicle is overspeeding (Speed: X km/h)
  Date: yyyy-mm-dd
  Time: hh:mm am/pm
  ```

**Running**
- **Trigger**: Vehicle speed exceeds 5 km/h
- **Title**: `{Vehicle_Reg_No}: Vehicle is Running`
- **Body**: 
  ```
  Your vehicle is moving (Speed: X km/h)
  Date: yyyy-mm-dd
  Time: hh:mm am/pm
  ```

## Smart Notification Logic

The system implements smart logic to avoid duplicate notifications:

### Ignition Notifications
- Only sends notification when ignition status **changes**
- If current ignition = "ON" and previous ignition = "ON" → No notification
- If current ignition = "OFF" and previous ignition = "OFF" → No notification
- If current ignition = "ON" and previous ignition = "OFF" → Send "Ignition ON" notification
- If current ignition = "OFF" and previous ignition = "ON" → Send "Ignition OFF" notification

### Speed Notifications
- **Overspeed**: Only sends when vehicle **starts** exceeding the limit
- **Running**: Only sends when vehicle **starts** moving (speed > 5 km/h)
- If vehicle is already overspeeding/moving → No duplicate notification

## User Permissions

Notifications are only sent to users who:
1. Have access to the vehicle (via `user_vehicles` table)
2. Have the `notification` permission set to `true`
3. Have active access (not expired)
4. Have a valid FCM token

## Configuration

### Environment Variables
The system uses the following environment variables for Ravipangali API:
- `RP_FIREBASE_APP_ID`: Firebase app ID
- `RP_ACCOUNT_EMAIL`: Ravipangali account email
- `RP_ACCOUNT_PASSWORD`: Ravipangali account password

### Vehicle Configuration
- **Overspeed Limit**: Set in the `vehicles.overspeed` column (default: 60 km/h)
- **Notification Permission**: Set in `user_vehicles.notification` column

## Implementation Details

### Files
- `internal/services/vehicle_notification_service.go`: Main notification service
- `internal/tcp/server.go`: TCP server integration

### Integration Points
1. **GPS Packet Processing**: Notifications triggered when GPS data is received
2. **Status Packet Processing**: Notifications triggered when status data is received
3. **Database Comparison**: System compares current data with previous database records

### Logging
The system provides detailed logging with emojis for easy identification:
- 🔔 Notification checking
- 🚗 Vehicle information
- 🔑 Ignition status
- 🏃 Speed information
- 🚨 Overspeed alerts
- 📤 Notification sending
- 👥 User discovery
- 📱 FCM token status

## Testing

To test the notification system:

1. Ensure a vehicle is registered in the database
2. Ensure users have notification permissions
3. Ensure users have FCM tokens
4. Send TCP data with ignition/speed changes
5. Check server logs for notification activity

## Troubleshooting

### Common Issues

1. **No notifications sent**
   - Check if vehicle exists in database
   - Check if users have notification permissions
   - Check if users have FCM tokens
   - Check Ravipangali API credentials

2. **Duplicate notifications**
   - System should prevent duplicates automatically
   - Check if database has stale data

3. **Missing FCM tokens**
   - Users need to register FCM tokens via mobile app
   - Check user registration process

### Debug Logs
Enable debug logging to see detailed notification flow:
```
🔔 Checking vehicle notifications for IMEI: 1234567890123456
🚗 Vehicle found: My Car (ABC-123)
🔑 Current ignition status: ON
📊 Previous ignition status: OFF
🔄 Ignition status changed from OFF to ON
📤 Sending notification to vehicle users for IMEI: 1234567890123456
👥 Found 2 users with notification permission for vehicle 1234567890123456
📱 User 1 (John Doe) has FCM token
📲 Sending notification to 1 FCM tokens
✅ Vehicle notification sent successfully to 1 users for vehicle 1234567890123456
``` 