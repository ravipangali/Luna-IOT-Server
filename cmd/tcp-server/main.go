package main

import (
	"encoding/json"
	"fmt"
	"log"
	"luna_iot_server/internal/db"
	"luna_iot_server/internal/http/controllers"
	"luna_iot_server/internal/models"
	"luna_iot_server/internal/protocol"
	"luna_iot_server/pkg/colors"
	"net"
	"os"

	"github.com/joho/godotenv"
)

// Global control controller instance to track active connections
var controlController *controllers.ControlController

// isDeviceRegistered checks if a device with given IMEI exists in the database
func isDeviceRegistered(imei string) bool {
	var device models.Device
	err := db.GetDB().Where("imei = ?", imei).First(&device).Error
	return err == nil
}

// saveGPSData saves GPS packet data to database
func saveGPSData(packet *protocol.DecodedPacket) error {
	if packet == nil {
		return nil
	}

	// Extract IMEI from TerminalID for LOGIN packets, or use stored IMEI
	var imei string
	if packet.ProtocolName == "LOGIN" && packet.TerminalID != "" {
		// TerminalID is hex encoded, convert to decimal IMEI
		imei = packet.TerminalID

		// Validate device exists in database before processing
		if !isDeviceRegistered(imei) {
			return fmt.Errorf("IMEI %s is not registered on our system", imei)
		}
	} else {
		// For other packets, we need to track the IMEI from login
		// This is a simplified approach - in production, you'd track by connection
		return nil
	}

	// Only save GPS data packets
	if packet.ProtocolName != "GPS_LBS_STATUS" &&
		packet.ProtocolName != "GPS_LBS_DATA" &&
		packet.ProtocolName != "GPS_LBS_STATUS_A0" &&
		packet.ProtocolName != "STATUS_INFO" {
		return nil
	}

	gpsData := models.GPSData{
		IMEI:         imei,
		Timestamp:    packet.Timestamp,
		ProtocolName: packet.ProtocolName,
		RawPacket:    packet.Raw,
	}

	// GPS Location Data
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

	// GPS Status
	if packet.GPSRealTime != nil {
		gpsData.GPSRealTime = packet.GPSRealTime
	}
	if packet.GPSPositioned != nil {
		gpsData.GPSPositioned = packet.GPSPositioned
	}

	// Device Status
	gpsData.Ignition = packet.Ignition
	gpsData.Charger = packet.Charger
	gpsData.GPSTracking = packet.GPSTracking
	gpsData.OilElectricity = packet.OilElectricity
	gpsData.DeviceStatus = packet.DeviceStatus

	// Signal & Power
	if packet.Voltage != nil {
		voltageLevel := int(packet.Voltage.Level)
		gpsData.VoltageLevel = &voltageLevel
		gpsData.VoltageStatus = packet.Voltage.Status
	}
	if packet.GSMSignal != nil {
		gsmSignal := int(packet.GSMSignal.Level)
		gpsData.GSMSignal = &gsmSignal
		gpsData.GSMStatus = packet.GSMSignal.Status
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
	if packet.LAC != nil {
		lac := int(*packet.LAC)
		gpsData.LAC = &lac
	}
	if packet.CellID != nil {
		cellID := int(*packet.CellID)
		gpsData.CellID = &cellID
	}

	// Alarm Data
	if packet.Alarm != nil {
		gpsData.AlarmActive = packet.Alarm.Active
		gpsData.AlarmType = packet.Alarm.Type
		gpsData.AlarmCode = packet.Alarm.Code
	}

	// Save to database
	if err := db.GetDB().Create(&gpsData).Error; err != nil {
		return fmt.Errorf("failed to save GPS data: %v", err)
	}

	return nil
}

func handleConnection(conn net.Conn) {
	// Track device IMEI for this connection - declared first so it's in scope for defer
	var deviceIMEI string

	defer func() {
		conn.Close()
		// Unregister connection when it closes
		if deviceIMEI != "" {
			controlController.UnregisterConnection(deviceIMEI)
		}
	}()

	colors.PrintConnection("📱", "Client connected: %s", conn.RemoteAddr())

	// Create a new GT06 decoder instance for this connection
	decoder := protocol.NewGT06Decoder()

	buffer := make([]byte, 1024)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err.Error() == "EOF" {
				colors.PrintConnection("📱", "Client disconnected: %s", conn.RemoteAddr())
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
					colors.PrintConnection("🔐", "Device login from %s - Terminal ID: %s", conn.RemoteAddr(), packet.TerminalID)

					// Convert hex terminal ID to IMEI and validate device exists
					if len(packet.TerminalID) >= 15 {
						potentialIMEI := packet.TerminalID[:15]

						// Check if device exists in database
						if !isDeviceRegistered(potentialIMEI) {
							colors.PrintError("Unauthorized device attempted login from %s - IMEI: %s (not registered)", conn.RemoteAddr(), potentialIMEI)
							colors.PrintWarning("Rejecting unregistered device: %s", potentialIMEI)
							// Close connection for unregistered devices
							conn.Close()
							return
						}

						// Device is registered, allow connection
						deviceIMEI = potentialIMEI
						colors.PrintSuccess("Authorized device login: %s", deviceIMEI)

						// Register connection for control operations
						controlController.RegisterConnection(deviceIMEI, conn)
					}

				case "GPS_LBS_STATUS", "GPS_LBS_DATA", "GPS_LBS_STATUS_A0":
					if packet.Latitude != nil && packet.Longitude != nil {
						colors.PrintData("📍", "GPS Location from %s: Lat=%.6f, Lng=%.6f, Speed=%v km/h", conn.RemoteAddr(), *packet.Latitude, *packet.Longitude, packet.Speed)
					}

					// Save GPS data to database if we have device IMEI and device is still registered
					if deviceIMEI != "" {
						// Verify device still exists before saving GPS data
						if !isDeviceRegistered(deviceIMEI) {
							colors.PrintWarning("IMEI %s is not registered on our system", deviceIMEI)
							continue
						}
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

						// Save to database
						if err := db.GetDB().Create(&gpsData).Error; err != nil {
							colors.PrintError("Error saving GPS data: %v", err)
						} else {
							colors.PrintSuccess("GPS data saved for device %s", deviceIMEI)
						}
					}

				case "STATUS_INFO":
					colors.PrintData("📊", "Status info from %s: Ignition=%s, Voltage=%v, GSM Signal=%v", conn.RemoteAddr(), packet.Ignition, packet.Voltage, packet.GSMSignal)

					// Save status data to database if we have device IMEI and device is still registered
					if deviceIMEI != "" {
						// Verify device still exists before saving status data
						if !isDeviceRegistered(deviceIMEI) {
							colors.PrintWarning("IMEI %s is not registered on our system", deviceIMEI)
							continue
						}
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

						// Save to database
						if err := db.GetDB().Create(&gpsData).Error; err != nil {
							colors.PrintError("Error saving status data: %v", err)
						} else {
							colors.PrintSuccess("Status data saved for device %s", deviceIMEI)
						}
					}

				case "ALARM_DATA":
					colors.PrintWarning("Alarm from %s: Type=%+v", conn.RemoteAddr(), packet.AlarmType)
				}

				// Send response if required
				if packet.NeedsResponse {
					imei := packet.TerminalID
					colors.PrintDebug("IMEI: %s", imei)

					response := decoder.GenerateResponse(uint16(packet.SerialNumber), packet.Protocol)

					_, err := conn.Write(response)
					if err != nil {
						colors.PrintError("Error sending response to %s: %v", conn.RemoteAddr(), err)
					} else {
						colors.PrintData("📤", "Sent response to %s: %X", conn.RemoteAddr(), response)
					}
				}
			}
		}
	}
}

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		colors.PrintWarning("No .env file found, using system environment variables")
	}

	// Initialize database connection
	if err := db.Initialize(); err != nil {
		colors.PrintError("Failed to initialize database: %v", err)
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize global control controller
	controlController = controllers.NewControlController()

	// Get TCP port from environment variable or use default
	port := os.Getenv("TCP_PORT")
	if port == "" {
		port = "5000"
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		colors.PrintError("Failed to start server: %v", err)
		log.Fatal("Failed to start server:", err)
	}

	defer listener.Close()

	colors.PrintServer("📡", "GT06 TCP Server is running on port %s", port)
	colors.PrintConnection("📶", "Waiting for GT06 device connections...")
	colors.PrintData("💾", "Database connectivity enabled - GPS data will be saved")
	colors.PrintControl("Oil/Electricity control system enabled - Ready for commands")

	for {
		conn, err := listener.Accept()
		if err != nil {
			colors.PrintError("Error accepting connection: %v", err)
			continue
		}

		// Handle each connection in a separate goroutine
		go handleConnection(conn)
	}
}
