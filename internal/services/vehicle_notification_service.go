package services

import (
	"fmt"
	"luna_iot_server/config"
	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"
	"time"
)

// VehicleNotificationService handles vehicle-specific notifications
type VehicleNotificationService struct {
	ravipangaliService *RavipangaliService
	// Track vehicle states to prevent duplicate notifications
	vehicleStates map[string]*VehicleState
}

// VehicleState tracks the current state of a vehicle
type VehicleState struct {
	IsMoving       bool
	IsOverspeeding bool
	LastSpeed      int
	LastUpdate     time.Time
}

// NewVehicleNotificationService creates a new vehicle notification service
func NewVehicleNotificationService() *VehicleNotificationService {
	return &VehicleNotificationService{
		ravipangaliService: NewRavipangaliService(),
		vehicleStates:      make(map[string]*VehicleState),
	}
}

// NotificationType represents different types of vehicle notifications
type NotificationType string

const (
	NotificationTypeIgnitionOn  NotificationType = "ignition_on"
	NotificationTypeIgnitionOff NotificationType = "ignition_off"
	NotificationTypeOverspeed   NotificationType = "overspeed"
	NotificationTypeRunning     NotificationType = "running"
)

// VehicleNotificationData represents the data needed for vehicle notifications
type VehicleNotificationData struct {
	IMEI        string
	RegNo       string
	VehicleName string
	Speed       *int
	Ignition    string
	Timestamp   time.Time
}

// CheckAndSendVehicleNotifications checks for vehicle state changes and sends notifications
func (vns *VehicleNotificationService) CheckAndSendVehicleNotifications(gpsData *models.GPSData) error {
	colors.PrintInfo("🔔 Checking vehicle notifications for IMEI: %s", gpsData.IMEI)

	// Get vehicle information
	var vehicle models.Vehicle
	if err := db.GetDB().Where("imei = ?", gpsData.IMEI).First(&vehicle).Error; err != nil {
		colors.PrintWarning("Vehicle not found for IMEI %s: %v", gpsData.IMEI, err)
		return nil // Not an error, just no vehicle registered
	}

	colors.PrintInfo("🚗 Vehicle found: %s (%s)", vehicle.Name, vehicle.RegNo)

	// Get or create vehicle state tracker
	vehicleState, exists := vns.vehicleStates[gpsData.IMEI]
	if !exists {
		vehicleState = &VehicleState{
			IsMoving:       false,
			IsOverspeeding: false,
			LastSpeed:      0,
			LastUpdate:     config.GetCurrentTime(),
		}
		vns.vehicleStates[gpsData.IMEI] = vehicleState
		colors.PrintInfo("🆕 Created new state tracker for vehicle %s", gpsData.IMEI)
	}

	// Prepare notification data
	notificationData := &VehicleNotificationData{
		IMEI:        gpsData.IMEI,
		RegNo:       vehicle.RegNo,
		VehicleName: vehicle.Name,
		Speed:       gpsData.Speed,
		Ignition:    gpsData.Ignition,
		Timestamp:   gpsData.Timestamp,
	}

	// Check ignition status changes
	if gpsData.Ignition != "" {
		colors.PrintInfo("🔑 Current ignition status: %s", gpsData.Ignition)

		// Get the PREVIOUS valid GPS data from database for ignition comparison
		var lastGPSData models.GPSData
		err := db.GetDB().Where("imei = ? AND ignition IS NOT NULL AND ignition != '' AND id != ?", gpsData.IMEI, gpsData.ID).
			Order("timestamp DESC").
			First(&lastGPSData).Error

		if err != nil {
			// No previous data, this is the first ignition status
			colors.PrintInfo("📝 No previous ignition data found")
			if gpsData.Ignition == "ON" {
				colors.PrintInfo("🚀 First ignition ON detected - sending notification")
				return vns.sendIgnitionNotification(notificationData, NotificationTypeIgnitionOn)
			}
		} else {
			// Compare with last known ignition status
			colors.PrintInfo("📊 Previous ignition status: %s", lastGPSData.Ignition)
			if lastGPSData.Ignition != gpsData.Ignition {
				colors.PrintInfo("🔄 Ignition status changed from %s to %s", lastGPSData.Ignition, gpsData.Ignition)
				if gpsData.Ignition == "ON" {
					return vns.sendIgnitionNotification(notificationData, NotificationTypeIgnitionOn)
				} else if gpsData.Ignition == "OFF" {
					return vns.sendIgnitionNotification(notificationData, NotificationTypeIgnitionOff)
				}
			} else {
				colors.PrintInfo("⏭️ Ignition status unchanged - skipping notification")
			}
		}
	}

	// Check speed-based notifications
	if gpsData.Speed != nil {
		currentSpeed := *gpsData.Speed
		colors.PrintInfo("🏃 Current speed: %d km/h, Overspeed limit: %d km/h", currentSpeed, vehicle.Overspeed)
		colors.PrintInfo("📊 Vehicle state - Moving: %v, Overspeeding: %v, Last Speed: %d",
			vehicleState.IsMoving, vehicleState.IsOverspeeding, vehicleState.LastSpeed)

		// Check for overspeed state change
		isCurrentlyOverspeeding := currentSpeed > vehicle.Overspeed
		if isCurrentlyOverspeeding && !vehicleState.IsOverspeeding {
			// Transition from normal speed to overspeed
			colors.PrintWarning("🚨 Overspeed detected! Speed: %d km/h, Limit: %d km/h", currentSpeed, vehicle.Overspeed)
			vehicleState.IsOverspeeding = true
			vehicleState.LastSpeed = currentSpeed
			vehicleState.LastUpdate = config.GetCurrentTime()
			return vns.sendSpeedNotification(notificationData, NotificationTypeOverspeed, currentSpeed, vehicle.Overspeed)
		} else if !isCurrentlyOverspeeding && vehicleState.IsOverspeeding {
			// Transition from overspeed to normal speed
			colors.PrintInfo("✅ Vehicle returned to normal speed: %d km/h", currentSpeed)
			vehicleState.IsOverspeeding = false
			vehicleState.LastSpeed = currentSpeed
			vehicleState.LastUpdate = config.GetCurrentTime()
		} else if isCurrentlyOverspeeding {
			colors.PrintInfo("⏭️ Already overspeeding - skipping notification")
		}

		// Check for moving state change
		isCurrentlyMoving := currentSpeed > 5
		if isCurrentlyMoving && !vehicleState.IsMoving {
			// Transition from stopped to moving
			colors.PrintInfo("🏃 Vehicle started moving! Speed: %d km/h (previous: %d)", currentSpeed, vehicleState.LastSpeed)
			vehicleState.IsMoving = true
			vehicleState.LastSpeed = currentSpeed
			vehicleState.LastUpdate = config.GetCurrentTime()
			return vns.sendSpeedNotification(notificationData, NotificationTypeRunning, currentSpeed, 5)
		} else if !isCurrentlyMoving && vehicleState.IsMoving {
			// Transition from moving to stopped
			colors.PrintInfo("🛑 Vehicle stopped moving. Speed: %d km/h", currentSpeed)
			vehicleState.IsMoving = false
			vehicleState.LastSpeed = currentSpeed
			vehicleState.LastUpdate = config.GetCurrentTime()
		} else if isCurrentlyMoving {
			colors.PrintInfo("⏭️ Vehicle already moving (speed: %d km/h) - skipping notification", currentSpeed)
			// Update last speed even if already moving
			vehicleState.LastSpeed = currentSpeed
			vehicleState.LastUpdate = config.GetCurrentTime()
		} else {
			// Vehicle is stopped
			vehicleState.LastSpeed = currentSpeed
			vehicleState.LastUpdate = config.GetCurrentTime()
		}
	}

	colors.PrintInfo("✅ No notifications needed for IMEI: %s", gpsData.IMEI)
	return nil
}

