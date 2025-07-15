package services

import (
	"testing"

	"luna_iot_server/internal/db"
)

// TestCreateNotification tests the notification creation functionality
func TestCreateNotification(t *testing.T) {
	// Initialize database connection for testing
	if err := db.Initialize(); err != nil {
		t.Skipf("Database not available for testing: %v", err)
	}
	defer db.Close()

	service := NewNotificationDBService()

	// Test data
	req := &CreateNotificationRequest{
		Title:     "Test Notification",
		Body:      "This is a test notification",
		Type:      "test",
		ImageURL:  "https://example.com/image.jpg",
		Sound:     "default",
		Priority:  "high",
		Data:      map[string]interface{}{"key": "value"},
		UserIDs:   []uint{1, 2, 3},
		CreatedBy: 1,
	}

	// Create notification
	response, err := service.CreateNotification(req)
	if err != nil {
		t.Errorf("Failed to create notification: %v", err)
		return
	}

	if !response.Success {
		t.Errorf("Notification creation failed: %s", response.Message)
		return
	}

	if response.Data == nil {
		t.Error("Notification data is nil")
		return
	}

	// Verify notification was created correctly
	if response.Data.Title != req.Title {
		t.Errorf("Expected title %s, got %s", req.Title, response.Data.Title)
	}

	if response.Data.Body != req.Body {
		t.Errorf("Expected body %s, got %s", req.Body, response.Data.Body)
	}

	if response.Data.IsSent {
		t.Error("Notification should not be marked as sent initially")
	}

	// Clean up - delete the test notification
	if err := service.DeleteNotification(response.Data.ID); err != nil {
		t.Logf("Warning: Failed to clean up test notification: %v", err)
	}
}

// TestUpdateNotification tests the notification update functionality
func TestUpdateNotification(t *testing.T) {
	// Initialize database connection for testing
	if err := db.Initialize(); err != nil {
		t.Skipf("Database not available for testing: %v", err)
	}
	defer db.Close()

	service := NewNotificationDBService()

	// First create a notification
	createReq := &CreateNotificationRequest{
		Title:     "Original Title",
		Body:      "Original Body",
		Type:      "test",
		UserIDs:   []uint{1},
		CreatedBy: 1,
	}

	createResponse, err := service.CreateNotification(createReq)
	if err != nil {
		t.Errorf("Failed to create notification for update test: %v", err)
		return
	}

	// Update the notification
	updateReq := &UpdateNotificationRequest{
		Title:    "Updated Title",
		Body:     "Updated Body",
		Type:     "updated_test",
		UserIDs:  []uint{1, 2},
		Priority: "normal",
	}

	updateResponse, err := service.UpdateNotification(createResponse.Data.ID, updateReq)
	if err != nil {
		t.Errorf("Failed to update notification: %v", err)
		return
	}

	if !updateResponse.Success {
		t.Errorf("Notification update failed: %s", updateResponse.Message)
		return
	}

	// Verify notification was updated correctly
	if updateResponse.Data.Title != updateReq.Title {
		t.Errorf("Expected updated title %s, got %s", updateReq.Title, updateResponse.Data.Title)
	}

	if updateResponse.Data.Body != updateReq.Body {
		t.Errorf("Expected updated body %s, got %s", updateReq.Body, updateResponse.Data.Body)
	}

	if updateResponse.Data.IsSent {
		t.Error("Updated notification should not be marked as sent")
	}

	// Clean up - delete the test notification
	if err := service.DeleteNotification(updateResponse.Data.ID); err != nil {
		t.Logf("Warning: Failed to clean up test notification: %v", err)
	}
}

// TestMarkNotificationAsSent tests marking a notification as sent
func TestMarkNotificationAsSent(t *testing.T) {
	// Initialize database connection for testing
	if err := db.Initialize(); err != nil {
		t.Skipf("Database not available for testing: %v", err)
	}
	defer db.Close()

	service := NewNotificationDBService()

	// Create a notification
	req := &CreateNotificationRequest{
		Title:     "Test Notification",
		Body:      "This is a test notification",
		Type:      "test",
		UserIDs:   []uint{1},
		CreatedBy: 1,
	}

	response, err := service.CreateNotification(req)
	if err != nil {
		t.Errorf("Failed to create notification: %v", err)
		return
	}

	// Mark as sent
	err = service.MarkNotificationAsSent(response.Data.ID)
	if err != nil {
		t.Errorf("Failed to mark notification as sent: %v", err)
		return
	}

	// Verify it was marked as sent
	notification, err := service.GetNotificationByID(response.Data.ID)
	if err != nil {
		t.Errorf("Failed to get notification: %v", err)
		return
	}

	if !notification.IsSent {
		t.Error("Notification should be marked as sent")
	}

	if notification.SentAt == nil {
		t.Error("SentAt should be set")
	}

	// Clean up
	if err := service.DeleteNotification(notification.ID); err != nil {
		t.Logf("Warning: Failed to clean up test notification: %v", err)
	}
}
