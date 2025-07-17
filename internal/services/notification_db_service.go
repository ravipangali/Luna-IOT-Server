package services

import (
	"encoding/json"
	"errors"
	"time"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"

	"gorm.io/gorm"
)

// NotificationDBService handles database operations for notifications
type NotificationDBService struct{}

// NewNotificationDBService creates a new notification database service
func NewNotificationDBService() *NotificationDBService {
	return &NotificationDBService{}
}

// CreateNotificationRequest represents the request for creating a notification
type CreateNotificationRequest struct {
	Title     string                 `json:"title" binding:"required"`
	Body      string                 `json:"body" binding:"required"`
	Type      string                 `json:"type"`
	ImageURL  string                 `json:"image_url"`
	ImageData string                 `json:"image_data"` // File path for uploaded images
	Sound     string                 `json:"sound"`
	Priority  string                 `json:"priority"`
	Data      map[string]interface{} `json:"data"`
	UserIDs   []uint                 `json:"user_ids" binding:"required"`
	CreatedBy uint                   `json:"created_by"`
}

// UpdateNotificationRequest represents the request for updating a notification
type UpdateNotificationRequest struct {
	Title     string                 `json:"title" binding:"required"`
	Body      string                 `json:"body" binding:"required"`
	Type      string                 `json:"type"`
	ImageURL  string                 `json:"image_url"`
	ImageData string                 `json:"image_data"` // File path for uploaded images
	Sound     string                 `json:"sound"`
	Priority  string                 `json:"priority"`
	Data      map[string]interface{} `json:"data"`
	UserIDs   []uint                 `json:"user_ids" binding:"required"`
}

// NotificationResponse represents the response from notification operations
type NotificationResponse struct {
	Success bool                 `json:"success"`
	Message string               `json:"message"`
	Data    *models.Notification `json:"data,omitempty"`
	Error   string               `json:"error,omitempty"`
}

// CreateNotification creates a new notification in the database
func (nds *NotificationDBService) CreateNotification(req *CreateNotificationRequest) (*NotificationResponse, error) {
	database := db.GetDB()

	// Validate required fields
	if req.Title == "" {
		return &NotificationResponse{
			Success: false,
			Message: "Title is required",
			Error:   "title_required",
		}, errors.New("title is required")
	}

	if req.Body == "" {
		return &NotificationResponse{
			Success: false,
			Message: "Body is required",
			Error:   "body_required",
		}, errors.New("body is required")
	}

	if len(req.UserIDs) == 0 {
		return &NotificationResponse{
			Success: false,
			Message: "At least one user must be selected",
			Error:   "users_required",
		}, errors.New("at least one user must be selected")
	}

	// Convert data map to JSON string
	dataJSON := ""
	if req.Data != nil {
		if dataBytes, err := json.Marshal(req.Data); err == nil {
			dataJSON = string(dataBytes)
		} else {
			colors.PrintWarning("Failed to marshal notification data: %v", err)
		}
	}

	// Set default values
	if req.Type == "" {
		req.Type = "system_notification"
	}
	if req.Priority == "" {
		req.Priority = "normal"
	}

	// Create the notification
	notification := models.Notification{
		Title:     req.Title,
		Body:      req.Body,
		Type:      req.Type,
		ImageURL:  req.ImageURL,
		ImageData: req.ImageData, // Add the image_data field
		Sound:     req.Sound,
		Priority:  req.Priority,
		Data:      dataJSON,
		CreatedBy: req.CreatedBy,
		IsSent:    false, // Always start as not sent
	}

	// Debug logging
	colors.PrintInfo("Saving notification to database with ImageURL: %s, ImageData: %s", notification.ImageURL, notification.ImageData)

	// Start transaction
	tx := database.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Save notification
	if err := tx.Create(&notification).Error; err != nil {
		tx.Rollback()
		colors.PrintError("Failed to create notification: %v", err)
		return &NotificationResponse{
			Success: false,
			Message: "Failed to create notification",
			Error:   "database_error",
		}, err
	}

	// Associate users with notification
	var notificationUsers []models.NotificationUser
	for _, userID := range req.UserIDs {
		notificationUsers = append(notificationUsers, models.NotificationUser{
			NotificationID: notification.ID,
			UserID:         userID,
			IsSent:         false, // Always start as not sent
		})
	}

	if err := tx.Create(&notificationUsers).Error; err != nil {
		tx.Rollback()
		colors.PrintError("Failed to associate users with notification: %v", err)
		return &NotificationResponse{
			Success: false,
			Message: "Failed to associate users with notification",
			Error:   "database_error",
		}, err
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		colors.PrintError("Failed to commit notification creation: %v", err)
		return &NotificationResponse{
			Success: false,
			Message: "Failed to save notification",
			Error:   "database_error",
		}, err
	}

	colors.PrintSuccess("Notification created successfully with ID: %d", notification.ID)

	return &NotificationResponse{
		Success: true,
		Message: "Notification created successfully",
		Data:    &notification,
	}, nil
}

