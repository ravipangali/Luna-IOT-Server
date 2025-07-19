package services

import (
	"fmt"
	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"
	"time"
)

// VehicleNotificationService handles vehicle-specific notifications
type VehicleNotificationService struct {
	ravipangaliService *RavipangaliService
}

// NewVehicleNotificationService creates a new vehicle notification service
func NewVehicleNotificationService() *VehicleNotificationService {
	return &VehicleNotificationService{
		ravipangaliService: NewRavipangaliService(),
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

	// Get the latest valid GPS data from database for comparison (this data is already saved)
	var lastGPSData models.GPSData
	err := db.GetDB().Where("imei = ? AND ignition IS NOT NULL AND ignition != ''", gpsData.IMEI).
		Order("timestamp DESC").
		First(&lastGPSData).Error

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

		// Check for overspeed
		if currentSpeed > vehicle.Overspeed {
			// Check if last speed was also overspeed
			if err != nil || (lastGPSData.Speed == nil || *lastGPSData.Speed <= vehicle.Overspeed) {
				colors.PrintWarning("🚨 Overspeed detected! Speed: %d km/h, Limit: %d km/h", currentSpeed, vehicle.Overspeed)
				return vns.sendSpeedNotification(notificationData, NotificationTypeOverspeed, currentSpeed, vehicle.Overspeed)
			} else {
				colors.PrintInfo("⏭️ Already overspeeding - skipping notification")
			}
		}

		// Check for running (speed > 5)
		if currentSpeed > 5 {
			// Check if last speed was also > 5
			if err != nil || (lastGPSData.Speed == nil || *lastGPSData.Speed <= 5) {
				colors.PrintInfo("🏃 Vehicle started moving! Speed: %d km/h", currentSpeed)
				return vns.sendSpeedNotification(notificationData, NotificationTypeRunning, currentSpeed, 5)
			} else {
				colors.PrintInfo("⏭️ Vehicle already moving - skipping notification")
			}
		}
	}

	colors.PrintInfo("✅ No notifications needed for IMEI: %s", gpsData.IMEI)
	return nil
}

// sendIgnitionNotification sends ignition-related notifications
func (vns *VehicleNotificationService) sendIgnitionNotification(data *VehicleNotificationData, notificationType NotificationType) error {
	var title, body string

	switch notificationType {
	case NotificationTypeIgnitionOn:
		title = fmt.Sprintf("%s: Ignition On", data.RegNo)
		body = fmt.Sprintf("Your vehicle is turned ON\nDate: %s\nTime: %s",
			data.Timestamp.Format("2006-01-02"),
			data.Timestamp.Format("03:04 PM"))
	case NotificationTypeIgnitionOff:
		title = fmt.Sprintf("%s: Ignition Off", data.RegNo)
		body = fmt.Sprintf("Your vehicle is turned OFF\nDate: %s\nTime: %s",
			data.Timestamp.Format("2006-01-02"),
			data.Timestamp.Format("03:04 PM"))
	default:
		return fmt.Errorf("unknown ignition notification type: %s", notificationType)
	}

	return vns.sendNotificationToVehicleUsers(data.IMEI, title, body, "alert")
}

// sendSpeedNotification sends speed-related notifications
func (vns *VehicleNotificationService) sendSpeedNotification(data *VehicleNotificationData, notificationType NotificationType, currentSpeed int, threshold int) error {
	var title, body string

	switch notificationType {
	case NotificationTypeOverspeed:
		title = fmt.Sprintf("%s: Vehicle is Overspeed", data.RegNo)
		body = fmt.Sprintf("Your vehicle is overspeeding (Speed: %d km/h)\nDate: %s\nTime: %s",
			currentSpeed,
			data.Timestamp.Format("2006-01-02"),
			data.Timestamp.Format("03:04 PM"))
	case NotificationTypeRunning:
		title = fmt.Sprintf("%s: Vehicle is Running", data.RegNo)
		body = fmt.Sprintf("Your vehicle is moving (Speed: %d km/h)\nDate: %s\nTime: %s",
			currentSpeed,
			data.Timestamp.Format("2006-01-02"),
			data.Timestamp.Format("03:04 PM"))
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
		if uv.ExpiresAt != nil && time.Now().After(*uv.ExpiresAt) {
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
			"timestamp":         time.Now().Unix(),
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
