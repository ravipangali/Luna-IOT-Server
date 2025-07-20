package tcp

import (
	"encoding/json"
	"fmt"
	"luna_iot_server/config"
	"luna_iot_server/internal/db"
	"luna_iot_server/internal/http"
	"luna_iot_server/internal/http/controllers"
	"luna_iot_server/internal/models"
	"luna_iot_server/internal/protocol"
	"luna_iot_server/internal/services"
	"luna_iot_server/pkg/colors"
	"math"
	"net"
	"sync"
	"time"
)

// DeviceConnection tracks device connection state and last activity
type DeviceConnection struct {
	Conn         net.Conn
	LastActivity time.Time
	IMEI         string
	IsActive     bool
}

// Server represents the TCP server for IoT devices
type Server struct {
	port              string
	listener          net.Listener
	controlController *controllers.ControlController
	// Track device connections with timestamps
	deviceConnections map[string]*DeviceConnection
	connectionMutex   sync.RWMutex
	timeoutTicker     *time.Ticker
	// Vehicle notification service
	vehicleNotificationService *services.VehicleNotificationService
	// GPS processing configuration
	enableGPSSmoothing  bool
	enableGPSValidation bool
}

// NewServer creates a new TCP server instance
func NewServer(port string) *Server {
	return &Server{
		port:                       port,
		controlController:          controllers.NewControlController(),
		deviceConnections:          make(map[string]*DeviceConnection),
		timeoutTicker:              time.NewTicker(5 * time.Minute), // Check every 5 minutes
		vehicleNotificationService: services.NewVehicleNotificationService(),
		enableGPSSmoothing:         true, // Enable GPS smoothing by default
		enableGPSValidation:        true, // Enable GPS validation by default
	}
}

// NewServerWithController creates a new TCP server instance with a shared control controller
func NewServerWithController(port string, sharedController *controllers.ControlController) *Server {
	return &Server{
		port:                       port,
		controlController:          sharedController,
		deviceConnections:          make(map[string]*DeviceConnection),
		timeoutTicker:              time.NewTicker(5 * time.Minute), // Check every 5 minutes
		vehicleNotificationService: services.NewVehicleNotificationService(),
		enableGPSSmoothing:         true, // Enable GPS smoothing by default
		enableGPSValidation:        true, // Enable GPS validation by default
	}
}

// Start starts the TCP server
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %v", err)
	}

	s.listener = listener
	defer listener.Close()

	colors.PrintServer("📡", "GT06 TCP Server is running on port %s", s.port)
	colors.PrintConnection("📶", "Waiting for IoT device connections...")
	colors.PrintData("💾", "Database connectivity enabled - GPS data will be saved")
	colors.PrintControl("Oil/Electricity control system enabled - Ready for commands")

	// Show GPS processing features
	if s.enableGPSValidation {
		colors.PrintInfo("📍 GPS Validation: Enabled (Nepal region, accuracy, erratic detection)")
	} else {
		colors.PrintWarning("📍 GPS Validation: Disabled")
	}

	if s.enableGPSSmoothing {
		colors.PrintInfo("📍 GPS Smoothing: Enabled (reduces zigzag patterns)")
	} else {
		colors.PrintWarning("📍 GPS Smoothing: Disabled")
	}

	// Start device timeout monitor
	go s.monitorDeviceTimeouts()

	// Start periodic cleanup of vehicle notification states
	go s.cleanupVehicleNotificationStates()

	for {
		conn, err := listener.Accept()
		if err != nil {
			colors.PrintError("Error accepting TCP connection: %v", err)
			continue
		}

		// Handle each connection in a separate goroutine
		go s.handleConnection(conn)
	}
}

// ConfigureGPSProcessing sets GPS processing options
func (s *Server) ConfigureGPSProcessing(enableValidation, enableSmoothing bool) {
	s.enableGPSValidation = enableValidation
	s.enableGPSSmoothing = enableSmoothing
	colors.PrintInfo("📍 GPS Processing configured: Validation=%v, Smoothing=%v", enableValidation, enableSmoothing)
}

