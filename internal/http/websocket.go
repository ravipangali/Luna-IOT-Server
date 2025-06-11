package http

import (
	"encoding/json"
	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocket upgrader configuration
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin in development
		// In production, you should validate the origin
		return true
	},
}

// WebSocketHub manages WebSocket connections
type WebSocketHub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mutex      sync.RWMutex
}

// WebSocketMessage represents a message sent through WebSocket
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// GPSUpdate represents real-time GPS data
type GPSUpdate struct {
	IMEI         string   `json:"imei"`
	DeviceName   string   `json:"device_name,omitempty"`
	VehicleName  string   `json:"vehicle_name,omitempty"`
	RegNo        string   `json:"reg_no,omitempty"`
	VehicleType  string   `json:"vehicle_type,omitempty"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	Speed        *int     `json:"speed"`
	Course       *int     `json:"course"`
	Ignition     string   `json:"ignition"`
	Timestamp    string   `json:"timestamp"`
	ProtocolName string   `json:"protocol_name"`

	// Enhanced status information
	Battery      *BatteryInfo `json:"battery,omitempty"`
	Signal       *SignalInfo  `json:"signal,omitempty"`
	DeviceStatus *DeviceInfo  `json:"device_status,omitempty"`
	AlarmStatus  *AlarmInfo   `json:"alarm_status,omitempty"`

	// Additional fields for better tracking
	IsMoving         bool   `json:"is_moving"`
	LastSeen         string `json:"last_seen"`
	ConnectionStatus string `json:"connection_status"` // "connected", "stopped", "inactive"

	// Map rotation support
	Bearing *float64 `json:"bearing,omitempty"` // Course converted to bearing (0-360)

	// Enhanced location validation
	LocationValid bool `json:"location_valid"`
	Accuracy      *int `json:"accuracy,omitempty"`
}

// DeviceStatus represents device connection status
type DeviceStatus struct {
	IMEI        string       `json:"imei"`
	Status      string       `json:"status"` // "connected", "stopped", "inactive"
	LastSeen    string       `json:"last_seen"`
	VehicleReg  string       `json:"vehicle_reg,omitempty"`
	VehicleName string       `json:"vehicle_name,omitempty"`
	VehicleType string       `json:"vehicle_type,omitempty"`
	Battery     *BatteryInfo `json:"battery,omitempty"`
	Signal      *SignalInfo  `json:"signal,omitempty"`
}

// BatteryInfo represents battery/voltage information
type BatteryInfo struct {
	Level    int    `json:"level"`    // 0-100 percentage
	Voltage  int    `json:"voltage"`  // Raw voltage level
	Status   string `json:"status"`   // "Normal", "Low", "Critical"
	Charging bool   `json:"charging"` // Whether charger is connected
}

// SignalInfo represents GSM signal information
type SignalInfo struct {
	Level      int    `json:"level"`      // Raw signal level
	Bars       int    `json:"bars"`       // 0-5 bars
	Status     string `json:"status"`     // "Excellent", "Good", "Fair", "Poor", "No Signal"
	Percentage int    `json:"percentage"` // 0-100 percentage
}

// DeviceInfo represents detailed device status
type DeviceInfo struct {
	Activated     bool `json:"activated"`
	GPSTracking   bool `json:"gps_tracking"`
	OilConnected  bool `json:"oil_connected"`
	EngineRunning bool `json:"engine_running"`
	Satellites    int  `json:"satellites"`
}

// AlarmInfo represents alarm status
type AlarmInfo struct {
	Active    bool   `json:"active"`
	Type      string `json:"type"`
	Code      int    `json:"code"`
	Emergency bool   `json:"emergency"`
	Overspeed bool   `json:"overspeed"`
	LowPower  bool   `json:"low_power"`
	Shock     bool   `json:"shock"`
}

// Global WebSocket hub instance
var WSHub *WebSocketHub

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	colors.PrintServer("🔗", "WebSocket Hub started - Ready for real-time connections")

	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			colors.PrintConnection("📱", "WebSocket client connected. Total clients: %d", len(h.clients))

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mutex.Unlock()
			colors.PrintConnection("📱", "WebSocket client disconnected. Total clients: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
					colors.PrintError("Error sending message to WebSocket client: %v", err)
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// BroadcastGPSUpdate sends GPS data updates to all connected clients
func (h *WebSocketHub) BroadcastGPSUpdate(gpsData *models.GPSData, vehicleName, regNo string) {
	// Get vehicle type information
	var vehicle models.Vehicle
	vehicleType := ""
	if err := db.GetDB().Where("imei = ?", gpsData.IMEI).First(&vehicle).Error; err == nil {
		vehicleType = string(vehicle.VehicleType)
	}

	// Validate GPS coordinates
	locationValid := false
	if gpsData.Latitude != nil && gpsData.Longitude != nil {
		lat := *gpsData.Latitude
		lng := *gpsData.Longitude
		locationValid = lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180 && lat != 0 && lng != 0
	}

	// Determine if vehicle is moving based on speed
	isMoving := false
	if gpsData.Speed != nil && *gpsData.Speed > 0 {
		isMoving = true
	}

	// Build battery information
	var battery *BatteryInfo
	if gpsData.VoltageLevel != nil {
		battery = &BatteryInfo{
			Level:    getVoltagePercentage(*gpsData.VoltageLevel),
			Voltage:  *gpsData.VoltageLevel,
			Status:   gpsData.VoltageStatus,
			Charging: gpsData.Charger == "CONNECTED",
		}
	}

	// Build signal information
	var signal *SignalInfo
	if gpsData.GSMSignal != nil {
		signal = &SignalInfo{
			Level:      *gpsData.GSMSignal,
			Bars:       getSignalBars(*gpsData.GSMSignal),
			Status:     gpsData.GSMStatus,
			Percentage: getSignalPercentage(*gpsData.GSMSignal),
		}
	}

	// Build device status information with improved ignition logic
	deviceStatus := &DeviceInfo{
		Activated:     gpsData.DeviceStatus == "ACTIVATED",
		GPSTracking:   gpsData.GPSTracking == "ENABLED",
		OilConnected:  gpsData.OilElectricity == "CONNECTED",
		EngineRunning: gpsData.Ignition == "ON",
		Satellites:    0, // Will be set if available
	}
	if gpsData.Satellites != nil {
		deviceStatus.Satellites = *gpsData.Satellites
	}

	// Build alarm information
	var alarmStatus *AlarmInfo
	if gpsData.AlarmActive {
		alarmStatus = &AlarmInfo{
			Active: gpsData.AlarmActive,
			Type:   gpsData.AlarmType,
			Code:   gpsData.AlarmCode,
			// Parse alarm type for specific flags
			Emergency: gpsData.AlarmType == "Emergency",
			Overspeed: gpsData.AlarmType == "Overspeed",
			LowPower:  gpsData.AlarmType == "Low Power",
			Shock:     gpsData.AlarmType == "Shock",
		}
	}

	// Convert course to bearing for map rotation (0-360 degrees)
	var bearing *float64
	if gpsData.Course != nil {
		bearingValue := float64(*gpsData.Course)
		// Ensure bearing is in 0-360 range
		if bearingValue < 0 {
			bearingValue += 360
		}
		if bearingValue >= 360 {
			bearingValue = bearingValue - 360*float64(int(bearingValue/360))
		}
		bearing = &bearingValue
	}

	// Determine connection status
	connectionStatus := "connected"
	if gpsData.Timestamp.Before(time.Now().Add(-1 * time.Hour)) {
		connectionStatus = "timeout"
	}

	update := GPSUpdate{
		IMEI:         gpsData.IMEI,
		VehicleName:  vehicleName,
		RegNo:        regNo,
		VehicleType:  vehicleType,
		Latitude:     gpsData.Latitude,
		Longitude:    gpsData.Longitude,
		Speed:        gpsData.Speed,
		Course:       gpsData.Course,
		Ignition:     gpsData.Ignition,
		Timestamp:    gpsData.Timestamp.Format("2006-01-02T15:04:05Z"),
		ProtocolName: gpsData.ProtocolName,
		Battery:      battery,
		Signal:       signal,
		DeviceStatus: deviceStatus,
		AlarmStatus:  alarmStatus,

		// Enhanced fields
		IsMoving:         isMoving,
		LastSeen:         gpsData.Timestamp.Format("2006-01-02T15:04:05Z"),
		ConnectionStatus: connectionStatus,
		Bearing:          bearing,
		LocationValid:    locationValid,
	}

	message := WebSocketMessage{
		Type:      "gps_update",
		Timestamp: gpsData.Timestamp.Format("2006-01-02T15:04:05Z"),
		Data:      update,
	}

	if data, err := json.Marshal(message); err == nil {
		h.broadcast <- data
		colors.PrintData("📡", "Broadcasted enhanced GPS update for IMEI %s to %d clients (Valid: %v, Moving: %v)",
			gpsData.IMEI, len(h.clients), locationValid, isMoving)
	} else {
		colors.PrintError("Error marshaling GPS update: %v", err)
	}
}

// BroadcastDeviceStatus sends device status updates with enhanced information
func (h *WebSocketHub) BroadcastDeviceStatus(imei, status, vehicleReg string) {
	// Get vehicle information
	var vehicle models.Vehicle
	vehicleName := ""
	vehicleType := ""
	if err := db.GetDB().Where("imei = ?", imei).First(&vehicle).Error; err == nil {
		vehicleName = vehicle.Name
		vehicleType = string(vehicle.VehicleType)
		if vehicleReg == "" {
			vehicleReg = vehicle.RegNo
		}
	}

	// Get latest GPS data for additional context
	var latestGPS models.GPSData
	var battery *BatteryInfo
	var signal *SignalInfo
	if err := db.GetDB().Where("imei = ?", imei).Order("timestamp DESC").First(&latestGPS).Error; err == nil {
		if latestGPS.VoltageLevel != nil {
			battery = &BatteryInfo{
				Level:    getVoltagePercentage(*latestGPS.VoltageLevel),
				Voltage:  *latestGPS.VoltageLevel,
				Status:   latestGPS.VoltageStatus,
				Charging: latestGPS.Charger == "CONNECTED",
			}
		}
		if latestGPS.GSMSignal != nil {
			signal = &SignalInfo{
				Level:      *latestGPS.GSMSignal,
				Bars:       getSignalBars(*latestGPS.GSMSignal),
				Status:     latestGPS.GSMStatus,
				Percentage: getSignalPercentage(*latestGPS.GSMSignal),
			}
		}
	}

	statusUpdate := DeviceStatus{
		IMEI:        imei,
		Status:      status,
		LastSeen:    time.Now().Format("2006-01-02T15:04:05Z"),
		VehicleReg:  vehicleReg,
		VehicleName: vehicleName,
		VehicleType: vehicleType,
		Battery:     battery,
		Signal:      signal,
	}

	message := WebSocketMessage{
		Type:      "device_status",
		Timestamp: time.Now().Format("2006-01-02T15:04:05Z"),
		Data:      statusUpdate,
	}

	if data, err := json.Marshal(message); err == nil {
		h.broadcast <- data
		colors.PrintConnection("📡", "Broadcasted device status for IMEI %s: %s (%s)", imei, status, vehicleName)
	}
}

// HandleWebSocket handles WebSocket connections
func HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		colors.PrintError("Failed to upgrade to WebSocket: %v", err)
		return
	}

	colors.PrintConnection("🔗", "New WebSocket connection from %s", c.ClientIP())

	// Register the connection
	WSHub.register <- conn

	// Handle connection in a goroutine
	go func() {
		defer func() {
			WSHub.unregister <- conn
		}()

		// Keep connection alive and handle incoming messages
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					colors.PrintError("WebSocket error: %v", err)
				}
				break
			}
			// Handle incoming messages if needed (ping/pong, subscriptions, etc.)
		}
	}()
}

// InitializeWebSocket initializes the global WebSocket hub
func InitializeWebSocket() {
	WSHub = NewWebSocketHub()
	go WSHub.Run()
}

// Helper functions for status calculations

// getVoltagePercentage converts voltage level to percentage
func getVoltagePercentage(level int) int {
	// Voltage levels typically range from 0-255 or 0-100 depending on device
	// This is a simplified calculation - adjust based on your device specifications
	if level >= 100 {
		return 100
	}
	if level <= 0 {
		return 0
	}

	// Assume level is 0-100 range, or convert from 0-255 range
	if level > 100 {
		return (level * 100) / 255
	}
	return level
}

// getSignalBars converts GSM signal level to bars (0-5)
func getSignalBars(level int) int {
	// Convert signal level to bars (0-5)
	if level >= 80 {
		return 5
	} else if level >= 60 {
		return 4
	} else if level >= 40 {
		return 3
	} else if level >= 20 {
		return 2
	} else if level > 0 {
		return 1
	}
	return 0
}

// getSignalPercentage converts GSM signal level to percentage
func getSignalPercentage(level int) int {
	// Assume signal level is 0-100 range, or convert from other ranges
	if level >= 100 {
		return 100
	}
	if level <= 0 {
		return 0
	}

	// If level is in 0-255 range, convert to percentage
	if level > 100 {
		return (level * 100) / 255
	}
	return level
}
