# Notification Saving Logic - Clean Architecture

## Overview

The notification system has been refactored to separate database operations from notification sending logic. This prevents errors from cascading and ensures that notifications are always saved to the database, even if sending fails.

## Architecture

### 1. Database Service (`NotificationDBService`)

Located in `internal/services/notification_db_service.go`, this service handles all database operations:

- **CreateNotification**: Saves notification to database with user associations
- **UpdateNotification**: Updates existing notification and user associations
- **MarkNotificationAsSent**: Marks notification as sent with timestamp
- **GetNotificationByID**: Retrieves notification with relationships
- **GetNotifications**: Retrieves paginated notifications
- **DeleteNotification**: Deletes notification and associations

### 2. Controller (`NotificationManagementController`)

Located in `internal/http/controllers/notification_management_controller.go`, this controller:

- Handles HTTP requests
- Uses database service for persistence
- Uses notification service for sending
- Provides clear error responses

### 3. Model (`Notification`)

Located in `internal/models/notification.go`, the model includes:

- Proper JSON data handling with `GetDataMap()` and `SetDataMap()`
- Default values in `BeforeCreate` hook
- Relationships with users and creator

## Key Features

### 1. Separation of Concerns

- **Database Operations**: Handled by `NotificationDBService`
- **Notification Sending**: Handled by `NotificationService`
- **HTTP Handling**: Handled by `NotificationManagementController`

### 2. Transaction Safety

All database operations use transactions to ensure data consistency:

```go
// Start transaction
tx := database.Begin()
defer func() {
    if r := recover(); r != nil {
        tx.Rollback()
    }
}()

// Database operations...

// Commit transaction
if err := tx.Commit().Error; err != nil {
    return err
}
```

### 3. Error Handling

- Database errors don't prevent notification creation
- Sending errors don't rollback database operations
- Clear error messages and logging

### 4. Data Validation

- Required fields validation
- User existence checks
- JSON data marshaling/unmarshaling

## Usage Examples

### Creating a Notification

```go
// Create notification request
req := &services.CreateNotificationRequest{
    Title:     "Test Notification",
    Body:      "This is a test notification",
    Type:      "system_notification",
    UserIDs:   []uint{1, 2, 3},
    CreatedBy: 1,
}

// Save to database
response, err := service.CreateNotification(req)
if err != nil {
    // Handle error
}

// Send notification (optional)
if sendImmediately {
    notificationData := &services.NotificationData{
        Title: response.Data.Title,
        Body:  response.Data.Body,
        // ... other fields
    }
    
    sendResponse, err := notificationService.SendToMultipleUsers(req.UserIDs, notificationData)
    if err == nil && sendResponse.Success {
        // Mark as sent
        service.MarkNotificationAsSent(response.Data.ID)
    }
}
```

### Updating a Notification

```go
// Update notification request
req := &services.UpdateNotificationRequest{
    Title:    "Updated Title",
    Body:     "Updated Body",
    UserIDs:  []uint{1, 2},
}

// Update in database
response, err := service.UpdateNotification(notificationID, req)
if err != nil {
    // Handle error
}
```

## Error Handling

### Database Errors

- Invalid data validation
- Foreign key constraint violations
- Transaction failures
- Connection issues

### Sending Errors

- Firebase configuration issues
- Network connectivity problems
- Invalid FCM tokens
- Rate limiting

## Benefits

1. **Reliability**: Notifications are always saved to database
2. **Maintainability**: Clear separation of concerns
3. **Testability**: Each service can be tested independently
4. **Scalability**: Easy to add new features or modify existing ones
5. **Error Isolation**: Database and sending errors don't affect each other

## Testing

The system includes comprehensive tests in `notification_db_service_test.go`:

- `TestCreateNotification`: Tests notification creation
- `TestUpdateNotification`: Tests notification updates
- `TestMarkNotificationAsSent`: Tests marking notifications as sent

## Migration Notes

The new system is backward compatible with existing notifications. The main changes are:

1. Database operations are now handled by a dedicated service
2. Sending logic is separated from database operations
3. Better error handling and logging
4. Improved data validation

## API Endpoints

All existing API endpoints remain the same:

- `POST /api/v1/notifications` - Create notification
- `PUT /api/v1/notifications/:id` - Update notification
- `DELETE /api/v1/notifications/:id` - Delete notification
- `POST /api/v1/notifications/:id/send` - Send notification
- `GET /api/v1/notifications` - List notifications
- `GET /api/v1/notifications/:id` - Get notification

The response format remains unchanged, ensuring compatibility with existing frontend code. 