// isDeviceRegistered checks if a device with given IMEI exists in the database
func (s *Server) isDeviceRegistered(imei string) bool {
	var device models.Device
	err := db.GetDB().Where("imei = ?", imei).First(&device).Error
	return err == nil
}

// handleConnection handles incoming IoT device connections
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	colors.PrintConnection("📱", "New IoT Device connected: %s", conn.RemoteAddr())

	// Create GT06 decoder for this connection
	decoder := protocol.NewGT06Decoder()
	deviceIMEI := ""

	// Set connection timeout
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	// Buffer for reading data
	buffer := make([]byte, 1024)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err.Error() == "EOF" {
				colors.PrintConnection("📱", "IoT Device disconnected: %s", conn.RemoteAddr())
				break
			}
			colors.PrintError("Error reading from connection %s: %v", conn.RemoteAddr(), err)
			break
		}

		if n > 0 {
			// Log raw data received
			colors.PrintData("📦", "Raw data from %s: %X", conn.RemoteAddr(), buffer[:n])

			// Process data through GT06 decoder
			packets, err := decoder.AddData(buffer[:n])
			if err != nil {
				colors.PrintError("Error decoding data from %s: %v", conn.RemoteAddr(), err)
				continue
			}

			// Process each decoded packet
			for _, packet := range packets {
				// Add null safety check
				if packet == nil {
					colors.PrintWarning("Received nil packet from %s, skipping...", conn.RemoteAddr())
					continue
				}

				colors.PrintData("📋", "Decoded packet from %s:", conn.RemoteAddr())

				// Convert packet to JSON for pretty printing
				jsonData, err := json.MarshalIndent(packet, "", "  ")
				if err != nil {
					colors.PrintError("Error marshaling packet to JSON: %v", err)
					colors.PrintDebug("Packet: %+v", packet)
				} else {
					colors.PrintDebug("Packet Data:\n%s", jsonData)
				}

				// Add additional safety checks for packet fields
				if packet.ProtocolName == "" {
					colors.PrintWarning("Packet with empty protocol name from %s, skipping...", conn.RemoteAddr())
					continue
				}

				// Handle different packet types
				switch packet.ProtocolName {
				case "LOGIN":
					deviceIMEI = s.handleLoginPacket(packet, conn)
				case "GPS_LBS", "GPS_LBS_STATUS", "GPS_LBS_DATA", "GPS_LBS_STATUS_A0":
					s.handleGPSPacket(packet, conn, deviceIMEI)
				case "STATUS_INFO":
					s.handleStatusPacket(packet, conn, deviceIMEI)
				case "ALARM_DATA":
					s.handleAlarmPacket(packet, conn)
				}

				// Send response if required
				if packet.NeedsResponse {
					s.sendResponse(packet, conn, decoder)
				}
			}
		}
	}
}

// handleLoginPacket processes login packets and returns the device IMEI
func (s *Server) handleLoginPacket(packet *protocol.DecodedPacket, conn net.Conn) string {
	deviceIMEI := packet.TerminalID
	colors.PrintConnection("🔐", "Device login: %s from %s", deviceIMEI, conn.RemoteAddr())

	// Register connection with control controller
	s.controlController.RegisterConnection(deviceIMEI, conn)

	// Update device activity
	s.updateDeviceActivity(deviceIMEI, conn)

	// Check if device is registered in database
	if s.isDeviceRegistered(deviceIMEI) {
		colors.PrintSuccess("✅ Device %s is registered in database", deviceIMEI)
	} else {
		colors.PrintWarning("⚠️ Device %s is not registered in database", deviceIMEI)
	}

	return deviceIMEI
}