// UpdateNotification updates an existing notification in the database
func (nds *NotificationDBService) UpdateNotification(notificationID uint, req *UpdateNotificationRequest) (*NotificationResponse, error) {
	database := db.GetDB()

	// Validate required fields
	if req.Title == "" {
		return &NotificationResponse{
			Success: false,
			Message: "Title is required",
			Error:   "title_required",
		}, errors.New("title is required")
	}

	if req.Body == "" {
		return &NotificationResponse{
			Success: false,
			Message: "Body is required",
			Error:   "body_required",
		}, errors.New("body is required")
	}

	if len(req.UserIDs) == 0 {
		return &NotificationResponse{
			Success: false,
			Message: "At least one user must be selected",
			Error:   "users_required",
		}, errors.New("at least one user must be selected")
	}

	// Check if notification exists
	var notification models.Notification
	if err := database.First(&notification, notificationID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &NotificationResponse{
				Success: false,
				Message: "Notification not found",
				Error:   "not_found",
			}, err
		}
		colors.PrintError("Failed to fetch notification: %v", err)
		return &NotificationResponse{
			Success: false,
			Message: "Failed to fetch notification",
			Error:   "database_error",
		}, err
	}

	// Convert data map to JSON string
	dataJSON := ""
	if req.Data != nil {
		if dataBytes, err := json.Marshal(req.Data); err == nil {
			dataJSON = string(dataBytes)
		} else {
			colors.PrintWarning("Failed to marshal notification data: %v", err)
		}
	}

	// Set default values
	if req.Type == "" {
		req.Type = "system_notification"
	}
	if req.Priority == "" {
		req.Priority = "normal"
	}

	// Start transaction
	tx := database.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update notification fields
	notification.Title = req.Title
	notification.Body = req.Body
	notification.Type = req.Type
	notification.ImageURL = req.ImageURL
	notification.ImageData = req.ImageData // Add the image_data field
	notification.Sound = req.Sound
	notification.Priority = req.Priority
	notification.Data = dataJSON
	// Reset sent status when updating
	notification.IsSent = false
	notification.SentAt = nil

	if err := tx.Save(&notification).Error; err != nil {
		tx.Rollback()
		colors.PrintError("Failed to update notification: %v", err)
		return &NotificationResponse{
			Success: false,
			Message: "Failed to update notification",
			Error:   "database_error",
		}, err
	}

	// Delete existing user associations
	if err := tx.Where("notification_id = ?", notification.ID).Delete(&models.NotificationUser{}).Error; err != nil {
		tx.Rollback()
		colors.PrintError("Failed to delete existing user associations: %v", err)
		return &NotificationResponse{
			Success: false,
			Message: "Failed to update user associations",
			Error:   "database_error",
		}, err
	}

	// Create new user associations
	var notificationUsers []models.NotificationUser
	for _, userID := range req.UserIDs {
		notificationUsers = append(notificationUsers, models.NotificationUser{
			NotificationID: notification.ID,
			UserID:         userID,
			IsSent:         false, // Reset sent status
			SentAt:         nil,
		})
	}

	if err := tx.Create(&notificationUsers).Error; err != nil {
		tx.Rollback()
		colors.PrintError("Failed to create new user associations: %v", err)
		return &NotificationResponse{
			Success: false,
			Message: "Failed to associate users with notification",
			Error:   "database_error",
		}, err
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		colors.PrintError("Failed to commit notification update: %v", err)
		return &NotificationResponse{
			Success: false,
			Message: "Failed to update notification",
			Error:   "database_error",
		}, err
	}

	colors.PrintSuccess("Notification updated successfully with ID: %d", notification.ID)

	return &NotificationResponse{
		Success: true,
		Message: "Notification updated successfully",
		Data:    &notification,
	}, nil
}

