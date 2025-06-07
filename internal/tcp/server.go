package tcp

import (
	"encoding/json"
	"fmt"
	"luna_iot_server/internal/db"
	"luna_iot_server/internal/http/controllers"
	"luna_iot_server/internal/models"
	"luna_iot_server/internal/protocol"
	"luna_iot_server/pkg/colors"
	"net"
)

// Server represents the TCP server for IoT devices
type Server struct {
	port              string
	listener          net.Listener
	controlController *controllers.ControlController
}

// NewServer creates a new TCP server instance
func NewServer(port string) *Server {
	return &Server{
		port:              port,
		controlController: controllers.NewControlController(),
	}
}

// NewServerWithController creates a new TCP server instance with a shared control controller
func NewServerWithController(port string, sharedController *controllers.ControlController) *Server {
	return &Server{
		port:              port,
		controlController: sharedController,
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
				colors.PrintData("📋", "Decoded packet from %s:", conn.RemoteAddr())

				// Convert packet to JSON for pretty printing
				jsonData, err := json.MarshalIndent(packet, "", "  ")
				if err != nil {
					colors.PrintError("Error marshaling packet to JSON: %v", err)
					colors.PrintDebug("Packet: %+v", packet)
				} else {
					colors.PrintDebug("Packet Data:\n%s", jsonData)
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

// handleLoginPacket processes LOGIN packets from IoT devices
func (s *Server) handleLoginPacket(packet *protocol.DecodedPacket, conn net.Conn) string {
	colors.PrintConnection("🔐", "Device login from %s - Terminal ID: %s", conn.RemoteAddr(), packet.TerminalID)

	// Convert hex terminal ID to IMEI and validate device exists
	if len(packet.TerminalID) >= 16 {
		potentialIMEI := packet.TerminalID[:16]

		// Check if device exists in database
		if !s.isDeviceRegistered(potentialIMEI) {
			colors.PrintError("Unauthorized device attempted login from %s - IMEI: %s (not registered)", conn.RemoteAddr(), potentialIMEI)
			colors.PrintWarning("Rejecting unregistered device: %s", potentialIMEI)
			// Close connection for unregistered devices
			conn.Close()
			return ""
		}

		// Device is registered, allow connection
		colors.PrintSuccess("Authorized device login: %s", potentialIMEI)

		// Register connection for control operations
		s.controlController.RegisterConnection(potentialIMEI, conn)
		return potentialIMEI
	}
	return ""
}

// handleGPSPacket processes GPS data packets
func (s *Server) handleGPSPacket(packet *protocol.DecodedPacket, conn net.Conn, deviceIMEI string) {
	if packet.Latitude != nil && packet.Longitude != nil {
		colors.PrintData("📍", "GPS Location from %s: Lat=%.6f, Lng=%.6f, Speed=%v km/h",
			conn.RemoteAddr(), *packet.Latitude, *packet.Longitude, packet.Speed)
	}

	// Save GPS data to database if we have device IMEI and device is still registered
	if deviceIMEI != "" {
		// Verify device still exists before saving GPS data
		if !s.isDeviceRegistered(deviceIMEI) {
			colors.PrintWarning("IMEI %s is not registered on our system", deviceIMEI)
			return
		}

		gpsData := s.buildGPSData(packet, deviceIMEI)

		// Save to database
		if err := db.GetDB().Create(&gpsData).Error; err != nil {
			colors.PrintError("Error saving GPS data: %v", err)
		} else {
			colors.PrintSuccess("GPS data saved for device %s", deviceIMEI)
		}
	}
}

// handleStatusPacket processes STATUS_INFO packets
func (s *Server) handleStatusPacket(packet *protocol.DecodedPacket, conn net.Conn, deviceIMEI string) {
	colors.PrintData("📊", "Status info from %s: Ignition=%s, Voltage=%v, GSM Signal=%v",
		conn.RemoteAddr(), packet.Ignition, packet.Voltage, packet.GSMSignal)

	// Save status data to database if we have device IMEI and device is still registered
	if deviceIMEI != "" {
		// Verify device still exists before saving status data
		if !s.isDeviceRegistered(deviceIMEI) {
			colors.PrintWarning("IMEI %s is not registered on our system", deviceIMEI)
			return
		}

		statusData := s.buildStatusData(packet, deviceIMEI)

		// Save to database
		if err := db.GetDB().Create(&statusData).Error; err != nil {
			colors.PrintError("Error saving status data: %v", err)
		} else {
			colors.PrintSuccess("Status data saved for device %s", deviceIMEI)
		}
	}
}

// handleAlarmPacket processes ALARM_DATA packets
func (s *Server) handleAlarmPacket(packet *protocol.DecodedPacket, conn net.Conn) {
	colors.PrintWarning("🚨 ALARM detected from %s: Type=%+v", conn.RemoteAddr(), packet.AlarmType)
}

// sendResponse sends response packets back to IoT devices
func (s *Server) sendResponse(packet *protocol.DecodedPacket, conn net.Conn, decoder *protocol.GT06Decoder) {
	imei := packet.TerminalID
	colors.PrintData("📤", "Preparing response for IMEI: %s", imei)

	response := decoder.GenerateResponse(uint16(packet.SerialNumber), packet.Protocol)

	_, err := conn.Write(response)
	if err != nil {
		colors.PrintError("Error sending response to %s: %v", conn.RemoteAddr(), err)
	} else {
		colors.PrintSuccess("Response sent to %s: %X", conn.RemoteAddr(), response)
	}
}

// buildGPSData constructs GPS data model from packet
func (s *Server) buildGPSData(packet *protocol.DecodedPacket, deviceIMEI string) models.GPSData {
	gpsData := models.GPSData{
		IMEI:         deviceIMEI,
		Timestamp:    packet.Timestamp,
		ProtocolName: packet.ProtocolName,
		RawPacket:    packet.Raw,
	}

	// Copy GPS data
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
	if packet.GPSRealTime != nil {
		gpsData.GPSRealTime = packet.GPSRealTime
	}
	if packet.GPSPositioned != nil {
		gpsData.GPSPositioned = packet.GPSPositioned
	}

	// LBS Data
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

// buildStatusData constructs status data model from packet
func (s *Server) buildStatusData(packet *protocol.DecodedPacket, deviceIMEI string) models.GPSData {
	gpsData := models.GPSData{
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

	// Voltage info
	if packet.Voltage != nil {
		voltageLevel := int(packet.Voltage.Level)
		gpsData.VoltageLevel = &voltageLevel
		gpsData.VoltageStatus = packet.Voltage.Status
	}

	// GSM info
	if packet.GSMSignal != nil {
		gsmSignal := int(packet.GSMSignal.Level)
		gpsData.GSMSignal = &gsmSignal
		gpsData.GSMStatus = packet.GSMSignal.Status
	}

	// Alarm info
	if packet.Alarm != nil {
		gpsData.AlarmActive = packet.Alarm.Active
		gpsData.AlarmType = packet.Alarm.Type
		gpsData.AlarmCode = packet.Alarm.Code
	}

	return gpsData
}