// handleGPSPacket processes GPS packets
func (s *Server) handleGPSPacket(packet *protocol.DecodedPacket, conn net.Conn, deviceIMEI string) {
	// Update device activity
	s.updateDeviceActivity(deviceIMEI, conn)

	// Validate GPS data exists
	if packet.Latitude == nil || packet.Longitude == nil {
		colors.PrintWarning("⚠️ Skipping GPS: Missing coordinates (Lat=%v, Lng=%v)", packet.Latitude, packet.Longitude)
		return
	}

	lat := *packet.Latitude
	lng := *packet.Longitude

	// REMOVED: Incorrect latitude conversion - Nepal should have positive latitude
	// if lat < 0 {
	// 	lat = -lat
	// 	packet.Latitude = &lat
	// }

	// Enhanced coordinate range validation for Nepal region
	// Nepal coordinates: Lat: 26.3478° to 30.4465°, Lng: 80.0586° to 88.2014°
	if s.enableGPSValidation && (lat < 26.0 || lat > 31.0 || lng < 79.0 || lng > 89.0) {
		colors.PrintWarning("📍 Invalid GPS coordinates (outside Nepal region): Lat=%.12f, Lng=%.12f", lat, lng)
		return
	}

	// Enhanced GPS accuracy validation
	if s.enableGPSValidation && packet.Satellites != nil && *packet.Satellites < 3 {
		colors.PrintWarning("📍 Poor GPS signal: Only %d satellites", *packet.Satellites)
		return
	}

	// Check if GPS is positioned - LESS STRICT: Accept if satellites >= 3 even if not positioned
	if s.enableGPSValidation && packet.GPSPositioned != nil && !*packet.GPSPositioned {
		// Only reject if we also have poor satellite signal
		if packet.Satellites == nil || *packet.Satellites < 3 {
			colors.PrintWarning("📍 GPS not positioned properly and poor satellite signal")
			return
		}
		// If we have good satellite signal (>=3), accept the GPS data even if not positioned
		colors.PrintWarning("⚠️ GPS not positioned but good satellite signal (%d satellites) - accepting", *packet.Satellites)
	}

	colors.PrintData("🌍", "Processing GPS: Lat=%.12f, Lng=%.12f, Speed=%v km/h, Ignition=%s, Satellites=%v",
		lat, lng, packet.Speed, packet.Ignition, packet.Satellites)

	// Step 1: Check ignition status requirement (LESS STRICT)
	shouldAcceptGPS := s.shouldAcceptGPSBasedOnIgnition(deviceIMEI, packet)
	if !shouldAcceptGPS {
		colors.PrintWarning("🚫 GPS rejected: Ignition is OFF - ignoring completely")
		return
	}

	// Step 2: Check for duplicate coordinates (INCREASED THRESHOLD)
	if s.isDuplicateCoordinates(deviceIMEI, lat, lng) {
		colors.PrintWarning("🚫 GPS rejected: Duplicate coordinates")
		return
	}

	// Step 3: Check for erratic GPS (sudden large jumps or unrealistic paths)
	if s.enableGPSValidation && s.isErraticGPS(deviceIMEI, lat, lng) {
		colors.PrintWarning("🚫 GPS rejected: Erratic GPS coordinates")
		return
	}

	// Smooth GPS coordinates
	var smoothedLat, smoothedLng float64
	if s.enableGPSSmoothing {
		smoothedLat, smoothedLng = s.smoothGPSCoordinates(deviceIMEI, lat, lng)
	} else {
		smoothedLat, smoothedLng = lat, lng
	}

	// Save GPS data and broadcast to WebSocket clients
	if deviceIMEI != "" && s.isDeviceRegistered(deviceIMEI) {
		gpsData := s.buildGPSData(packet, deviceIMEI)

		// Apply smoothed coordinates to the GPS data
		gpsData.Latitude = &smoothedLat
		gpsData.Longitude = &smoothedLng

		// STEP 1: Check and send vehicle notifications FIRST (before saving to database)
		var notificationError error
		if s.vehicleNotificationService != nil {
			colors.PrintInfo("🔔 Checking notifications BEFORE saving to database")
			notificationError = s.vehicleNotificationService.CheckAndSendVehicleNotifications(&gpsData)
			if notificationError != nil {
				colors.PrintError("❌ Notification check failed: %v - NOT saving to database", notificationError)
				return // Don't save to database if notification check fails
			}
			colors.PrintSuccess("✅ Notification check completed successfully")
		}

		// STEP 2: Save to database only if notification check succeeded
		if err := db.GetDB().Create(&gpsData).Error; err != nil {
			colors.PrintError("Error saving GPS data: %v", err)
		} else {
			colors.PrintSuccess("✅ GPS data saved for device %s (Original: %.12f,%.12f -> Smoothed: %.12f,%.12f)",
				deviceIMEI, lat, lng, smoothedLat, smoothedLng)

			// STEP 3: Broadcast the new full GPS data object over WebSocket
			if http.WSHub != nil {
				go http.WSHub.BroadcastFullGPSUpdate(&gpsData)
			}
		}
	}
}

