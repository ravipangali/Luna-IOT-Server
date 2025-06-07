package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

// Protocol constants for GPS tracker control
const (
	ProtocolServer   = 0x80 // Server to terminal
	ProtocolTerminal = 0x15 // Terminal response
	StartBit         = 0x7878
	StopBit          = 0x0D0A
	LanguageChinese  = 0x0001
	LanguageEnglish  = 0x0002
)

// Control Commands
const (
	CmdCutOil     = "DYD#"  // Cut oil and electricity
	CmdConnectOil = "HFYD#" // Connect oil and electricity
	CmdLocation   = "DWXX#" // Get location info
)

// ControlPacket represents the GPS tracker communication packet for control commands
type ControlPacket struct {
	StartBit         uint16
	PacketLength     uint8
	ProtocolNumber   uint8
	CommandLength    uint8
	ServerFlag       [4]byte
	CommandContent   string
	Language         uint16
	InfoSerialNumber uint16
	ErrorCheck       uint16
	StopBit          uint16
}

// GPSTrackerController handles oil and electricity control communication with GPS tracking device
type GPSTrackerController struct {
	conn         net.Conn
	serverFlag   [4]byte
	serialNumber uint16
	language     uint16
	deviceIMEI   string
}

// ControlResponse represents the response from a control command
type ControlResponse struct {
	Command    string    `json:"command"`
	Response   string    `json:"response"`
	Success    bool      `json:"success"`
	Message    string    `json:"message"`
	Timestamp  time.Time `json:"timestamp"`
	DeviceIMEI string    `json:"device_imei"`
}

// NewGPSTrackerController creates a new GPS tracker controller instance
func NewGPSTrackerController(conn net.Conn, deviceIMEI string) *GPSTrackerController {
	return &GPSTrackerController{
		conn:         conn,
		serverFlag:   [4]byte{0x01, 0x02, 0x03, 0x04}, // Server identification flag
		serialNumber: 1,
		language:     LanguageEnglish,
		deviceIMEI:   deviceIMEI,
	}
}

// buildControlPacket creates a packet for sending control commands to terminal
func (g *GPSTrackerController) buildControlPacket(command string) *ControlPacket {
	commandBytes := []byte(command)
	commandLength := uint8(4 + len(commandBytes)) // ServerFlag(4) + Command content

	packet := &ControlPacket{
		StartBit:         StartBit,
		PacketLength:     uint8(1 + 1 + commandLength + 2 + 2 + 2), // Protocol + Length + Command + Language + Serial + ErrorCheck
		ProtocolNumber:   ProtocolServer,
		CommandLength:    commandLength,
		ServerFlag:       g.serverFlag,
		CommandContent:   command,
		Language:         g.language,
		InfoSerialNumber: g.serialNumber,
		ErrorCheck:       0, // Will be calculated
		StopBit:          StopBit,
	}

	// Calculate error check (simple checksum)
	packet.ErrorCheck = g.calculateChecksum(packet)
	g.serialNumber++

	return packet
}

// calculateChecksum calculates a simple checksum for the packet
func (g *GPSTrackerController) calculateChecksum(packet *ControlPacket) uint16 {
	var sum uint16
	sum += uint16(packet.PacketLength)
	sum += uint16(packet.ProtocolNumber)
	sum += uint16(packet.CommandLength)

	for _, b := range packet.ServerFlag {
		sum += uint16(b)
	}

	for _, b := range []byte(packet.CommandContent) {
		sum += uint16(b)
	}

	sum += packet.Language
	sum += packet.InfoSerialNumber

	return sum
}

// packetToBytes converts control packet to byte array for transmission
func (g *GPSTrackerController) packetToBytes(packet *ControlPacket) []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, packet.StartBit)
	binary.Write(buf, binary.BigEndian, packet.PacketLength)
	binary.Write(buf, binary.BigEndian, packet.ProtocolNumber)
	binary.Write(buf, binary.BigEndian, packet.CommandLength)
	buf.Write(packet.ServerFlag[:])
	buf.Write([]byte(packet.CommandContent))
	binary.Write(buf, binary.BigEndian, packet.Language)
	binary.Write(buf, binary.BigEndian, packet.InfoSerialNumber)
	binary.Write(buf, binary.BigEndian, packet.ErrorCheck)
	binary.Write(buf, binary.BigEndian, packet.StopBit)

	return buf.Bytes()
}

