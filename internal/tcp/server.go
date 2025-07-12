package tcp

import (
	"encoding/json"
	"fmt"
	"luna_iot_server/internal/db"
	"luna_iot_server/internal/http"
	"luna_iot_server/internal/http/controllers"
	"luna_iot_server/internal/models"
	"luna_iot_server/internal/protocol"
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
}

// NewServer creates a new TCP server instance
func NewServer(port string) *Server {
	return &Server{
		port:              port,
		controlController: controllers.NewControlController(),
		deviceConnections: make(map[string]*DeviceConnection),
		timeoutTicker:     time.NewTicker(5 * time.Minute), // Check every 5 minutes
	}
}

// NewServerWithController creates a new TCP server instance with a shared control controller
func NewServerWithController(port string, sharedController *controllers.ControlController) *Server {
	return &Server{
		port:              port,
		controlController: sharedController,
		deviceConnections: make(map[string]*DeviceConnection),
		timeoutTicker:     time.NewTicker(5 * time.Minute), // Check every 5 minutes
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

	// Start device timeout monitor
	go s.monitorDeviceTimeouts()

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

// isDeviceRegistered checks if a device with given IMEI exists in the database
func (s *Server) isDeviceRegistered(imei string) bool {
	var device models.Device
	err := db.GetDB().Where("imei = ?", imei).First(&device).Error
	return err == nil
}

// handleConnection handles incoming IoT device connections
func (s *Server) handleConnection(conn net.Conn) {
	// Track device IMEI for this connection
	var deviceIMEI string

	defer func() {
		conn.Close()
		// Unregister connection when it closes
		if deviceIMEI != "" {
			s.controlController.UnregisterConnection(deviceIMEI)
			s.removeDeviceConnection(deviceIMEI)

			// ENHANCED FIX: When device disconnects, don't send device status - let the monitoring system handle it
			// The checkDevicesForInactiveStatus will properly broadcast status based on GPS data age
			colors.PrintInfo("📱 Device %s disconnected, monitoring system will handle status updates based on GPS data", deviceIMEI)
		}
	}()

	colors.PrintConnection("📱", "IoT Device connected from %s", conn.RemoteAddr())

	// Create a new GT06 decoder instance for this connection
	decoder := protocol.NewGT06Decoder()

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

// handleLoginPacket processes device login packets
func (s *Server) handleLoginPacket(packet *protocol.DecodedPacket, conn net.Conn) string {
	// Add safety checks
	if packet == nil {
		colors.PrintError("Received nil packet in handleLoginPacket")
		return ""
	}

	if packet.TerminalID == "" {
		colors.PrintWarning("Login packet with empty TerminalID from %s", conn.RemoteAddr())
		return ""
	}

	if len(packet.TerminalID) < 16 {
		colors.PrintWarning("Login packet with invalid TerminalID length (%d) from %s", len(packet.TerminalID), conn.RemoteAddr())
		return ""
	}

	potentialIMEI := packet.TerminalID[:16]

	// Validate device registration
	if !s.isDeviceRegistered(potentialIMEI) {
		colors.PrintError("Unauthorized device: %s", potentialIMEI)
		conn.Close()
		return ""
	}

	colors.PrintSuccess("Authorized device login: %s", potentialIMEI)
	s.controlController.RegisterConnection(potentialIMEI, conn)

	// Get vehicle info for WebSocket broadcast
	var vehicle models.Vehicle
	vehicleReg := ""
	if err := db.GetDB().Where("imei = ?", potentialIMEI).First(&vehicle).Error; err == nil {
		vehicleReg = vehicle.RegNo
	}

	// Broadcast device connection to WebSocket clients
	if http.WSHub != nil {
		http.WSHub.BroadcastDeviceStatus(potentialIMEI, "connected", vehicleReg)
	}

	return potentialIMEI
}

// handleGPSPacket processes GPS data packets with advanced filtering
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

	// Convert negative latitude to positive (remove minus sign)
	if lat < 0 {
		lat = -lat
		packet.Latitude = &lat
	}

	// Basic coordinate range validation
	if lat <= 0 || lat > 90 || lng < -180 || lng > 180 {
		colors.PrintWarning("📍 Invalid GPS coordinates (out of range): Lat=%.12f, Lng=%.12f", lat, lng)
		return
	}

	colors.PrintData("🌍", "Processing GPS: Lat=%.12f, Lng=%.12f, Speed=%v km/h, Ignition=%s",
		lat, lng, packet.Speed, packet.Ignition)

	// Step 1: Check ignition status requirement
	shouldAcceptGPS := s.shouldAcceptGPSBasedOnIgnition(deviceIMEI, packet)
	if !shouldAcceptGPS {
		colors.PrintWarning("🚫 GPS rejected: Ignition is OFF or not valid")
		// Still save status data without coordinates
		s.saveStatusOnlyData(packet, deviceIMEI)
		return
	}

	// Step 2: Check for duplicate coordinates
	if s.isDuplicateCoordinates(deviceIMEI, lat, lng) {
		colors.PrintWarning("🚫 GPS rejected: Duplicate coordinates")
		return
	}

	// Step 3: Advanced GPS accuracy validation with late data handling
	isValidGPS, processedLat, processedLng := s.validateAndProcessGPSWithHistory(deviceIMEI, lat, lng, packet)
	if !isValidGPS {
		colors.PrintWarning("🚫 GPS rejected: Failed accuracy validation")
		return
	}

	// Update packet with processed coordinates
	packet.Latitude = &processedLat
	packet.Longitude = &processedLng

	// Save GPS data and broadcast to WebSocket clients
	if deviceIMEI != "" && s.isDeviceRegistered(deviceIMEI) {
		gpsData := s.buildGPSData(packet, deviceIMEI)

		if err := db.GetDB().Create(&gpsData).Error; err != nil {
			colors.PrintError("Error saving GPS data: %v", err)
		} else {
			colors.PrintSuccess("✅ GPS data saved for device %s (Lat=%.12f, Lng=%.12f)",
				deviceIMEI, processedLat, processedLng)

			// Broadcast the new full GPS data object over WebSocket
			if http.WSHub != nil {
				go http.WSHub.BroadcastFullGPSUpdate(&gpsData)
			}
		}
	}
}

// shouldAcceptGPSBasedOnIgnition checks if GPS should be accepted based on ignition status
func (s *Server) shouldAcceptGPSBasedOnIgnition(imei string, packet *protocol.DecodedPacket) bool {
	// First check current packet ignition
	currentIgnition := packet.Ignition

	// If current packet has ignition data
	if currentIgnition != "" {
		if currentIgnition == "ON" {
			colors.PrintData("🔑", "Ignition ON in current packet - accepting GPS")
			return true
		} else {
			colors.PrintWarning("🔑 Ignition OFF in current packet - rejecting GPS")
			return false
		}
	}

	// If no ignition in current packet, check database for last known ignition status
	var lastStatus models.GPSData
	err := db.GetDB().Where("imei = ? AND ignition IS NOT NULL AND ignition != ''", imei).
		Order("timestamp DESC").
		First(&lastStatus).Error

	if err != nil {
		colors.PrintWarning("🔑 No ignition history found for device %s - rejecting GPS", imei)
		return false
	}

	if lastStatus.Ignition == "ON" {
		colors.PrintData("🔑", "Last known ignition status: ON - accepting GPS")
		return true
	} else {
		colors.PrintWarning("🔑 Last known ignition status: %s - rejecting GPS", lastStatus.Ignition)
		return false
	}
}

// isDuplicateCoordinates checks if the coordinates are the same as the last saved ones
func (s *Server) isDuplicateCoordinates(imei string, lat, lng float64) bool {
	var lastGPS models.GPSData
	err := db.GetDB().Where("imei = ? AND latitude IS NOT NULL AND longitude IS NOT NULL", imei).
		Order("timestamp DESC").
		First(&lastGPS).Error

	if err != nil {
		// No previous GPS data, not a duplicate
		return false
	}

	// Check if coordinates are exactly the same (with precision tolerance)
	const tolerance = 0.000001 // ~1 meter precision
	latDiff := math.Abs(*lastGPS.Latitude - lat)
	lngDiff := math.Abs(*lastGPS.Longitude - lng)

	isDuplicate := latDiff < tolerance && lngDiff < tolerance
	if isDuplicate {
		colors.PrintWarning("📍 Duplicate coordinates detected: Last(%.12f,%.12f) Current(%.12f,%.12f)",
			*lastGPS.Latitude, *lastGPS.Longitude, lat, lng)
	}

	return isDuplicate
}

// validateAndProcessGPSWithHistory validates GPS data against historical data and handles late data
func (s *Server) validateAndProcessGPSWithHistory(imei string, lat, lng float64, packet *protocol.DecodedPacket) (bool, float64, float64) {
	// Get last 10 GPS points with valid coordinates for analysis
	var recentGPSData []models.GPSData
	err := db.GetDB().Where("imei = ? AND latitude IS NOT NULL AND longitude IS NOT NULL", imei).
		Order("timestamp DESC").
		Limit(10).
		Find(&recentGPSData).Error

	if err != nil || len(recentGPSData) == 0 {
		// No previous GPS data, accept first point
		colors.PrintData("🔍", "No GPS history found - accepting first point")
		return true, lat, lng
	}

	// Analyze if this is late data (older than the most recent data)
	mostRecentTime := recentGPSData[0].Timestamp
	isLateData := packet.Timestamp.Before(mostRecentTime)

	if isLateData {
		colors.PrintData("⏰", "Late data detected: packet time=%v, most recent=%v",
			packet.Timestamp, mostRecentTime)
		return s.processLateData(imei, lat, lng, packet, recentGPSData)
	}

	// Process current/future data
	return s.processCurrentData(lat, lng, packet, recentGPSData)
}

// processLateData handles GPS data that arrived late (timestamp is older than recent data)
func (s *Server) processLateData(imei string, lat, lng float64, packet *protocol.DecodedPacket, recentData []models.GPSData) (bool, float64, float64) {
	colors.PrintData("⏰", "Processing late GPS data for device %s", imei)

	// Find the correct position in timeline for this late data
	var beforeData, afterData *models.GPSData

	for i := len(recentData) - 1; i >= 0; i-- {
		if packet.Timestamp.After(recentData[i].Timestamp) {
			if i > 0 {
				beforeData = &recentData[i]
				afterData = &recentData[i-1]
			} else {
				beforeData = &recentData[i]
			}
			break
		}
	}

	// If we found surrounding data points, validate against them
	if beforeData != nil && afterData != nil {
		// Calculate expected position based on interpolation
		expectedLat, expectedLng := s.interpolatePosition(beforeData, afterData, packet.Timestamp)

		// Check if late data is reasonable compared to expected position
		distance := s.calculateDistance(lat, lng, expectedLat, expectedLng)

		// Allow reasonable deviation (500m) for late data
		if distance > 500 {
			colors.PrintWarning("⏰ Late data rejected: too far from expected position (%.1fm)", distance)
			return false, lat, lng
		}

		// Validate speed consistency
		if !s.validateSpeedConsistency(lat, lng, packet, beforeData, afterData) {
			colors.PrintWarning("⏰ Late data rejected: speed inconsistency")
			return false, lat, lng
		}

		colors.PrintData("⏰", "Late data accepted: within %.1fm of expected position", distance)
		return true, lat, lng
	}

	// If only before data exists, validate against it
	if beforeData != nil {
		distance := s.calculateDistance(*beforeData.Latitude, *beforeData.Longitude, lat, lng)
		timeDiff := packet.Timestamp.Sub(beforeData.Timestamp).Seconds()

		if timeDiff > 0 {
			requiredSpeed := (distance / 1000) / (timeDiff / 3600)
			if requiredSpeed > 200 {
				colors.PrintWarning("⏰ Late data rejected: impossible speed %.1f km/h", requiredSpeed)
				return false, lat, lng
			}
		}
	}

	colors.PrintData("⏰", "Late data accepted with limited validation")
	return true, lat, lng
}

// processCurrentData handles current/future GPS data
func (s *Server) processCurrentData(lat, lng float64, packet *protocol.DecodedPacket, recentData []models.GPSData) (bool, float64, float64) {
	lastGPS := recentData[0]

	// Calculate distance and time difference
	distance := s.calculateDistance(*lastGPS.Latitude, *lastGPS.Longitude, lat, lng)
	timeDiff := packet.Timestamp.Sub(lastGPS.Timestamp).Seconds()

	// Skip validation if time difference is too small (less than 5 seconds)
	if timeDiff < 5 {
		if distance < 50 { // Accept only if within 50 meters for very recent points
			return true, lat, lng
		} else {
			colors.PrintWarning("🚫 GPS rejected: too far (%.1fm) for short time interval (%.1fs)", distance, timeDiff)
			return false, lat, lng
		}
	}

	// Calculate required speed in km/h to cover this distance
	requiredSpeed := (distance / 1000) / (timeDiff / 3600)

	// Get current speed from packet
	currentSpeed := 0
	if packet.Speed != nil {
		currentSpeed = int(*packet.Speed)
	}

	// Enhanced validation rules:

	// 1. Reject points that would require impossible speeds (>250 km/h)
	if requiredSpeed > 250 {
		colors.PrintWarning("🚫 GPS rejected: Required speed %.1f km/h too high (distance: %.1fm, time: %.1fs)",
			requiredSpeed, distance, timeDiff)
		return false, lat, lng
	}

	// 2. Reject points with unrealistic distance jumps for stopped/slow vehicles
	if currentSpeed < 5 && distance > 500 {
		colors.PrintWarning("🚫 GPS rejected: Large distance jump %.1fm while vehicle stopped/slow (speed: %d km/h)",
			distance, currentSpeed)
		return false, lat, lng
	}

	// 3. Check for GPS accuracy issues using multiple reference points
	if len(recentData) >= 3 {
		avgDistance := s.calculateAverageDistanceFromRecentPoints(lat, lng, recentData[:3])
		if avgDistance > 2000 && requiredSpeed > 150 {
			colors.PrintWarning("🚫 GPS rejected: Likely GPS accuracy issue (avg distance: %.1fm, speed: %.1f km/h)",
				avgDistance, requiredSpeed)
			return false, lat, lng
		}
	}

	// 4. Validate against movement pattern
	if len(recentData) >= 5 {
		if !s.validateMovementPattern(lat, lng, recentData[:5]) {
			colors.PrintWarning("🚫 GPS rejected: Inconsistent movement pattern")
			return false, lat, lng
		}
	}

	// 5. Speed consistency check
	if currentSpeed > 0 {
		speedRatio := requiredSpeed / float64(currentSpeed)
		if speedRatio > 4.0 {
			colors.PrintWarning("🚫 GPS rejected: Required speed %.1f km/h much higher than reported %d km/h (ratio: %.1f)",
				requiredSpeed, currentSpeed, speedRatio)
			return false, lat, lng
		}
	}

	colors.PrintData("✅", "GPS validation passed: distance=%.1fm, time=%.1fs, required_speed=%.1f km/h, reported_speed=%d km/h",
		distance, timeDiff, requiredSpeed, currentSpeed)

	return true, lat, lng
}

// interpolatePosition calculates expected position between two GPS points at a given time
func (s *Server) interpolatePosition(before, after *models.GPSData, targetTime time.Time) (float64, float64) {
	totalTime := after.Timestamp.Sub(before.Timestamp).Seconds()
	elapsedTime := targetTime.Sub(before.Timestamp).Seconds()

	if totalTime <= 0 {
		return *before.Latitude, *before.Longitude
	}

	ratio := elapsedTime / totalTime

	// Linear interpolation
	lat := *before.Latitude + (*after.Latitude-*before.Latitude)*ratio
	lng := *before.Longitude + (*after.Longitude-*before.Longitude)*ratio

	return lat, lng
}

// validateSpeedConsistency checks if the speed is consistent with surrounding data points
func (s *Server) validateSpeedConsistency(lat, lng float64, packet *protocol.DecodedPacket, before, after *models.GPSData) bool {
	// Calculate speed to previous point
	distToPrev := s.calculateDistance(*before.Latitude, *before.Longitude, lat, lng)
	timeToPrev := packet.Timestamp.Sub(before.Timestamp).Seconds()

	// Calculate speed to next point
	distToNext := s.calculateDistance(lat, lng, *after.Latitude, *after.Longitude)
	timeToNext := after.Timestamp.Sub(packet.Timestamp).Seconds()

	if timeToPrev > 0 && timeToNext > 0 {
		speedToPrev := (distToPrev / 1000) / (timeToPrev / 3600)
		speedToNext := (distToNext / 1000) / (timeToNext / 3600)

		// Check if speeds are reasonable (not exceeding 200 km/h)
		if speedToPrev > 200 || speedToNext > 200 {
			return false
		}

		// Check if speed change is reasonable
		speedDiff := math.Abs(speedToPrev - speedToNext)
		if speedDiff > 100 { // Allow max 100 km/h speed change
			return false
		}
	}

	return true
}

// calculateAverageDistanceFromRecentPoints calculates average distance from recent GPS points
func (s *Server) calculateAverageDistanceFromRecentPoints(lat, lng float64, recentPoints []models.GPSData) float64 {
	if len(recentPoints) == 0 {
		return 0
	}

	totalDistance := 0.0
	for _, point := range recentPoints {
		distance := s.calculateDistance(*point.Latitude, *point.Longitude, lat, lng)
		totalDistance += distance
	}

	return totalDistance / float64(len(recentPoints))
}

// validateMovementPattern checks if the GPS point fits the recent movement pattern
func (s *Server) validateMovementPattern(lat, lng float64, recentPoints []models.GPSData) bool {
	if len(recentPoints) < 3 {
		return true // Not enough data for pattern analysis
	}

	// Calculate recent movement vector
	recent1 := recentPoints[0]
	recent2 := recentPoints[1]
	recent3 := recentPoints[2]

	// Calculate direction vectors
	dir1Lat := *recent1.Latitude - *recent2.Latitude
	dir1Lng := *recent1.Longitude - *recent2.Longitude

	dir2Lat := *recent2.Latitude - *recent3.Latitude
	dir2Lng := *recent2.Longitude - *recent3.Longitude

	// Calculate direction to new point
	newDirLat := lat - *recent1.Latitude
	newDirLng := lng - *recent1.Longitude

	// Calculate angles (simplified)
	recentAngle := math.Atan2(dir1Lat, dir1Lng)
	prevAngle := math.Atan2(dir2Lat, dir2Lng)
	newAngle := math.Atan2(newDirLat, newDirLng)

	// Check for sudden direction changes (>120 degrees)
	angleDiff1 := math.Abs(recentAngle - newAngle)
	angleDiff2 := math.Abs(prevAngle - newAngle)

	if angleDiff1 > 2.0 && angleDiff2 > 2.0 { // ~120 degrees in radians
		// Check if this might be a valid turn by looking at recent speed data
		if recent1.Speed != nil && *recent1.Speed > 30 {
			return false // High speed sharp turn is suspicious
		}
	}

	return true
}

// calculateDistance calculates the distance between two GPS points in meters
func (s *Server) calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371000 // Earth radius in meters

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180

	// Haversine formula
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// saveStatusOnlyData saves status information without GPS coordinates
func (s *Server) saveStatusOnlyData(packet *protocol.DecodedPacket, deviceIMEI string) {
	// Create status data without coordinates
	statusData := s.buildStatusData(packet, deviceIMEI)

	// Clear coordinates since ignition is off
	statusData.Latitude = nil
	statusData.Longitude = nil
	statusData.Speed = nil
	statusData.Course = nil

	if err := db.GetDB().Create(&statusData).Error; err != nil {
		colors.PrintError("Error saving status-only data: %v", err)
	} else {
		colors.PrintSuccess("Status-only data saved for device %s (ignition off)", deviceIMEI)

		// Broadcast status update without coordinates
		if http.WSHub != nil {
			go http.WSHub.BroadcastStatusUpdate(&statusData, "", "")
		}
	}
}

// handleStatusPacket processes device status packets
func (s *Server) handleStatusPacket(packet *protocol.DecodedPacket, conn net.Conn, deviceIMEI string) {
	// Update device activity
	s.updateDeviceActivity(deviceIMEI, conn)

	colors.PrintData("📊", "Status info from %s: Ignition=%s, Voltage=%v, GSM Signal=%v",
		conn.RemoteAddr(), packet.Ignition, packet.Voltage, packet.GSMSignal)

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

		if err := db.GetDB().Create(&statusData).Error; err != nil {
			colors.PrintError("Error saving status data: %v", err)
		} else {
			colors.PrintSuccess("Status data saved for device %s", deviceIMEI)

			// Broadcast status update as a full GPS update to WebSocket clients
			if http.WSHub != nil {
				go http.WSHub.BroadcastFullGPSUpdate(&statusData)
			}
		}
	}
}

// handleAlarmPacket processes alarm packets
func (s *Server) handleAlarmPacket(packet *protocol.DecodedPacket, conn net.Conn) {
	colors.PrintWarning("Alarm from %s: Type=%+v", conn.RemoteAddr(), packet.AlarmType)
}

// sendResponse sends response back to device
func (s *Server) sendResponse(packet *protocol.DecodedPacket, conn net.Conn, decoder *protocol.GT06Decoder) {
	response := decoder.GenerateResponse(uint16(packet.SerialNumber), packet.Protocol)

	_, err := conn.Write(response)
	if err != nil {
		colors.PrintError("Error sending response to %s: %v", conn.RemoteAddr(), err)
	} else {
		colors.PrintData("📤", "Sent response to %s: %X", conn.RemoteAddr(), response)
	}
}

// buildGPSData creates a GPSData model from decoded packet
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
		IMEI:           deviceIMEI,
		Timestamp:      packet.Timestamp,
		ProtocolName:   packet.ProtocolName,
		RawPacket:      packet.Raw,
		Ignition:       packet.Ignition,
		Charger:        packet.Charger,
		GPSTracking:    packet.GPSTracking,
		OilElectricity: packet.OilElectricity,
		DeviceStatus:   packet.DeviceStatus,
	}

	// Voltage information
	if packet.Voltage != nil {
		voltageLevel := int(packet.Voltage.Level)
		statusData.VoltageLevel = &voltageLevel
		statusData.VoltageStatus = packet.Voltage.Status
	}

	// GSM information
	if packet.GSMSignal != nil {
		gsmSignal := int(packet.GSMSignal.Level)
		statusData.GSMSignal = &gsmSignal
		statusData.GSMStatus = packet.GSMSignal.Status
	}

	// Alarm information
	if packet.Alarm != nil {
		statusData.AlarmActive = packet.Alarm.Active
		statusData.AlarmType = packet.Alarm.Type
		statusData.AlarmCode = packet.Alarm.Code
	}

	return statusData
}

