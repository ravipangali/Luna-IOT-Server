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

			// Get vehicle info for WebSocket broadcast
			var vehicle models.Vehicle
			vehicleReg := ""
			if err := db.GetDB().Where("imei = ?", deviceIMEI).First(&vehicle).Error; err == nil {
				vehicleReg = vehicle.RegNo
			}

			// Broadcast device disconnection to WebSocket clients
			if http.WSHub != nil {
				http.WSHub.BroadcastDeviceStatus(deviceIMEI, "disconnected", vehicleReg)
			}
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
				case "GPS_LBS_STATUS", "GPS_LBS_DATA", "GPS_LBS_STATUS_A0":
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

	// Validate GPS coordinates before saving
	var hasValidGPS bool
	if packet.Latitude != nil && packet.Longitude != nil {
		lat := *packet.Latitude
		lng := *packet.Longitude

		// Basic GPS coordinate validation
		if lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180 &&
			lat != 0 && lng != 0 { // Exclude null island (0,0)
			hasValidGPS = true
			colors.PrintData("📍", "Valid GPS Location: Lat=%.6f, Lng=%.6f, Speed=%v km/h",
				lat, lng, packet.Speed)
		} else {
			colors.PrintWarning("📍 Invalid GPS coordinates: Lat=%.6f, Lng=%.6f", lat, lng)
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

				// Get vehicle information for WebSocket broadcast
				var vehicle models.Vehicle
				vehicleName := ""
				regNo := ""
				if err := db.GetDB().Where("imei = ?", deviceIMEI).First(&vehicle).Error; err == nil {
					vehicleName = vehicle.Name
					regNo = vehicle.RegNo
				}

				// Broadcast GPS update to WebSocket clients
				if http.WSHub != nil {
					http.WSHub.BroadcastGPSUpdate(&gpsData, vehicleName, regNo)
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

			// Get vehicle information for WebSocket broadcast
			var vehicle models.Vehicle
			vehicleName := ""
			regNo := ""
			if err := db.GetDB().Where("imei = ?", deviceIMEI).First(&vehicle).Error; err == nil {
				vehicleName = vehicle.Name
				regNo = vehicle.RegNo
			}

			// Broadcast status update as GPS update to WebSocket clients
			if http.WSHub != nil {
				http.WSHub.BroadcastGPSUpdate(&statusData, vehicleName, regNo)
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

	// GPS location data
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

	// LBS data
	if packet.MCC != nil {
		mcc := int(*packet.MCC)
		gpsData.MCC = &mcc
	}
	if packet.MNC != nil {
		mnc := int(*packet.MNC)
		gpsData.MNC = &mnc
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

		for imei, deviceConn := range s.deviceConnections {
			// Check if device hasn't sent data for more than 1 hour
			if now.Sub(deviceConn.LastActivity) > time.Hour && deviceConn.IsActive {
				colors.PrintWarning("📱 Device %s timed out (no data for %v)",
					imei, now.Sub(deviceConn.LastActivity))

				// Mark as inactive
				deviceConn.IsActive = false

				// Close connection
				if deviceConn.Conn != nil {
					deviceConn.Conn.Close()
				}

				// Unregister from control controller
				s.controlController.UnregisterConnection(imei)

				// Get vehicle info for WebSocket broadcast
				var vehicle models.Vehicle
				vehicleReg := ""
				if err := db.GetDB().Where("imei = ?", imei).First(&vehicle).Error; err == nil {
					vehicleReg = vehicle.RegNo
				}

				// Broadcast device disconnection to WebSocket clients
				if http.WSHub != nil {
					http.WSHub.BroadcastDeviceStatus(imei, "disconnected", vehicleReg)
				}
			}
		}
		s.connectionMutex.Unlock()
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