// MarkNotificationAsSent marks a notification as sent in the database
func (nds *NotificationDBService) MarkNotificationAsSent(notificationID uint) error {
	database := db.GetDB()

	// Start transaction
	tx := database.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update notification as sent
	now := time.Now()
	if err := tx.Model(&models.Notification{}).
		Where("id = ?", notificationID).
		Updates(map[string]interface{}{
			"is_sent": true,
			"sent_at": &now,
		}).Error; err != nil {
		tx.Rollback()
		colors.PrintError("Failed to mark notification as sent: %v", err)
		return err
	}

	// Update notification users as sent
	if err := tx.Model(&models.NotificationUser{}).
		Where("notification_id = ?", notificationID).
		Updates(map[string]interface{}{
			"is_sent": true,
			"sent_at": &now,
		}).Error; err != nil {
		tx.Rollback()
		colors.PrintError("Failed to mark notification users as sent: %v", err)
		return err
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		colors.PrintError("Failed to commit notification sent status: %v", err)
		return err
	}

	colors.PrintSuccess("Notification marked as sent: %d", notificationID)
	return nil
}

// GetNotificationByID retrieves a notification by ID
func (nds *NotificationDBService) GetNotificationByID(notificationID uint) (*models.Notification, error) {
	database := db.GetDB()

	colors.PrintInfo("Attempting to fetch notification with ID: %d", notificationID)

	var notification models.Notification
	if err := database.
		Preload("Creator").
		Preload("Users").
		First(&notification, notificationID).Error; err != nil {
		colors.PrintError("Database error fetching notification %d: %v", notificationID, err)
		return nil, err
	}

	colors.PrintInfo("Successfully fetched notification %d: Title='%s', Users count=%d",
		notificationID, notification.Title, len(notification.Users))

	return &notification, nil
}

// GetNotifications retrieves notifications with pagination
func (nds *NotificationDBService) GetNotifications(page, limit int) ([]models.Notification, int64, error) {
	database := db.GetDB()

	var notifications []models.Notification
	var total int64

	offset := (page - 1) * limit

	// Count total
	if err := database.Model(&models.Notification{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get notifications with creator and users
	if err := database.
		Preload("Creator").
		Preload("Users").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&notifications).Error; err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

// DeleteNotification deletes a notification from the database
func (nds *NotificationDBService) DeleteNotification(notificationID uint) error {
	database := db.GetDB()

	// Start transaction
	tx := database.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Delete notification users first (due to foreign key constraint)
	if err := tx.Unscoped().Where("notification_id = ?", notificationID).Delete(&models.NotificationUser{}).Error; err != nil {
		tx.Rollback()
		colors.PrintError("Failed to delete notification associations: %v", err)
		return err
	}

	// Delete notification permanently
	if err := tx.Unscoped().Delete(&models.Notification{}, notificationID).Error; err != nil {
		tx.Rollback()
		colors.PrintError("Failed to delete notification: %v", err)
		return err
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		colors.PrintError("Failed to commit notification deletion: %v", err)
		return err
	}

	colors.PrintSuccess("Notification deleted successfully: %d", notificationID)
	return nil
}
