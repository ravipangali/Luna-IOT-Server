package protocol

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"luna_iot_server/pkg/colors"
	"strings"
	"time"
)

// GT06Decoder handles decoding of GT06 protocol packets
type GT06Decoder struct {
	buffer []byte

	// GT06 protocol supports both start bit patterns
	startBits78      []byte // Standard packets
	startBits79      []byte // Extended packets
	stopBits         []byte
	protocolNumbers  map[byte]string
	responseRequired []byte
}

// DecodedPacket represents a decoded GT06 packet
type DecodedPacket struct {
	Raw           string      `json:"raw"`
	Timestamp     time.Time   `json:"timestamp"`
	Length        byte        `json:"length"`
	Protocol      byte        `json:"protocol"`
	ProtocolName  string      `json:"protocolName"`
	SerialNumber  byte        `json:"serialNumber"`
	Checksum      byte        `json:"checksum"`
	NeedsResponse bool        `json:"needsResponse"`
	Data          interface{} `json:"data,omitempty"`

	// Login data
	TerminalID     string  `json:"terminalId,omitempty"`
	DeviceType     *uint16 `json:"deviceType,omitempty"`
	TimezoneOffset *int16  `json:"timezoneOffset,omitempty"`

	// GPS data
	GPSTime       *time.Time `json:"gpsTime,omitempty"`
	Latitude      *float64   `json:"latitude,omitempty"`
	Longitude     *float64   `json:"longitude,omitempty"`
	Speed         *byte      `json:"speed,omitempty"`
	Course        *uint16    `json:"course,omitempty"`
	Altitude      *int       `json:"altitude,omitempty"`
	GPSRealTime   *bool      `json:"gpsRealTime,omitempty"`
	GPSPositioned *bool      `json:"gpsPositioned,omitempty"`
	EastLongitude *bool      `json:"eastLongitude,omitempty"`
	NorthLatitude *bool      `json:"northLatitude,omitempty"`
	Satellites    *byte      `json:"satellites,omitempty"`

	// LBS data
	MCC    *uint16 `json:"mcc,omitempty"`
	MNC    *byte   `json:"mnc,omitempty"`
	LAC    *uint16 `json:"lac,omitempty"`
	CellID *uint32 `json:"cellId,omitempty"`

	// Status data
	Ignition       string       `json:"ignition,omitempty"`
	Charger        string       `json:"charger,omitempty"`
	GPSTracking    string       `json:"gpsTracking,omitempty"`
	Alarm          *AlarmInfo   `json:"alarm,omitempty"`
	Voltage        *VoltageInfo `json:"voltage,omitempty"`
	GSMSignal      *GSMInfo     `json:"gsmSignal,omitempty"`
	OilElectricity string       `json:"oilElectricity,omitempty"`
	DeviceStatus   string       `json:"deviceStatus,omitempty"`
	StatusByte     string       `json:"statusByte,omitempty"`
	BinaryRepr     string       `json:"binaryRepresentation,omitempty"`
	RawData        string       `json:"rawData,omitempty"`

	// Alarm data
	AlarmType *AlarmTypeInfo `json:"alarmType,omitempty"`

	// Additional data
	AdditionalData string `json:"additionalData,omitempty"`
}

// AlarmInfo represents alarm information
type AlarmInfo struct {
	Active bool   `json:"active"`
	Type   string `json:"type"`
	Code   int    `json:"code"`
}

// VoltageInfo represents voltage information
type VoltageInfo struct {
	Level      byte   `json:"level"`
	Status     string `json:"status"`
	Percentage int    `json:"percentage"`
}

// GSMInfo represents GSM signal information
type GSMInfo struct {
	Level  byte   `json:"level"`
	Status string `json:"status"`
	Bars   int    `json:"bars"`
}

// AlarmTypeInfo represents detailed alarm type information
type AlarmTypeInfo struct {
	Emergency       bool `json:"emergency"`
	Overspeed       bool `json:"overspeed"`
	LowPower        bool `json:"lowPower"`
	Shock           bool `json:"shock"`
	IntoArea        bool `json:"intoArea"`
	OutArea         bool `json:"outArea"`
	LongNoOperation bool `json:"longNoOperation"`
	Distance        bool `json:"distance"`
}