// sendIgnitionNotification sends ignition-related notifications
func (vns *VehicleNotificationService) sendIgnitionNotification(data *VehicleNotificationData, notificationType NotificationType) error {
	var title, body string

	// Use timezone-aware time formatting
	currentTime := config.GetCurrentTime()

	switch notificationType {
	case NotificationTypeIgnitionOn:
		title = fmt.Sprintf("%s: Ignition On", data.RegNo)
		body = fmt.Sprintf("Your vehicle is turned ON\nDate: %s\nTime: %s",
			currentTime.Format("2006-01-02"),
			currentTime.Format("03:04 PM"))
	case NotificationTypeIgnitionOff:
		title = fmt.Sprintf("%s: Ignition Off", data.RegNo)
		body = fmt.Sprintf("Your vehicle is turned OFF\nDate: %s\nTime: %s",
			currentTime.Format("2006-01-02"),
			currentTime.Format("03:04 PM"))
	default:
		return fmt.Errorf("unknown ignition notification type: %s", notificationType)
	}

	return vns.sendNotificationToVehicleUsers(data.IMEI, title, body, "alert")
}

// sendSpeedNotification sends speed-related notifications
func (vns *VehicleNotificationService) sendSpeedNotification(data *VehicleNotificationData, notificationType NotificationType, currentSpeed int, threshold int) error {
	var title, body string

	// Use timezone-aware time formatting
	currentTime := config.GetCurrentTime()

	switch notificationType {
	case NotificationTypeOverspeed:
		title = fmt.Sprintf("%s: Vehicle is Overspeed", data.RegNo)
		body = fmt.Sprintf("Your vehicle is overspeeding (Speed: %d km/h)\nDate: %s\nTime: %s",
			currentSpeed,
			currentTime.Format("2006-01-02"),
			currentTime.Format("03:04 PM"))
	case NotificationTypeRunning:
		title = fmt.Sprintf("%s: Vehicle is Running", data.RegNo)
		body = fmt.Sprintf("Your vehicle is moving (Speed: %d km/h)\nDate: %s\nTime: %s",
			currentSpeed,
			currentTime.Format("2006-01-02"),
			currentTime.Format("03:04 PM"))
	default:
		return fmt.Errorf("unknown speed notification type: %s", notificationType)
	}

	return vns.sendNotificationToVehicleUsers(data.IMEI, title, body, "alert")
}