// shouldAcceptGPSBasedOnIgnition checks if GPS should be accepted based on ignition status
func (s *Server) shouldAcceptGPSBasedOnIgnition(imei string, packet *protocol.DecodedPacket) bool {
	// If ignition is explicitly OFF, still accept GPS data but log it
	if packet.Ignition == "OFF" {
		colors.PrintWarning("⚠️ GPS from %s with ignition OFF - accepting anyway for route continuity", imei)
		return true // Changed from false to true
	}

	// If ignition is ON or empty, accept GPS data
	if packet.Ignition == "ON" || packet.Ignition == "" {
		return true
	}

	// For any other ignition status, accept GPS data
	return true
}

// isDuplicateCoordinates checks if the coordinates are duplicate (within 5 meters)
func (s *Server) isDuplicateCoordinates(imei string, lat, lng float64) bool {
	// Get the latest GPS data for this device
	var latestGPS models.GPSData
	err := db.GetDB().Where("imei = ? AND latitude IS NOT NULL AND longitude IS NOT NULL",
		imei).Order("timestamp DESC").First(&latestGPS).Error

	if err != nil {
		// No previous GPS data, not a duplicate
		return false
	}

	// Calculate distance between current and latest coordinates
	distance := s.calculateDistance(lat, lng, *latestGPS.Latitude, *latestGPS.Longitude)

	// If distance is less than 5 meters, consider it duplicate (reduced from 10m)
	if distance < 0.005 { // 5 meters = 0.005 km
		colors.PrintDebug("📍 Duplicate coordinates detected: Distance=%.3f km", distance)
		return true
	}

	return false
}

// calculateDistance calculates the distance between two coordinates using Haversine formula
func (s *Server) calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371 // Earth's radius in kilometers

	// Convert to radians
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180

	// Haversine formula
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// isErraticGPS checks if GPS coordinates are too erratic (sudden large jumps)
func (s *Server) isErraticGPS(imei string, lat, lng float64) bool {
	// Get the last 3 GPS points for this device
	var recentGPS []models.GPSData
	err := db.GetDB().Where("imei = ? AND latitude IS NOT NULL AND longitude IS NOT NULL",
		imei).Order("timestamp DESC").Limit(3).Find(&recentGPS).Error

	if err != nil || len(recentGPS) < 2 {
		// Not enough data to determine if erratic
		return false
	}

	// Calculate distance to the most recent point
	latestPoint := recentGPS[0]
	distance := s.calculateDistance(lat, lng, *latestPoint.Latitude, *latestPoint.Longitude)

	// If the jump is more than 1km in a single update, it's likely erratic
	if distance > 1.0 {
		colors.PrintWarning("📍 Erratic GPS detected: Jump of %.3f km", distance)
		return true
	}

	// Check if the new point creates an unrealistic path
	if len(recentGPS) >= 3 {
		// Calculate the angle between the last two segments
		// If the angle is too sharp (> 150 degrees), it might be erratic
		prevPoint := recentGPS[1]

		// Calculate vectors
		vec1Lat := *latestPoint.Latitude - *prevPoint.Latitude
		vec1Lng := *latestPoint.Longitude - *prevPoint.Longitude
		vec2Lat := lat - *latestPoint.Latitude
		vec2Lng := lng - *latestPoint.Longitude

		// Calculate dot product
		dotProduct := vec1Lat*vec2Lat + vec1Lng*vec2Lng
		magnitude1 := math.Sqrt(vec1Lat*vec1Lat + vec1Lng*vec1Lng)
		magnitude2 := math.Sqrt(vec2Lat*vec2Lat + vec2Lng*vec2Lng)

		if magnitude1 > 0 && magnitude2 > 0 {
			cosAngle := dotProduct / (magnitude1 * magnitude2)
			// Clamp to [-1, 1] to avoid math domain errors
			if cosAngle > 1 {
				cosAngle = 1
			} else if cosAngle < -1 {
				cosAngle = -1
			}
			angle := math.Acos(cosAngle) * 180 / math.Pi

			// If angle is greater than 150 degrees, it's likely erratic
			if angle > 150 {
				colors.PrintWarning("📍 Erratic GPS detected: Sharp angle of %.1f degrees", angle)
				return true
			}
		}
	}

	return false
}