// StartInfo represents start bit information
type StartInfo struct {
	Index int
}

// NewGT06Decoder creates a new GT06 decoder instance
func NewGT06Decoder() *GT06Decoder {
	return &GT06Decoder{
		buffer:      make([]byte, 0),
		startBits78: []byte{0x78, 0x78}, // Standard packets
		startBits79: []byte{0x79, 0x79}, // Extended packets
		stopBits:    []byte{0x0D, 0x0A},
		protocolNumbers: map[byte]string{
			0x01: "LOGIN",
			0x12: "GPS_LBS_STATUS",
			0x13: "STATUS_INFO",
			0x15: "STRING_INFO",
			0x16: "ALARM_DATA",
			0x1A: "GPS_LBS_DATA",
			0x22: "GPS_LBS", // GPS Data Packet - this is what your device is sending
			0xA0: "GPS_LBS_STATUS_A0",
		},
		responseRequired: []byte{0x01, 0x21, 0x15, 0x16, 0x18, 0x19},
	}
}

// AddData adds new data to the buffer and processes it
func (d *GT06Decoder) AddData(data []byte) ([]*DecodedPacket, error) {
	d.clearBuffer()
	d.buffer = append(d.buffer, data...)
	return d.processBuffer()
}

// clearBuffer clears the internal buffer
func (d *GT06Decoder) clearBuffer() {
	d.buffer = make([]byte, 0)
}

// processBuffer processes the buffer and extracts packets
func (d *GT06Decoder) processBuffer() ([]*DecodedPacket, error) {
	var packets []*DecodedPacket

	for len(d.buffer) >= 5 {
		startInfo := d.findStartBits()

		if startInfo.Index == -1 {
			d.buffer = make([]byte, 0)
			break
		}

		if startInfo.Index > 0 {
			d.buffer = d.buffer[startInfo.Index:]
		}

		if len(d.buffer) < 5 {
			break
		}

		lengthByte := d.buffer[2]
		totalLength := int(lengthByte) + 5

		if len(d.buffer) < totalLength {
			break
		}

		packet := d.buffer[0:totalLength]

		if packet[totalLength-2] == 0x0D && packet[totalLength-1] == 0x0A {
			decoded, err := d.decodePacket(packet)
			if err != nil {
				colors.PrintError("Error decoding packet: %v", err)
			} else if decoded != nil {
				packets = append(packets, decoded)
			}
		}

		d.buffer = d.buffer[totalLength:]
	}

	return packets, nil
}

// findStartBits finds the start bits in the buffer
func (d *GT06Decoder) findStartBits() StartInfo {
	for i := 0; i <= len(d.buffer)-2; i++ {
		// Check for 7878 start bits
		if d.buffer[i] == 0x78 && d.buffer[i+1] == 0x78 {
			return StartInfo{Index: i}
		}
		// Check for 7979 start bits (extended packets)
		if d.buffer[i] == 0x79 && d.buffer[i+1] == 0x79 {
			return StartInfo{Index: i}
		}
	}
	return StartInfo{Index: -1}
}