// monitorDeviceTimeouts checks for devices that haven't sent data in over an hour
func (s *Server) monitorDeviceTimeouts() {
	for range s.timeoutTicker.C {
		s.connectionMutex.Lock()
		now := time.Now()

		// Check connected devices for timeout
		for imei, deviceConn := range s.deviceConnections {
			// Check if device hasn't sent data for more than 1 hour
			if now.Sub(deviceConn.LastActivity) > time.Hour && deviceConn.IsActive {
				colors.PrintWarning("📱 Device %s connection timed out (no data for %v)",
					imei, now.Sub(deviceConn.LastActivity))

				// Mark as inactive
				deviceConn.IsActive = false

				// Close connection
				if deviceConn.Conn != nil {
					deviceConn.Conn.Close()
				}

				// Unregister from control controller
				s.controlController.UnregisterConnection(imei)

				// ENHANCED FIX: Don't broadcast status on timeout - let the monitoring system handle it
				// The checkDevicesForInactiveStatus will properly determine status based on GPS data age
				colors.PrintInfo("📱 Device %s connection timed out, monitoring system will determine proper status", imei)
			}
		}
		s.connectionMutex.Unlock()

		// Check all devices in database for inactive status based on GPS data
		s.checkDevicesForInactiveStatus()
	}
}