// smoothGPSCoordinates applies simple smoothing to reduce zigzag patterns
func (s *Server) smoothGPSCoordinates(imei string, lat, lng float64) (float64, float64) {
	// Get the last 2 GPS points for this device
	var recentGPS []models.GPSData
	err := db.GetDB().Where("imei = ? AND latitude IS NOT NULL AND longitude IS NOT NULL",
		imei).Order("timestamp DESC").Limit(2).Find(&recentGPS).Error

	if err != nil || len(recentGPS) < 1 {
		// Not enough data for smoothing, return original coordinates
		return lat, lng
	}

	// Simple moving average with 70% weight for new point, 30% for previous
	prevLat := *recentGPS[0].Latitude
	prevLng := *recentGPS[0].Longitude

	// Apply smoothing
	smoothedLat := 0.7*lat + 0.3*prevLat
	smoothedLng := 0.7*lng + 0.3*prevLng

	colors.PrintDebug("📍 GPS smoothing: Original(%.12f,%.12f) -> Smoothed(%.12f,%.12f)",
		lat, lng, smoothedLat, smoothedLng)

	return smoothedLat, smoothedLng
}

// handleStatusPacket processes status packets
func (s *Server) handleStatusPacket(packet *protocol.DecodedPacket, conn net.Conn, deviceIMEI string) {
	// Update device activity
	s.updateDeviceActivity(deviceIMEI, conn)

	colors.PrintData("📊", "Status info from %s: Ignition=%s, Voltage=%v, GSM Signal=%v",
		conn.RemoteAddr(), packet.Ignition, packet.Voltage, packet.GSMSignal)

	// Validate for duplicate status data
	if s.isDuplicateStatusData(deviceIMEI, packet) {
		return
	}

	// Save status data to database and broadcast to WebSocket clients
	if deviceIMEI != "" && s.isDeviceRegistered(deviceIMEI) {
		// Get the latest GPS data for this device to preserve location
		var latestGPS models.GPSData
		hasLatestGPS := false
		if err := db.GetDB().Where("imei = ? AND latitude IS NOT NULL AND longitude IS NOT NULL",
			deviceIMEI).Order("timestamp DESC").First(&latestGPS).Error; err == nil {
			hasLatestGPS = true
		}

		statusData := s.buildStatusData(packet, deviceIMEI)

		// Preserve latest GPS coordinates if status packet doesn't have them
		if !hasLatestGPS && packet.Latitude == nil && packet.Longitude == nil {
			if hasLatestGPS {
				statusData.Latitude = latestGPS.Latitude
				statusData.Longitude = latestGPS.Longitude
				statusData.Speed = latestGPS.Speed
				statusData.Course = latestGPS.Course
			}
		}

		// STEP 1: Check and send vehicle notifications FIRST (before saving to database)
		var notificationError error
		if s.vehicleNotificationService != nil {
			colors.PrintInfo("🔔 Checking status notifications BEFORE saving to database")
			notificationError = s.vehicleNotificationService.CheckAndSendVehicleNotifications(&statusData)
			if notificationError != nil {
				colors.PrintError("❌ Status notification check failed: %v - NOT saving to database", notificationError)
				return // Don't save to database if notification check fails
			}
			colors.PrintSuccess("✅ Status notification check completed successfully")
		}

		// STEP 2: Save to database only if notification check succeeded
		if err := db.GetDB().Create(&statusData).Error; err != nil {
			colors.PrintError("Error saving status data: %v", err)
		} else {
			colors.PrintSuccess("✅ Status data saved for device %s", deviceIMEI)

			// Broadcast status update to WebSocket clients
			if http.WSHub != nil {
				go http.WSHub.BroadcastStatusUpdate(&statusData, "", "")
			}
		}
	}
}