// decodePacket decodes a single packet
func (d *GT06Decoder) decodePacket(packet []byte) (*DecodedPacket, error) {
	if len(packet) < 5 {
		return nil, nil
	}

	protocolOffset := 3
	dataStartOffset := 4
	serialOffset := len(packet) - 6
	checksumOffset := len(packet) - 4

	result := &DecodedPacket{
		Raw:           strings.ToUpper(hex.EncodeToString(packet)),
		Timestamp:     time.Now(), // Will be updated with GPS time if available
		Length:        packet[2],
		Protocol:      packet[protocolOffset],
		ProtocolName:  d.getProtocolName(packet[protocolOffset]),
		SerialNumber:  0,
		Checksum:      0,
		NeedsResponse: d.needsResponse(packet[protocolOffset]),
	}

	if serialOffset >= 0 {
		result.SerialNumber = packet[serialOffset]
	}
	if checksumOffset >= 0 {
		result.Checksum = packet[checksumOffset]
	}

	var dataPayload []byte
	if serialOffset > dataStartOffset {
		dataPayload = packet[dataStartOffset:serialOffset]
	}

	colors.PrintDebug("GT06 Data Payload: %v", dataPayload)

	switch packet[protocolOffset] {
	case 0x01:
		d.decodeLogin(dataPayload, result)
	case 0x12, 0x22, 0xA0, 0x1A:
		d.decodeGPSLBS(dataPayload, result)
	case 0x13:
		d.decodeStatusInfo(dataPayload, result)
	case 0x16:
		d.decodeAlarmData(dataPayload, result)
	default:
		result.Data = strings.ToUpper(hex.EncodeToString(dataPayload))
	}

	return result, nil
}

// getProtocolName returns the protocol name for a given protocol number
func (d *GT06Decoder) getProtocolName(protocol byte) string {
	if name, exists := d.protocolNumbers[protocol]; exists {
		return name
	}
	return "UNKNOWN"
}

// needsResponse checks if a protocol requires a response
func (d *GT06Decoder) needsResponse(protocol byte) bool {
	for _, p := range d.responseRequired {
		if p == protocol {
			return true
		}
	}
	return false
}

// decodeLogin decodes login packet data
func (d *GT06Decoder) decodeLogin(data []byte, result *DecodedPacket) {
	if len(data) >= 8 {
		result.TerminalID = strings.ToUpper(hex.EncodeToString(data[0:8]))
		if len(data) > 8 {
			deviceType := binary.BigEndian.Uint16(data[8:10])
			result.DeviceType = &deviceType
		}
		if len(data) > 10 {
			timezoneOffset := int16(binary.BigEndian.Uint16(data[10:12]))
			result.TimezoneOffset = &timezoneOffset
		}
	}
}

