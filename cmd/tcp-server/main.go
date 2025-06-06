package main

import (
	"encoding/json"
	"fmt"
	"log"
	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/internal/protocol"
	"net"
	"os"

	"github.com/joho/godotenv"
)

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
	defer conn.Close()

	fmt.Printf("Client connected: %s\n", conn.RemoteAddr())

	// Create a new GT06 decoder instance for this connection
	decoder := protocol.NewGT06Decoder()

	// Track device IMEI for this connection
	var deviceIMEI string

	buffer := make([]byte, 1024)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Printf("Client disconnected: %s\n", conn.RemoteAddr())
				break
			}
			log.Printf("Error reading from connection %s: %v", conn.RemoteAddr(), err)
			break
		}

		if n > 0 {
			// Log raw data received
			fmt.Printf("Raw data from %s: %X\n", conn.RemoteAddr(), buffer[:n])

			// Process data through GT06 decoder
			packets, err := decoder.AddData(buffer[:n])
			if err != nil {
				log.Printf("Error decoding data from %s: %v", conn.RemoteAddr(), err)
				continue
			}

			// Process each decoded packet
			for _, packet := range packets {
				fmt.Printf("Decoded packet from %s:\n", conn.RemoteAddr())

				// Convert packet to JSON for pretty printing
				jsonData, err := json.MarshalIndent(packet, "", "  ")
				if err != nil {
					log.Printf("Error marshaling packet to JSON: %v", err)
					fmt.Printf("Packet: %+v\n", packet)
				} else {
					fmt.Printf("%s\n", jsonData)
				}

				// Handle different packet types
				switch packet.ProtocolName {
				case "LOGIN":
					fmt.Printf("Device login from %s - Terminal ID: %s\n",
						conn.RemoteAddr(), packet.TerminalID)

					// Convert hex terminal ID to IMEI and store for this connection
					if len(packet.TerminalID) >= 15 {
						deviceIMEI = packet.TerminalID[:15]
					}

				case "GPS_LBS_STATUS", "GPS_LBS_DATA", "GPS_LBS_STATUS_A0":
					if packet.Latitude != nil && packet.Longitude != nil {
						fmt.Printf("GPS Location from %s: Lat=%.6f, Lng=%.6f, Speed=%v\n",
							conn.RemoteAddr(), *packet.Latitude, *packet.Longitude, packet.Speed)
					}

					// Save GPS data to database if we have device IMEI
					if deviceIMEI != "" {
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
							log.Printf("Error saving GPS data: %v", err)
						} else {
							fmt.Printf("GPS data saved for device %s\n", deviceIMEI)
						}
					}

				case "STATUS_INFO":
					fmt.Printf("Status info from %s: Ignition=%s, Voltage=%v, GSM Signal=%v\n",
						conn.RemoteAddr(), packet.Ignition, packet.Voltage, packet.GSMSignal)

					// Save status data to database if we have device IMEI
					if deviceIMEI != "" {
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
							log.Printf("Error saving status data: %v", err)
						} else {
							fmt.Printf("Status data saved for device %s\n", deviceIMEI)
						}
					}

				case "ALARM_DATA":
					fmt.Printf("Alarm from %s: Type=%+v\n",
						conn.RemoteAddr(), packet.AlarmType)
				}

				// Send response if required
				if packet.NeedsResponse {
					response := decoder.GenerateResponse(uint16(packet.SerialNumber), packet.Protocol)

					_, err := conn.Write(response)
					if err != nil {
						log.Printf("Error sending response to %s: %v", conn.RemoteAddr(), err)
					} else {
						fmt.Printf("Sent response to %s: %X\n", conn.RemoteAddr(), response)
					}
				}
			}
		}
	}
}

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize database connection
	if err := db.Initialize(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Get TCP port from environment variable or use default
	port := os.Getenv("TCP_PORT")
	if port == "" {
		port = "5000"
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}

	defer listener.Close()

	fmt.Printf("GT06 TCP Server is running on port %s\n", port)
	fmt.Println("Waiting for GT06 device connections...")
	fmt.Println("Database connectivity enabled - GPS data will be saved")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		// Handle each connection in a separate goroutine
		go handleConnection(conn)
	}
}