// handleAlarmPacket processes alarm packets
func (s *Server) handleAlarmPacket(packet *protocol.DecodedPacket, conn net.Conn) {
	colors.PrintWarning("🚨 Alarm data received from %s: %+v", conn.RemoteAddr(), packet)
}

// sendResponse sends a response to the device
func (s *Server) sendResponse(packet *protocol.DecodedPacket, conn net.Conn, decoder *protocol.GT06Decoder) {
	response := decoder.GenerateResponse(uint16(packet.SerialNumber), packet.Protocol)
	conn.Write(response)
	colors.PrintData("📤", "Response sent to device: %X", response)
}

// buildGPSData creates a GPSData model from a decoded packet
func (s *Server) buildGPSData(packet *protocol.DecodedPacket, deviceIMEI string) models.GPSData {
	gpsData := models.GPSData{
		IMEI:         deviceIMEI,
		Timestamp:    packet.Timestamp,
		ProtocolName: packet.ProtocolName,
		RawPacket:    packet.Raw,
	}

	// GPS location data with enhanced precision
	if packet.Latitude != nil {
		gpsData.Latitude = packet.Latitude
	}
	if packet.Longitude != nil {
		gpsData.Longitude = packet.Longitude
	}
	if packet.Speed != nil {
		speed := int(*packet.Speed)
		gpsData.Speed = &speed
	}
	if packet.Course != nil {
		course := int(*packet.Course)
		gpsData.Course = &course
	}
	if packet.Altitude != nil {
		gpsData.Altitude = packet.Altitude
	}
	if packet.Satellites != nil {
		satellites := int(*packet.Satellites)
		gpsData.Satellites = &satellites
	}

	// GPS status
	if packet.GPSRealTime != nil {
		gpsData.GPSRealTime = packet.GPSRealTime
	}
	if packet.GPSPositioned != nil {
		gpsData.GPSPositioned = packet.GPSPositioned
	}

	// Device status
	gpsData.Ignition = packet.Ignition
	gpsData.Charger = packet.Charger
	gpsData.GPSTracking = packet.GPSTracking
	gpsData.OilElectricity = packet.OilElectricity
	gpsData.DeviceStatus = packet.DeviceStatus

	// LBS data (cell tower information)
	if packet.MCC != nil {
		mcc := int(*packet.MCC)
		gpsData.MCC = &mcc
	}
	if packet.MNC != nil {
		mnc := int(*packet.MNC)
		gpsData.MNC = &mnc
	}
	if packet.LAC != nil {
		lac := int(*packet.LAC)
		gpsData.LAC = &lac
	}
	if packet.CellID != nil {
		cellID := int(*packet.CellID)
		gpsData.CellID = &cellID
	}

	return gpsData
}

// buildStatusData creates a GPSData model for status information
func (s *Server) buildStatusData(packet *protocol.DecodedPacket, deviceIMEI string) models.GPSData {
	statusData := models.GPSData{
		IMEI:         deviceIMEI,
		Timestamp:    packet.Timestamp,
		ProtocolName: packet.ProtocolName,
		RawPacket:    packet.Raw,
	}

	// Device status
	statusData.Ignition = packet.Ignition
	statusData.Charger = packet.Charger
	statusData.GPSTracking = packet.GPSTracking
	statusData.OilElectricity = packet.OilElectricity
	statusData.DeviceStatus = packet.DeviceStatus

	// Signal & Power
	if packet.Voltage != nil {
		voltageLevel := int(packet.Voltage.Level)
		statusData.VoltageLevel = &voltageLevel
		statusData.VoltageStatus = packet.Voltage.Status
	}
	if packet.GSMSignal != nil {
		gsmSignal := int(packet.GSMSignal.Level)
		statusData.GSMSignal = &gsmSignal
		statusData.GSMStatus = packet.GSMSignal.Status
	}

	// Alarm data
	if packet.Alarm != nil {
		statusData.AlarmActive = packet.Alarm.Active
		statusData.AlarmType = packet.Alarm.Type
		statusData.AlarmCode = packet.Alarm.Code
	}

	return statusData
}