// decodeGPSLBS decodes GPS and LBS data
func (d *GT06Decoder) decodeGPSLBS(data []byte, result *DecodedPacket) {
	if len(data) < 12 {
		return
	}

	offset := 0

	// Decode time
	if len(data) >= 6 {
		year := 2000 + int(data[offset])
		month := int(data[offset+1])
		day := int(data[offset+2])
		hour := int(data[offset+3])
		minute := int(data[offset+4])
		second := int(data[offset+5])
		offset += 6

		if year >= 2000 && year <= 2050 && month >= 1 && month <= 12 &&
			day >= 1 && day <= 31 && hour <= 23 && minute <= 59 && second <= 59 {
			gpsTime := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC)
			result.GPSTime = &gpsTime
			// Use GPS time as the main timestamp for the packet
			result.Timestamp = gpsTime
		}
	}

	// if result.Protocol == 0xA0 {
	if offset < len(data) {
		// Parse satellites count from first byte (upper 4 bits)
		if offset < len(data) {
			satellitesByte := data[offset]
			satellites := (satellitesByte >> 4) & 0x0F
			result.Satellites = &satellites
		}
		offset += 1

		if offset+12 <= len(data) {
			latRaw := binary.BigEndian.Uint32(data[offset : offset+4])
			if latRaw > 0 && latRaw < 0xFFFFFFFF {
				lat := float64(latRaw) / 1800000.0

				// Convert negative latitude to positive (remove minus sign)
				if lat < 0 {
					lat = -lat
				}

				if lat > 0 && lat <= 90 {
					result.Latitude = &lat
				}
			}
			offset += 4

			lngRaw := binary.BigEndian.Uint32(data[offset : offset+4])
			if lngRaw > 0 && lngRaw < 0xFFFFFFFF {
				lng := float64(lngRaw) / 1800000.0

				// Accept both negative and positive longitude values
				if lng >= -180 && lng <= 180 {
					result.Longitude = &lng
				}
			}
			offset += 4

			if offset+3 <= len(data) {
				speed := data[offset]
				result.Speed = &speed

				courseStatus := binary.BigEndian.Uint16(data[offset+1 : offset+3])
				course := courseStatus & 0x03FF
				result.Course = &course

				// Status flags
				gpsRealTime := (courseStatus & 0x2000) == 0
				result.GPSRealTime = &gpsRealTime

				gpsPositioned := (courseStatus & 0x1000) == 0
				result.GPSPositioned = &gpsPositioned

				eastLongitude := (courseStatus & 0x0800) == 0
				result.EastLongitude = &eastLongitude

				northLatitude := (courseStatus & 0x0400) == 0
				result.NorthLatitude = &northLatitude

				// Apply hemisphere corrections for longitude only
				// Longitude: negative for western hemisphere, positive for eastern
				if result.Longitude != nil && !eastLongitude {
					lng := -*result.Longitude
					result.Longitude = &lng
				}

				// For latitude: always convert to positive regardless of hemisphere
				if result.Latitude != nil && !northLatitude {
					// Convert negative latitude to positive
					lat := *result.Latitude
					if lat < 0 {
						lat = -lat
					}
					result.Latitude = &lat
				}

				offset += 3
			}
		}

		// Parse altitude if available (after GPS data)
		if offset+2 <= len(data) {
			altitudeRaw := binary.BigEndian.Uint16(data[offset : offset+2])
			if altitudeRaw > 0 && altitudeRaw < 0xFFFF {
				altitude := int(altitudeRaw)
				result.Altitude = &altitude
			}
			offset += 2
		}

		// Decode cell tower information
		for offset+9 <= len(data) {
			testMCC := binary.BigEndian.Uint16(data[offset : offset+2])

			if testMCC >= 100 && testMCC <= 999 {
				result.MCC = &testMCC
				mnc := data[offset+2]
				result.MNC = &mnc
				lac := binary.BigEndian.Uint16(data[offset+3 : offset+5])
				result.LAC = &lac

				cellId1 := data[offset+5]
				cellId2 := data[offset+6]
				cellId3 := data[offset+7]
				cellId := (uint32(cellId1) << 16) | (uint32(cellId2) << 8) | uint32(cellId3)
				result.CellID = &cellId

				offset += 8
				break
			}
			offset += 1
		}
	}
	// } else {
	// 	if offset < len(data) {
	// 		gpsInfoLength := data[offset]
	// 		offset += 1

	// 		if gpsInfoLength > 0 && gpsInfoLength <= 50 && offset+int(gpsInfoLength) <= len(data) {
	// 			if offset+4 <= len(data) {
	// 				satellites := (data[offset] >> 4) & 0x0F
	// 				result.Satellites = &satellites

	// 				lat1 := data[offset] & 0x0F
	// 				lat2 := data[offset+1]
	// 				lat3 := data[offset+2]
	// 				lat4 := data[offset+3]

	// 				latRaw := (uint32(lat1) << 24) | (uint32(lat2) << 16) | (uint32(lat3) << 8) | uint32(lat4)

	// 				if latRaw > 0 {
	// 					lat := float64(latRaw) / 1800000.0
	// 					result.Latitude = &lat
	// 				}
	// 				offset += 4
	// 			}

	// 			if offset+4 <= len(data) {
	// 				lngRaw := binary.BigEndian.Uint32(data[offset : offset+4])

	// 				if lngRaw > 0 {
	// 					lng := float64(lngRaw) / 1800000.0
	// 					result.Longitude = &lng
	// 				}
	// 				offset += 4
	// 			}

	// 			if offset < len(data) {
	// 				speed := data[offset]
	// 				result.Speed = &speed
	// 				offset += 1
	// 			}

	// 			if offset+2 <= len(data) {
	// 				courseStatus := binary.BigEndian.Uint16(data[offset : offset+2])

	// 				course := courseStatus & 0x03FF
	// 				result.Course = &course

	// 				gpsRealTime := (courseStatus & 0x2000) == 0
	// 				result.GPSRealTime = &gpsRealTime

	// 				gpsPositioned := (courseStatus & 0x1000) == 0
	// 				result.GPSPositioned = &gpsPositioned

	// 				eastLongitude := (courseStatus & 0x0800) == 0
	// 				result.EastLongitude = &eastLongitude

	// 				northLatitude := (courseStatus & 0x0400) == 0
	// 				result.NorthLatitude = &northLatitude

	// 				offset += 2

	// 				// Adjust coordinates
	// 				if result.Longitude != nil && !*result.EastLongitude {
	// 					lng := -*result.Longitude
	// 					result.Longitude = &lng
	// 				}
	// 				if result.Latitude != nil && !*result.NorthLatitude {
	// 					lat := -*result.Latitude
	// 					result.Latitude = &lat
	// 				}
	// 			}
	// 		}
	// 	}

	// 	if offset+9 <= len(data) {
	// 		mcc := binary.BigEndian.Uint16(data[offset : offset+2])
	// 		result.MCC = &mcc

	// 		mnc := data[offset+2]
	// 		result.MNC = &mnc

	// 		lac := binary.BigEndian.Uint16(data[offset+3 : offset+5])
	// 		result.LAC = &lac

	// 		cellId1 := data[offset+5]
	// 		cellId2 := data[offset+6]
	// 		cellId3 := data[offset+7]
	// 		cellId := (uint32(cellId1) << 16) | (uint32(cellId2) << 8) | uint32(cellId3)
	// 		result.CellID = &cellId

	// 		offset += 8
	// 	}
	// }

	if len(data) > offset {
		result.AdditionalData = strings.ToUpper(hex.EncodeToString(data[offset:]))
	}
}

