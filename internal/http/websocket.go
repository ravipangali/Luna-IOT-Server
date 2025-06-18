package http

import (
	"encoding/json"
	"fmt"
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
	Altitude     *int     `json:"altitude"` // meters above sea level
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
	// Get vehicle information for overspeed checking
	var vehicle models.Vehicle
	vehicleType := ""
	overspeedLimit := 60 // Default overspeed limit
	if err := db.GetDB().Where("imei = ?", gpsData.IMEI).First(&vehicle).Error; err == nil {
		vehicleType = string(vehicle.VehicleType)
		overspeedLimit = vehicle.Overspeed
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
	currentSpeed := 0
	if gpsData.Speed != nil {
		currentSpeed = *gpsData.Speed
		isMoving = currentSpeed > 5 // Consider moving if speed > 5 km/h
	}

	// CRITICAL FIX: Strong checking for GPS data existence and age
	var gpsCount int64
	db.GetDB().Model(&models.GPSData{}).Where("imei = ?", gpsData.IMEI).Count(&gpsCount)
	hasGPSData := gpsCount > 0

	// Calculate data age precisely using GPS timestamp
	dataAge := time.Since(gpsData.Timestamp)
	dataAgeMinutes := dataAge.Minutes()

	// FIXED: Determine status purely based on GPS data with proper logic
	var connectionStatus string

	// Log for debugging
	colors.PrintInfo("🔍", "Status Check for IMEI %s: HasGPSData=%v, DataAge=%.1f min, Speed=%d, Ignition=%s",
		gpsData.IMEI, hasGPSData, dataAgeMinutes, currentSpeed, gpsData.Ignition)

	// Apply user-specified status logic with strong validation
	if !hasGPSData {
		connectionStatus = "nodata" // Only if literally no GPS data exists in database
		colors.PrintWarning("⚠️", "IMEI %s: No GPS data in database", gpsData.IMEI)
	} else if dataAgeMinutes >= 60 {
		connectionStatus = "inactive" // More than 1 hour old GPS data
		colors.PrintWarning("⏰", "IMEI %s: GPS data is %.1f minutes old (>60min)", gpsData.IMEI, dataAgeMinutes)
	} else {
		// GPS data is less than 1 hour old, determine based on speed first
		ignition := gpsData.Ignition

		// FIXED: For running and overspeed states, only speed matters (ignore ignition and connection)
		// Speed > 5 = running state regardless of device connection status
		if isMoving { // isMoving means speed > 5
			if currentSpeed > overspeedLimit {
				connectionStatus = "overspeed"
				colors.PrintError("🚨", "IMEI %s: Overspeed %d km/h (limit: %d)", gpsData.IMEI, currentSpeed, overspeedLimit)
			} else {
				connectionStatus = "running"
				colors.PrintSuccess("🟢", "IMEI %s: Running at %d km/h", gpsData.IMEI, currentSpeed)
			}
		} else {
			// For speeds <= 5, check ignition status to differentiate between idle and stop
			if ignition == "ON" {
				connectionStatus = "idle"
				colors.PrintInfo("🟡", "IMEI %s: Idle (ignition ON, speed ≤5)", gpsData.IMEI)
			} else {
				connectionStatus = "stop" // Changed to "stop" to match frontend
				colors.PrintInfo("🔴", "IMEI %s: Stopped (ignition OFF or speed ≤5)", gpsData.IMEI)
			}
		}
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
			Overspeed: gpsData.AlarmType == "Overspeed" || connectionStatus == "overspeed",
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

	update := GPSUpdate{
		IMEI:         gpsData.IMEI,
		VehicleName:  vehicleName,
		RegNo:        regNo,
		VehicleType:  vehicleType,
		Latitude:     gpsData.Latitude,
		Longitude:    gpsData.Longitude,
		Speed:        gpsData.Speed,
		Course:       gpsData.Course,
		Altitude:     gpsData.Altitude,
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
		ConnectionStatus: connectionStatus, // This is based on GPS data, not device connection
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
		latStr := "N/A"
		lngStr := "N/A"
		if gpsData.Latitude != nil && gpsData.Longitude != nil {
			latStr = fmt.Sprintf("%.12f", *gpsData.Latitude)
			lngStr = fmt.Sprintf("%.12f", *gpsData.Longitude)
		}

		// Enhanced logging with course/bearing information for map rotation debugging
		courseStr := "N/A"
		bearingStr := "N/A"
		if gpsData.Course != nil {
			courseStr = fmt.Sprintf("%d°", *gpsData.Course)
		}
		if bearing != nil {
			bearingStr = fmt.Sprintf("%.1f°", *bearing)
		}

		colors.PrintSuccess("📡", "Broadcasted GPS update for IMEI %s to %d clients (Status: %s, Age: %.1f min, Lat: %s, Lng: %s, Course: %s, Bearing: %s, Speed: %d km/h)",
			gpsData.IMEI, len(h.clients), connectionStatus, dataAgeMinutes, latStr, lngStr, courseStr, bearingStr,
			func() int {
				if gpsData.Speed != nil {
					return *gpsData.Speed
				} else {
					return 0
				}
			}())
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

// getVoltagePercentage converts voltage level (0-6) to percentage (0-100)
func getVoltagePercentage(level int) int {
	// Voltage levels range from 0-6, convert to 0-100 percentage
	if level <= 0 {
		return 0
	}
	if level >= 6 {
		return 100
	}
	// Convert 0-6 to 0-100 percentage
	return (level * 100) / 6
}

// getSignalBars converts GSM signal level (0-4) to bars (0-5)
func getSignalBars(level int) int {
	// Convert signal level (0-4) to bars (0-5)
	if level <= 0 {
		return 0
	}
	if level >= 4 {
		return 5
	}
	// Convert 0-4 to 1-5 bars (level 1 = 1 bar, level 4 = 5 bars)
	return level + 1
}

// getSignalPercentage converts GSM signal level (0-4) to percentage (0-100)
func getSignalPercentage(level int) int {
	// Signal levels range from 0-4, convert to 0-100 percentage
	if level <= 0 {
		return 0
	}
	if level >= 4 {
		return 100
	}
	// Convert 0-4 to 0-100 percentage
	percentage := (level * 100) / 4

	// FIXED: Cap at 100% to prevent values like 2500%
	if percentage > 100 {
		return 100
	}

	return percentage
}
