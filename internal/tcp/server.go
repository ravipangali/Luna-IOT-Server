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

// handleGPSPacket processes GPS data packets
func (s *Server) handleGPSPacket(packet *protocol.DecodedPacket, conn net.Conn, deviceIMEI string) {
	// Update device activity
	s.updateDeviceActivity(deviceIMEI, conn)

	// Validate and process GPS coordinates before saving
	var hasValidGPS bool
	if packet.Latitude != nil && packet.Longitude != nil {
		lat := *packet.Latitude
		lng := *packet.Longitude

		// Convert negative latitude to positive (remove minus sign)
		if lat < 0 {
			lat = -lat
			packet.Latitude = &lat
		}

		// Accept both negative and positive longitude values
		// Latitude: 0-90 (positive only), Longitude: -180 to +180 (both negative/positive)
		if lat > 0 && lat <= 90 && lng >= -180 && lng <= 180 {
			hasValidGPS = true
			colors.PrintData("📍", "Valid GPS Location: Lat=%.12f, Lng=%.12f, Speed=%v km/h",
				lat, lng, packet.Speed)
		} else {
			colors.PrintWarning("📍 Invalid GPS coordinates (out of range): Lat=%.12f, Lng=%.12f", lat, lng)
		}
	}

	// Save GPS data and broadcast to WebSocket clients
	if deviceIMEI != "" && s.isDeviceRegistered(deviceIMEI) {
		gpsData := s.buildGPSData(packet, deviceIMEI)

		// Only save if we have valid GPS coordinates or this is a status update
		if hasValidGPS || packet.ProtocolName == "STATUS_INFO" {
			if err := db.GetDB().Create(&gpsData).Error; err != nil {
				colors.PrintError("Error saving GPS data: %v", err)
			} else {
				colors.PrintSuccess("GPS data saved for device %s", deviceIMEI)

				// Broadcast the new full GPS data object over WebSocket
				if http.WSHub != nil {
					go http.WSHub.BroadcastFullGPSUpdate(&gpsData)
				}
			}
		} else {
			colors.PrintWarning("Skipping GPS data save due to invalid coordinates")
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