// decodeStatusInfo decodes status information
func (d *GT06Decoder) decodeStatusInfo(data []byte, result *DecodedPacket) {
	if len(data) < 3 {
		colors.PrintWarning("Status info data too short")
		return
	}

	// Terminal Information Byte (first byte) - GT06 Protocol Documentation
	terminalInfoByte := data[0]

	// IGNITION STATUS (Bit1 - ACC)
	if (terminalInfoByte & 0x02) != 0 {
		result.Ignition = "ON"
	} else {
		result.Ignition = "OFF"
	}

	// CHARGER STATUS (Bit2 - Charging)
	if (terminalInfoByte & 0x04) != 0 {
		result.Charger = "CONNECTED"
	} else {
		result.Charger = "DISCONNECTED"
	}

	// GPS TRACKING STATUS (Bit6)
	if (terminalInfoByte & 0x40) != 0 {
		result.GPSTracking = "ENABLED"
	} else {
		result.GPSTracking = "DISABLED"
	}

	// ALARM STATUS (Bits 5,4,3)
	alarmBits := (terminalInfoByte >> 3) & 0x07
	result.Alarm = &AlarmInfo{
		Active: alarmBits != 0,
		Type:   d.getAlarmTypeFromBits(alarmBits),
		Code:   int(alarmBits),
	}

	// VOLTAGE LEVEL (Second byte)
	voltageLevel := data[1]
	result.Voltage = &VoltageInfo{
		Level:      voltageLevel,
		Status:     d.getVoltageStatus(voltageLevel),
		Percentage: d.getVoltagePercentage(voltageLevel),
	}

	// GSM SIGNAL STRENGTH (Third byte)
	gsmLevel := data[2]
	bars := int(gsmLevel)
	if bars > 4 {
		bars = 4
	}
	result.GSMSignal = &GSMInfo{
		Level:  gsmLevel,
		Status: d.getGsmStatus(gsmLevel),
		Bars:   bars,
	}

	// Additional status from documentation
	if (terminalInfoByte & 0x80) == 0 {
		result.OilElectricity = "CONNECTED"
	} else {
		result.OilElectricity = "DISCONNECTED"
	}

	if (terminalInfoByte & 0x01) != 0 {
		result.DeviceStatus = "ACTIVATED"
	} else {
		result.DeviceStatus = "DEACTIVATED"
	}

	// Debug info
	result.StatusByte = fmt.Sprintf("0x%02X", terminalInfoByte)
	result.BinaryRepr = fmt.Sprintf("%08b", terminalInfoByte)
	result.RawData = strings.ToUpper(hex.EncodeToString(data))
}