// isDuplicateStatusData checks if status data is duplicate (within 1 minute)
func (s *Server) isDuplicateStatusData(imei string, packet *protocol.DecodedPacket) bool {
	// Get the latest status data for this device
	var latestStatus models.GPSData
	err := db.GetDB().Where("imei = ? AND ignition IS NOT NULL AND ignition != ''",
		imei).Order("timestamp DESC").First(&latestStatus).Error

	if err != nil {
		// No previous status data, not a duplicate
		return false
	}

	// Check if the latest status data is within 1 minute
	timeDiff := packet.Timestamp.Sub(latestStatus.Timestamp)
	if timeDiff < time.Minute {
		// Check if ignition status is the same
		if latestStatus.Ignition == packet.Ignition {
			colors.PrintWarning("🚫 Status data rejected: Duplicate status within 1 minute")
			return true
		}
	}

	return false
}

// monitorDeviceTimeouts monitors device connections for timeouts
func (s *Server) monitorDeviceTimeouts() {
	colors.PrintInfo("⏰ Starting device timeout monitor...")

	for range s.timeoutTicker.C {
		s.checkDevicesForInactiveStatus()
	}
}

// checkDevicesForInactiveStatus checks all devices for inactive status
func (s *Server) checkDevicesForInactiveStatus() {
	var devices []models.Device
	if err := db.GetDB().Find(&devices).Error; err != nil {
		colors.PrintError("Error fetching devices for inactive check: %v", err)
		return
	}

	now := config.GetCurrentTime()
	oneHourAgo := now.Add(-time.Hour)

	for _, device := range devices {
		// Get latest GPS data for this device
		var latestGPS models.GPSData
		err := db.GetDB().Where("imei = ?", device.IMEI).
			Order("timestamp DESC").
			First(&latestGPS).Error

		if err != nil {
			// No GPS data found at all - this is true "no data" case
			// Device is registered but never sent any GPS data to database
			colors.PrintWarning("📱 Device %s has no GPS data in database, broadcasting no-data status", device.IMEI)
			s.broadcastNoDataStatus(device.IMEI)
			continue
		}

		// ENHANCED FIX: Device has GPS data - always show vehicle status based on GPS data
		// Check if GPS data is older than 1 hour to show "inactive"
		if latestGPS.Timestamp.Before(oneHourAgo) {
			// GPS data is older than 1 hour - show as inactive
			colors.PrintInfo("📱 Device %s last GPS data is %v old, broadcasting inactive status (not no-data)",
				device.IMEI, now.Sub(latestGPS.Timestamp))
			s.broadcastInactiveStatusWithGPS(device.IMEI, &latestGPS)
		} else {
			// GPS data is recent (< 1 hour) - broadcast current vehicle status based on GPS data
			s.broadcastVehicleStatusFromGPS(device.IMEI, &latestGPS)
		}
	}
}

// broadcastStoppedStatus broadcasts stopped status for a device (1-2 hours without data)
func (s *Server) broadcastStoppedStatus(imei string) {
	// Get vehicle info for WebSocket broadcast
	var vehicle models.Vehicle
	vehicleReg := ""
	if err := db.GetDB().Where("imei = ?", imei).First(&vehicle).Error; err == nil {
		vehicleReg = vehicle.RegNo
	}

	// Broadcast device as stopped
	if http.WSHub != nil {
		http.WSHub.BroadcastDeviceStatus(imei, "stopped", vehicleReg)
	}
}

// broadcastInactiveStatus broadcasts inactive status for a device (2+ hours without data)
func (s *Server) broadcastInactiveStatus(imei string) {
	// Get vehicle info for WebSocket broadcast
	var vehicle models.Vehicle
	vehicleReg := ""
	if err := db.GetDB().Where("imei = ?", imei).First(&vehicle).Error; err == nil {
		vehicleReg = vehicle.RegNo
	}

	// Broadcast device as inactive
	if http.WSHub != nil {
		http.WSHub.BroadcastDeviceStatus(imei, "inactive", vehicleReg)
	}
}