// checkDevicesForInactiveStatus checks all devices and marks them stopped/inactive based on data age
func (s *Server) checkDevicesForInactiveStatus() {
	var devices []models.Device
	if err := db.GetDB().Find(&devices).Error; err != nil {
		colors.PrintError("Error fetching devices for inactive check: %v", err)
		return
	}

	now := time.Now()
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

// broadcastNoDataStatus broadcasts no-data status for a device (literally no GPS data in database)
func (s *Server) broadcastNoDataStatus(imei string) {
	// Get vehicle info for WebSocket broadcast
	var vehicle models.Vehicle
	vehicleReg := ""
	vehicleName := ""
	if err := db.GetDB().Where("imei = ?", imei).First(&vehicle).Error; err == nil {
		vehicleReg = vehicle.RegNo
		vehicleName = vehicle.Name
	}

	// Create a GPS update with no-data status
	if http.WSHub != nil {
		// Use BroadcastGPSUpdate to send no-data status properly
		gpsData := &models.GPSData{
			IMEI:      imei,
			Timestamp: time.Now(),
			// No coordinates - will show as no-data
			Latitude:     nil,
			Longitude:    nil,
			Speed:        nil,
			Course:       nil,
			Ignition:     "OFF",
			ProtocolName: "NO_DATA",
		}
		http.WSHub.BroadcastGPSUpdate(gpsData, vehicleName, vehicleReg)
	}
}

// broadcastInactiveStatusWithGPS broadcasts inactive status but with GPS data for positioning
func (s *Server) broadcastInactiveStatusWithGPS(imei string, gpsData *models.GPSData) {
	// Get vehicle info for WebSocket broadcast
	var vehicle models.Vehicle
	vehicleReg := ""
	vehicleName := ""
	if err := db.GetDB().Where("imei = ?", imei).First(&vehicle).Error; err == nil {
		vehicleReg = vehicle.RegNo
		vehicleName = vehicle.Name
	}

	// Broadcast GPS data - the frontend will calculate status as "inactive" due to old timestamp
	if http.WSHub != nil {
		http.WSHub.BroadcastGPSUpdate(gpsData, vehicleName, vehicleReg)
	}
}

// broadcastVehicleStatusFromGPS broadcasts current vehicle status based on GPS data
func (s *Server) broadcastVehicleStatusFromGPS(imei string, gpsData *models.GPSData) {
	// Get vehicle info for WebSocket broadcast
	var vehicle models.Vehicle
	vehicleReg := ""
	vehicleName := ""
	if err := db.GetDB().Where("imei = ?", imei).First(&vehicle).Error; err == nil {
		vehicleReg = vehicle.RegNo
		vehicleName = vehicle.Name
	}

	// Broadcast GPS data - frontend will calculate appropriate status (running/idle/stopped)
	if http.WSHub != nil {
		http.WSHub.BroadcastGPSUpdate(gpsData, vehicleName, vehicleReg)
	}
}

// updateDeviceActivity updates the last activity time for a device
func (s *Server) updateDeviceActivity(imei string, conn net.Conn) {
	s.connectionMutex.Lock()
	defer s.connectionMutex.Unlock()

	if deviceConn, exists := s.deviceConnections[imei]; exists {
		deviceConn.LastActivity = time.Now()
		deviceConn.IsActive = true
	} else {
		s.deviceConnections[imei] = &DeviceConnection{
			Conn:         conn,
			LastActivity: time.Now(),
			IMEI:         imei,
			IsActive:     true,
		}
	}
}

// removeDeviceConnection removes a device connection
func (s *Server) removeDeviceConnection(imei string) {
	s.connectionMutex.Lock()
	defer s.connectionMutex.Unlock()
	delete(s.deviceConnections, imei)
}
