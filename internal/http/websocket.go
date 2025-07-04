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
	clients    map[*websocket.Conn]*ClientInfo
	broadcast  chan []byte
	register   chan *ClientConnection
	unregister chan *websocket.Conn
	mutex      sync.RWMutex
}

// ClientInfo stores information about connected clients
type ClientInfo struct {
	UserID          uint
	AccessibleIMEIs []string
	IsAuthenticated bool
	LastActivity    time.Time
}

// ClientConnection represents a new client connection with user info
type ClientConnection struct {
	Conn   *websocket.Conn
	UserID uint
	IMEIs  []string
}

// WebSocketMessage represents a message sent through WebSocket
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// LocationUpdate represents real-time GPS location data (with coordinates)
type LocationUpdate struct {
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
	Timestamp    string   `json:"timestamp"`
	ProtocolName string   `json:"protocol_name"`

	// Enhanced location validation
	LocationValid bool `json:"location_valid"`
	Accuracy      *int `json:"accuracy,omitempty"`
}

// StatusUpdate represents real-time device status data (without coordinates requirement)
type StatusUpdate struct {
	IMEI         string `json:"imei"`
	DeviceName   string `json:"device_name,omitempty"`
	VehicleName  string `json:"vehicle_name,omitempty"`
	RegNo        string `json:"reg_no,omitempty"`
	VehicleType  string `json:"vehicle_type,omitempty"`
	Speed        *int   `json:"speed"`
	Ignition     string `json:"ignition"`
	Timestamp    string `json:"timestamp"`
	ProtocolName string `json:"protocol_name"`

	// Enhanced status information
	Battery      *BatteryInfo `json:"battery,omitempty"`
	Signal       *SignalInfo  `json:"signal,omitempty"`
	DeviceStatus *DeviceInfo  `json:"device_status,omitempty"`
	AlarmStatus  *AlarmInfo   `json:"alarm_status,omitempty"`

	// Additional fields for better tracking
	IsMoving         bool   `json:"is_moving"`
	LastSeen         string `json:"last_seen"`
	ConnectionStatus string `json:"connection_status"` // "connected", "stopped", "inactive"
}