// broadcastNoDataStatus broadcasts no-data status for a device (never sent GPS data)
func (s *Server) broadcastNoDataStatus(imei string) {
	// Get vehicle info for WebSocket broadcast
	var vehicle models.Vehicle
	vehicleReg := ""
	if err := db.GetDB().Where("imei = ?", imei).First(&vehicle).Error; err == nil {
		vehicleReg = vehicle.RegNo
	}

	// Broadcast device as no-data
	if http.WSHub != nil {
		http.WSHub.BroadcastDeviceStatus(imei, "no-data", vehicleReg)
	}
}

// broadcastInactiveStatusWithGPS broadcasts inactive status with GPS data
func (s *Server) broadcastInactiveStatusWithGPS(imei string, gpsData *models.GPSData) {
	// Get vehicle info for WebSocket broadcast
	var vehicle models.Vehicle
	vehicleReg := ""
	vehicleName := ""
	if err := db.GetDB().Where("imei = ?", imei).First(&vehicle).Error; err == nil {
		vehicleReg = vehicle.RegNo
		vehicleName = vehicle.Name
	}

	// Broadcast inactive status with GPS data
	if http.WSHub != nil {
		http.WSHub.BroadcastStatusUpdate(gpsData, vehicleName, vehicleReg)
	}
}

// broadcastVehicleStatusFromGPS broadcasts vehicle status based on GPS data
func (s *Server) broadcastVehicleStatusFromGPS(imei string, gpsData *models.GPSData) {
	// Get vehicle info for WebSocket broadcast
	var vehicle models.Vehicle
	vehicleReg := ""
	vehicleName := ""
	if err := db.GetDB().Where("imei = ?", imei).First(&vehicle).Error; err == nil {
		vehicleReg = vehicle.RegNo
		vehicleName = vehicle.Name
	}

	// Broadcast vehicle status based on GPS data
	if http.WSHub != nil {
		colors.PrintConnection("📡", "Broadcasting vehicle status for IMEI %s: %s (%s)", imei, vehicleName, vehicleReg)
		http.WSHub.BroadcastStatusUpdate(gpsData, vehicleName, vehicleReg)
	} else {
		colors.PrintWarning("WebSocket hub not available for broadcasting vehicle status")
	}
}

// updateDeviceActivity updates the last activity time for a device
func (s *Server) updateDeviceActivity(imei string, conn net.Conn) {
	s.connectionMutex.Lock()
	defer s.connectionMutex.Unlock()

	if deviceConn, exists := s.deviceConnections[imei]; exists {
		deviceConn.LastActivity = config.GetCurrentTime()
		deviceConn.IsActive = true
		colors.PrintConnection("📱", "Updated device activity for IMEI %s", imei)
	} else {
		s.deviceConnections[imei] = &DeviceConnection{
			Conn:         conn,
			LastActivity: config.GetCurrentTime(),
			IMEI:         imei,
			IsActive:     true,
		}
		colors.PrintConnection("📱", "Registered new device connection for IMEI %s", imei)
	}
}

// removeDeviceConnection removes a device connection
func (s *Server) removeDeviceConnection(imei string) {
	s.connectionMutex.Lock()
	defer s.connectionMutex.Unlock()

	if deviceConn, exists := s.deviceConnections[imei]; exists {
		deviceConn.IsActive = false
		colors.PrintConnection("📱", "Device %s marked as inactive", imei)
	} else {
		colors.PrintWarning("Attempted to remove non-existent device connection for IMEI %s", imei)
	}
}

// cleanupVehicleNotificationStates periodically cleans up old vehicle notification states
func (s *Server) cleanupVehicleNotificationStates() {
	colors.PrintInfo("🧹 Starting vehicle notification state cleanup...")

	// Run cleanup every 6 hours
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		if s.vehicleNotificationService != nil {
			s.vehicleNotificationService.CleanupOldVehicleStates()
		}
	}
}