// sendNotificationToVehicleUsers sends notification to all users who have notification permission for the vehicle
func (vns *VehicleNotificationService) sendNotificationToVehicleUsers(imei, title, body, notificationType string) error {
	colors.PrintInfo("📤 Sending notification to vehicle users for IMEI: %s", imei)
	colors.PrintInfo("📋 Title: %s", title)
	colors.PrintInfo("📝 Body: %s", body)

	// Get all users who have notification permission for this vehicle
	var userVehicles []models.UserVehicle
	err := db.GetDB().Preload("User").
		Where("vehicle_id = ? AND notification = ? AND is_active = ?", imei, true, true).
		Find(&userVehicles).Error

	if err != nil {
		colors.PrintError("Failed to get users for vehicle %s: %v", imei, err)
		return err
	}

	colors.PrintInfo("👥 Found %d users with notification permission for vehicle %s", len(userVehicles), imei)

	if len(userVehicles) == 0 {
		colors.PrintWarning("No users with notification permission found for vehicle %s", imei)
		return nil
	}

	// Collect FCM tokens from users
	var fcmTokens []string
	for _, uv := range userVehicles {
		// Check if access has expired
		if uv.ExpiresAt != nil && config.GetCurrentTime().After(*uv.ExpiresAt) {
			colors.PrintWarning("⏰ User %d access expired for vehicle %s", uv.UserID, imei)
			continue
		}

		if uv.User.FCMToken != "" {
			fcmTokens = append(fcmTokens, uv.User.FCMToken)
			colors.PrintInfo("📱 User %d (%s) has FCM token", uv.UserID, uv.User.Name)
		} else {
			colors.PrintWarning("📱 User %d (%s) has no FCM token", uv.UserID, uv.User.Name)
		}
	}

	if len(fcmTokens) == 0 {
		colors.PrintWarning("No FCM tokens found for vehicle %s users", imei)
		return nil
	}

	colors.PrintInfo("📲 Sending notification to %d FCM tokens", len(fcmTokens))

	// Send notification via Ravipangali API
	response, err := vns.ravipangaliService.SendPushNotification(
		title,
		body,
		fcmTokens,
		"", // No image
		map[string]interface{}{
			"vehicle_imei":      imei,
			"notification_type": notificationType,
			"timestamp":         config.GetCurrentTime().Unix(),
		},
		"high", // High priority for vehicle notifications
		notificationType,
		"default",
	)

	if err != nil {
		colors.PrintError("Failed to send vehicle notification: %v", err)
		return err
	}

	if response.Success {
		colors.PrintSuccess("✅ Vehicle notification sent successfully to %d users for vehicle %s", len(fcmTokens), imei)
		colors.PrintInfo("📊 Notification details: Sent=%d, Delivered=%d, Failed=%d",
			response.TokensSent, response.TokensDelivered, response.TokensFailed)
	} else {
		colors.PrintError("❌ Failed to send vehicle notification: %s", response.Error)
	}

	return nil
}

// CleanupOldVehicleStates removes vehicle states that haven't been updated for more than 24 hours
func (vns *VehicleNotificationService) CleanupOldVehicleStates() {
	colors.PrintInfo("🧹 Cleaning up old vehicle states...")

	cutoffTime := config.GetCurrentTime().Add(-24 * time.Hour)
	removedCount := 0

	for imei, state := range vns.vehicleStates {
		if state.LastUpdate.Before(cutoffTime) {
			delete(vns.vehicleStates, imei)
			removedCount++
			colors.PrintInfo("🗑️ Removed old state for vehicle %s (last update: %s)", imei, state.LastUpdate.Format("2006-01-02 15:04:05"))
		}
	}

	if removedCount > 0 {
		colors.PrintSuccess("✅ Cleaned up %d old vehicle states", removedCount)
	} else {
		colors.PrintInfo("✅ No old vehicle states to clean up")
	}
}

// GetVehicleStateInfo returns information about the current state of a vehicle
func (vns *VehicleNotificationService) GetVehicleStateInfo(imei string) *VehicleState {
	if state, exists := vns.vehicleStates[imei]; exists {
		return state
	}
	return nil
}

// ResetVehicleState resets the state for a specific vehicle (useful for testing)
func (vns *VehicleNotificationService) ResetVehicleState(imei string) {
	if _, exists := vns.vehicleStates[imei]; exists {
		delete(vns.vehicleStates, imei)
		colors.PrintInfo("🔄 Reset state for vehicle %s", imei)
	}
}