// getVoltageStatus returns voltage status string
func (d *GT06Decoder) getVoltageStatus(level byte) string {
	switch level {
	case 0:
		return "NO_POWER"
	case 1:
		return "CRITICALLY_LOW"
	case 2:
		return "VERY_LOW"
	case 3:
		return "LOW"
	case 4:
		return "MEDIUM"
	case 5:
		return "HIGH"
	case 6:
		return "VERY_HIGH"
	default:
		return "UNKNOWN"
	}
}

// getVoltagePercentage returns voltage percentage
func (d *GT06Decoder) getVoltagePercentage(level byte) int {
	percentages := []int{0, 10, 25, 40, 60, 80, 100}
	if int(level) < len(percentages) {
		return percentages[level]
	}
	return 0
}

// getGsmStatus returns GSM signal status string
func (d *GT06Decoder) getGsmStatus(level byte) string {
	switch level {
	case 0:
		return "NO_SIGNAL"
	case 1:
		return "VERY_WEAK"
	case 2:
		return "WEAK"
	case 3:
		return "GOOD"
	case 4:
		return "EXCELLENT"
	default:
		return "UNKNOWN"
	}
}

// getAlarmTypeFromBits returns alarm type from bits
func (d *GT06Decoder) getAlarmTypeFromBits(alarmBits byte) string {
	switch alarmBits {
	case 0b000:
		return "NORMAL" // 000: Normal
	case 0b001:
		return "SHOCK" // 001: Shock Alarm
	case 0b010:
		return "POWER_CUT" // 010: Power Cut Alarm
	case 0b011:
		return "LOW_BATTERY" // 011: Low Battery Alarm
	case 0b100:
		return "SOS" // 100: SOS
	default:
		return "UNKNOWN"
	}
}

// decodeAlarmData decodes alarm data
func (d *GT06Decoder) decodeAlarmData(data []byte, result *DecodedPacket) {
	if len(data) >= 1 {
		alarmType := data[0]
		result.AlarmType = &AlarmTypeInfo{
			Emergency:       (alarmType & 0x01) != 0,
			Overspeed:       (alarmType & 0x02) != 0,
			LowPower:        (alarmType & 0x04) != 0,
			Shock:           (alarmType & 0x08) != 0,
			IntoArea:        (alarmType & 0x10) != 0,
			OutArea:         (alarmType & 0x20) != 0,
			LongNoOperation: (alarmType & 0x40) != 0,
			Distance:        (alarmType & 0x80) != 0,
		}
	}

	// Decode GPS data if present
	if len(data) > 1 {
		d.decodeGPSLBS(data[1:], result)
	}
}

// GenerateResponse generates a response packet
func (d *GT06Decoder) GenerateResponse(serialNumber uint16, protocolNumber byte) []byte {
	response := make([]byte, 10)
	offset := 0

	// Start bits
	response[offset] = 0x78
	offset++
	response[offset] = 0x78
	offset++

	// Length
	response[offset] = 0x05
	offset++

	// Protocol number
	response[offset] = protocolNumber
	offset++

	// Serial number
	binary.BigEndian.PutUint16(response[offset:], serialNumber)
	offset += 2

	// CRC (simplified - in real implementation should calculate proper CRC)
	crc := d.calculateCRC(response[2:offset])
	binary.BigEndian.PutUint16(response[offset:], crc)
	offset += 2

	// Stop bits
	response[offset] = 0x0D
	offset++
	response[offset] = 0x0A

	return response
}

// calculateCRC calculates CRC for packet (simplified implementation)
func (d *GT06Decoder) calculateCRC(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for i := 0; i < len(data); i++ {
		crc ^= uint16(data[i])
		for j := 0; j < 8; j++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0x8408
			} else {
				crc >>= 1
			}
		}
	}
	return (^crc) & 0xFFFF
}