// GPSUpdate represents real-time GPS data (LEGACY - for backward compatibility)
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
		clients:    make(map[*websocket.Conn]*ClientInfo),
		broadcast:  make(chan []byte),
		register:   make(chan *ClientConnection),
		unregister: make(chan *websocket.Conn),
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	colors.PrintServer("🔗", "WebSocket Hub started - Ready for real-time connections")

	for {
		select {
		case clientConn := <-h.register:
			h.mutex.Lock()
			h.clients[clientConn.Conn] = &ClientInfo{
				UserID:          clientConn.UserID,
				AccessibleIMEIs: clientConn.IMEIs,
				IsAuthenticated: true,
				LastActivity:    time.Now(),
			}
			h.mutex.Unlock()
			colors.PrintConnection("📱", "WebSocket client connected for User ID %d. Total clients: %d", clientConn.UserID, len(h.clients))

		case client := <-h.unregister:
			h.mutex.Lock()
			if clientInfo, ok := h.clients[client]; ok {
				colors.PrintConnection("📱", "WebSocket client disconnected for User ID %d. Total clients: %d", clientInfo.UserID, len(h.clients)-1)
				delete(h.clients, client)
				client.Close()
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.mutex.RLock()
			// To authorize, we need to know the IMEI. We can get this by
			// unmarshalling the message into a temporary struct.
			var msg struct {
				Data struct {
					IMEI string `json:"imei"`
				} `json:"data"`
			}
			if err := json.Unmarshal(message, &msg); err != nil {
				colors.PrintError("Could not unmarshal broadcast message for auth: %v", err)
				h.mutex.RUnlock()
				continue
			}
			imei := msg.Data.IMEI

			// Send to authorized clients only
			for client, clientInfo := range h.clients {
				if clientInfo.IsAuthenticated && h.isClientAuthorizedForIMEI(clientInfo, imei) {
					err := client.WriteMessage(websocket.TextMessage, message)
					if err != nil {
						colors.PrintError("Error sending WebSocket message to User ID %d: %v", clientInfo.UserID, err)
						// The client is likely disconnected, so we unregister them
						go func(c *websocket.Conn) {
							h.unregister <- c
						}(client)
					}
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// isClientAuthorizedForIMEI checks if client has access to the specific IMEI
func (h *WebSocketHub) isClientAuthorizedForIMEI(clientInfo *ClientInfo, imei string) bool {
	if imei == "" {
		return false // No IMEI specified, can't authorize
	}

	for _, accessibleIMEI := range clientInfo.AccessibleIMEIs {
		if accessibleIMEI == imei {
			return true
		}
	}
	return false
}

// BroadcastGPSUpdate sends GPS data updates to all connected clients
func (h *WebSocketHub) BroadcastGPSUpdate(gpsData *models.GPSData, vehicleName, regNo string) {
	// Check if this is location data or status data
	hasValidCoordinates := false
	if gpsData.Latitude != nil && gpsData.Longitude != nil {
		lat := *gpsData.Latitude
		lng := *gpsData.Longitude
		hasValidCoordinates = lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180 && lat != 0 && lng != 0
	}

	if hasValidCoordinates {
		// This is location data - broadcast as location update
		h.BroadcastLocationUpdate(gpsData, vehicleName, regNo)
	} else {
		// This is status data - broadcast as status update
		h.BroadcastStatusUpdate(gpsData, vehicleName, regNo)
	}
}

// BroadcastLocationUpdate sends location data updates to all connected clients
// Only broadcasts when valid coordinates are present
func (h *WebSocketHub) BroadcastLocationUpdate(gpsData *models.GPSData, vehicleName, regNo string) {
	// Get vehicle information for overspeed checking
	var vehicle models.Vehicle
	vehicleType := ""
	if err := db.GetDB().Where("imei = ?", gpsData.IMEI).First(&vehicle).Error; err == nil {
		vehicleType = string(vehicle.VehicleType)
	}

	// CRITICAL CHECK: Only broadcast if we have valid GPS coordinates
	locationValid := false
	if gpsData.Latitude != nil && gpsData.Longitude != nil {
		lat := *gpsData.Latitude
		lng := *gpsData.Longitude
		locationValid = lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180 && lat != 0 && lng != 0
	}

	// If coordinates are invalid or null, don't broadcast location update
	if !locationValid {
		colors.PrintWarning("📍 Not broadcasting location update for IMEI %s - invalid or null coordinates (lat=%v, lng=%v)",
			gpsData.IMEI, gpsData.Latitude, gpsData.Longitude)
		return
	}

	locationUpdate := LocationUpdate{
		IMEI:          gpsData.IMEI,
		VehicleName:   vehicleName,
		RegNo:         regNo,
		VehicleType:   vehicleType,
		Latitude:      gpsData.Latitude,
		Longitude:     gpsData.Longitude,
		Speed:         gpsData.Speed,
		Course:        gpsData.Course,
		Altitude:      gpsData.Altitude,
		Timestamp:     gpsData.Timestamp.Format("2006-01-02T15:04:05Z"),
		ProtocolName:  gpsData.ProtocolName,
		LocationValid: locationValid,
	}

	message := WebSocketMessage{
		Type:      "location_update",
		Timestamp: gpsData.Timestamp.Format("2006-01-02T15:04:05Z"),
		Data:      locationUpdate,
	}

	if data, err := json.Marshal(message); err == nil {
		h.broadcast <- data
		colors.PrintSuccess("📍 Broadcasted location update for IMEI %s to %d clients (Lat: %.12f, Lng: %.12f)",
			gpsData.IMEI, len(h.clients), *gpsData.Latitude, *gpsData.Longitude)
	} else {
		colors.PrintError("Error marshaling location update: %v", err)
	}
}

// BroadcastStatusUpdate sends status data updates to all connected clients
// Broadcasts device status information regardless of coordinates
func (h *WebSocketHub) BroadcastStatusUpdate(gpsData *models.GPSData, vehicleName, regNo string) {
	// Get vehicle information
	var vehicle models.Vehicle
	vehicleType := ""
	overspeedLimit := 60 // Default overspeed limit
	if err := db.GetDB().Where("imei = ?", gpsData.IMEI).First(&vehicle).Error; err == nil {
		vehicleType = string(vehicle.VehicleType)
		overspeedLimit = vehicle.Overspeed
	}

	// Determine if vehicle is moving based on speed
	isMoving := false
	currentSpeed := 0
	if gpsData.Speed != nil {
		currentSpeed = *gpsData.Speed
		isMoving = currentSpeed > 5 // Consider moving if speed > 5 km/h
	}

	// Calculate data age precisely using GPS timestamp
	dataAge := time.Since(gpsData.Timestamp)
	dataAgeMinutes := dataAge.Minutes()

	// Determine connection status based on data age and GPS availability
	var connectionStatus string
	if dataAgeMinutes <= 5 {
		if isMoving {
			connectionStatus = "running"
		} else if gpsData.Ignition == "ON" {
			connectionStatus = "idle"
		} else {
			connectionStatus = "stopped"
		}
	} else if dataAgeMinutes <= 60 {
		connectionStatus = "inactive"
	} else {
		connectionStatus = "nodata"
	}

	// Check for overspeed condition
	if currentSpeed > overspeedLimit && overspeedLimit > 0 {
		connectionStatus = "overspeed"
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

	// Build device status information
	deviceStatus := &DeviceInfo{
		Activated:     gpsData.DeviceStatus == "ACTIVATED",
		GPSTracking:   gpsData.GPSTracking == "ENABLED",
		OilConnected:  gpsData.OilElectricity == "CONNECTED",
		EngineRunning: gpsData.Ignition == "ON",
		Satellites:    0,
	}
	if gpsData.Satellites != nil {
		deviceStatus.Satellites = *gpsData.Satellites
	}

	// Build alarm information
	var alarmStatus *AlarmInfo
	if gpsData.AlarmActive {
		alarmStatus = &AlarmInfo{
			Active:    gpsData.AlarmActive,
			Type:      gpsData.AlarmType,
			Code:      gpsData.AlarmCode,
			Emergency: gpsData.AlarmType == "Emergency",
			Overspeed: gpsData.AlarmType == "Overspeed" || connectionStatus == "overspeed",
			LowPower:  gpsData.AlarmType == "Low Power",
			Shock:     gpsData.AlarmType == "Shock",
		}
	}

	statusUpdate := StatusUpdate{
		IMEI:         gpsData.IMEI,
		VehicleName:  vehicleName,
		RegNo:        regNo,
		VehicleType:  vehicleType,
		Speed:        gpsData.Speed,
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
	}

	message := WebSocketMessage{
		Type:      "status_update",
		Timestamp: gpsData.Timestamp.Format("2006-01-02T15:04:05Z"),
		Data:      statusUpdate,
	}

	if data, err := json.Marshal(message); err == nil {
		h.broadcast <- data
		colors.PrintSuccess("📊 Broadcasted status update for IMEI %s to %d clients (Status: %s, Speed: %d km/h, Ignition: %s)",
			gpsData.IMEI, len(h.clients), connectionStatus, currentSpeed, gpsData.Ignition)
	} else {
		colors.PrintError("Error marshaling status update: %v", err)
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

// HandleWebSocket handles WebSocket connections with user authentication
func HandleWebSocket(c *gin.Context) {
	// Check for authentication token in query parameters
	token := c.Query("token")
	if token == "" {
		colors.PrintError("WebSocket connection attempted without authentication token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication token required"})
		return
	}

	// Validate user token and get user information
	var user models.User
	if err := db.GetDB().Where("token = ? AND token_exp > ?", token, time.Now()).First(&user).Error; err != nil {
		colors.PrintError("WebSocket connection attempted with invalid token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
		return
	}

	// Get user's accessible vehicles
	var userVehicles []models.UserVehicle
	if err := db.GetDB().Where("user_id = ? AND is_active = ? AND (live_tracking = ? OR all_access = ?)",
		user.ID, true, true, true).Find(&userVehicles).Error; err != nil {
		colors.PrintError("Failed to get user vehicles for WebSocket: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user vehicles"})
		return
	}

	// Extract accessible IMEIs
	var accessibleIMEIs []string
	for _, userVehicle := range userVehicles {
		if !userVehicle.IsExpired() {
			accessibleIMEIs = append(accessibleIMEIs, userVehicle.VehicleID)
		}
	}

	// Upgrade the HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		colors.PrintError("Failed to upgrade to WebSocket: %v", err)
		return
	}

	colors.PrintConnection("🔗", "New WebSocket connection established for User ID %d from %s", user.ID, c.ClientIP())

	// Register the connection with user information
	WSHub.register <- &ClientConnection{
		Conn:   conn,
		UserID: user.ID,
		IMEIs:  accessibleIMEIs,
	}

	// Handle connection in a goroutine
	go func() {
		defer func() {
			WSHub.unregister <- conn
		}()

		// Send initial welcome message with user's accessible vehicles
		welcomeMsg := WebSocketMessage{
			Type:      "welcome",
			Timestamp: time.Now().Format(time.RFC3339),
			Data: map[string]interface{}{
				"user_id":          user.ID,
				"accessible_imeis": accessibleIMEIs,
				"message":          "WebSocket connection established",
			},
		}

		if welcomeData, err := json.Marshal(welcomeMsg); err == nil {
			conn.WriteMessage(websocket.TextMessage, welcomeData)
		}

		// Keep connection alive and handle incoming messages
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					colors.PrintError("WebSocket error for User ID %d: %v", user.ID, err)
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

// BroadcastFullGPSUpdate sends the entire GPSData model to relevant clients.
// This is the primary method for broadcasting live updates.
func (h *WebSocketHub) BroadcastFullGPSUpdate(gpsData *models.GPSData) {
	// The 'type' helps the client distinguish this message from others.
	message := WebSocketMessage{
		Type:      "gps_update",
		Timestamp: time.Now().Format(time.RFC3339),
		Data:      gpsData,
	}

	payload, err := json.Marshal(message)
	if err != nil {
		colors.PrintError("Failed to marshal GPS update message: %v", err)
		return
	}

	// Send the payload to the central broadcast channel.
	// The Run() method will handle authorization and distribution.
	h.broadcast <- payload
}