// sendCommand sends a control command to the GPS tracker and waits for response
func (g *GPSTrackerController) sendCommand(command string) (*ControlResponse, error) {
	response := &ControlResponse{
		Command:    command,
		Timestamp:  time.Now(),
		DeviceIMEI: g.deviceIMEI,
	}

	// Build and send packet
	packet := g.buildControlPacket(command)
	data := g.packetToBytes(packet)

	log.Printf("Sending command %s to device %s", command, g.deviceIMEI)
	log.Printf("Packet bytes: %x", data)

	_, err := g.conn.Write(data)
	if err != nil {
		response.Success = false
		response.Message = fmt.Sprintf("Failed to send command: %v", err)
		return response, fmt.Errorf("failed to send command: %v", err)
	}

	// Read response with timeout
	responseData := make([]byte, 256)
	g.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	n, err := g.conn.Read(responseData)
	if err != nil {
		response.Success = false
		response.Message = fmt.Sprintf("Failed to read response: %v", err)
		return response, fmt.Errorf("failed to read response: %v", err)
	}

	responseStr, parseErr := g.parseResponse(responseData[:n])
	response.Response = responseStr

	if parseErr != nil {
		response.Success = false
		response.Message = fmt.Sprintf("Failed to parse response: %v", parseErr)
		return response, parseErr
	}

	// Analyze response for success/failure
	response.Success = g.isSuccessfulResponse(command, responseStr)
	response.Message = g.getResponseMessage(command, responseStr)

	return response, nil
}

// parseResponse extracts the command content from terminal response
func (g *GPSTrackerController) parseResponse(data []byte) (string, error) {
	if len(data) < 10 {
		return "", fmt.Errorf("response too short")
	}

	// Skip header and extract command content
	// This is a simplified parser - in real implementation, you'd parse the full packet structure
	commandStart := 8           // Approximate position where command content starts
	commandEnd := len(data) - 6 // Approximate position where command content ends

	if commandStart >= commandEnd {
		return "", fmt.Errorf("invalid response format")
	}

	return string(data[commandStart:commandEnd]), nil
}

// isSuccessfulResponse checks if the response indicates success
func (g *GPSTrackerController) isSuccessfulResponse(command, response string) bool {
	switch command {
	case CmdCutOil:
		return contains(response, "Success")
	case CmdConnectOil:
		return contains(response, "Success")
	case CmdLocation:
		return !contains(response, "Fail")
	default:
		return contains(response, "Success")
	}
}

// getResponseMessage returns a human-readable message based on the response
func (g *GPSTrackerController) getResponseMessage(command, response string) string {
	switch command {
	case CmdCutOil:
		switch {
		case contains(response, "Success"):
			return "Oil and electricity successfully cut"
		case contains(response, "Speed Limit"):
			return "Cannot cut oil - vehicle speed too high (>20km/h)"
		case contains(response, "Unvalued Fix"):
			return "Cannot cut oil - GPS tracking is off"
		default:
			return fmt.Sprintf("Unknown response: %s", response)
		}
	case CmdConnectOil:
		switch {
		case contains(response, "Success"):
			return "Oil and electricity successfully connected"
		case contains(response, "Fail"):
			return "Failed to connect oil and electricity"
		default:
			return fmt.Sprintf("Unknown response: %s", response)
		}
	case CmdLocation:
		return fmt.Sprintf("Location response: %s", response)
	default:
		return response
	}
}

// CutOilAndElectricity sends command to cut oil and electricity
func (g *GPSTrackerController) CutOilAndElectricity() (*ControlResponse, error) {
	log.Printf("=== CUTTING OIL AND ELECTRICITY for device %s ===", g.deviceIMEI)

	response, err := g.sendCommand(CmdCutOil)
	if err != nil {
		return response, fmt.Errorf("failed to cut oil and electricity: %v", err)
	}

	if response.Success {
		log.Printf("✅ Oil and electricity successfully cut for device %s", g.deviceIMEI)
	} else {
		log.Printf("❌ Failed to cut oil and electricity for device %s: %s", g.deviceIMEI, response.Message)
	}

	return response, nil
}

// ConnectOilAndElectricity sends command to connect oil and electricity
func (g *GPSTrackerController) ConnectOilAndElectricity() (*ControlResponse, error) {
	log.Printf("=== CONNECTING OIL AND ELECTRICITY for device %s ===", g.deviceIMEI)

	response, err := g.sendCommand(CmdConnectOil)
	if err != nil {
		return response, fmt.Errorf("failed to connect oil and electricity: %v", err)
	}

	if response.Success {
		log.Printf("✅ Oil and electricity successfully connected for device %s", g.deviceIMEI)
	} else {
		log.Printf("❌ Failed to connect oil and electricity for device %s: %s", g.deviceIMEI, response.Message)
	}

	return response, nil
}

// GetLocation sends command to get current location
func (g *GPSTrackerController) GetLocation() (*ControlResponse, error) {
	log.Printf("=== GETTING LOCATION for device %s ===", g.deviceIMEI)

	response, err := g.sendCommand(CmdLocation)
	if err != nil {
		return response, fmt.Errorf("failed to get location: %v", err)
	}

	log.Printf("Location response for device %s: %s", g.deviceIMEI, response.Response)
	return response, nil
}

// Helper function to check if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}
