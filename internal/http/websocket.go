package http

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Global WebSocket hub instance
var WSHub *WebSocketHub

// WebSocket upgrader configuration
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// WebSocketHub manages all WebSocket connections
type WebSocketHub struct {
	clients    map[*websocket.Conn]*ClientInfo
	broadcast  chan []byte
	register   chan *ClientConnection
	unregister chan *websocket.Conn
	mutex      sync.RWMutex
}

// ClientInfo stores information about a connected client
type ClientInfo struct {
	UserID          uint
	AccessibleIMEIs []string
	IsAuthenticated bool
	LastActivity    time.Time
}

// ClientConnection represents a new client connection
type ClientConnection struct {
	Conn   *websocket.Conn
	UserID uint
	IMEIs  []string
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// LocationUpdate represents a location update message
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

// StatusUpdate represents a status update message
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

// GPSUpdate represents a complete GPS update message
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

// DeviceStatus represents a device status update
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

// BatteryInfo represents battery status
type BatteryInfo struct {
	Level    int    `json:"level"`    // 0-100 percentage
	Voltage  int    `json:"voltage"`  // Raw voltage level
	Status   string `json:"status"`   // "Normal", "Low", "Critical"
	Charging bool   `json:"charging"` // Whether charger is connected
}

// SignalInfo represents GSM signal status
type SignalInfo struct {
	Level      int    `json:"level"`      // Raw signal level
	Bars       int    `json:"bars"`       // 0-5 bars
	Status     string `json:"status"`     // "Excellent", "Good", "Fair", "Poor", "No Signal"
	Percentage int    `json:"percentage"` // 0-100 percentage
}

// DeviceInfo represents device status information
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

	// Start connection health monitoring
	go h.monitorConnections()

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

			// Send to authorized clients only with improved error handling
			clientsToRemove := []*websocket.Conn{}
			successfulSends := 0
			totalClients := 0

			for client, clientInfo := range h.clients {
				totalClients++
				if clientInfo.IsAuthenticated && h.isClientAuthorizedForIMEI(clientInfo, imei) {
					// FIXED: Use WriteControl for better error handling and timeouts
					client.SetWriteDeadline(time.Now().Add(10 * time.Second))
					err := client.WriteMessage(websocket.TextMessage, message)

					if err != nil {
						colors.PrintError("Error sending WebSocket message to User ID %d: %v", clientInfo.UserID, err)
						// Mark client for removal
						clientsToRemove = append(clientsToRemove, client)
					} else {
						// Update last activity on successful message send
						clientInfo.LastActivity = time.Now()
						successfulSends++
					}
				}
			}

			colors.PrintDebug("📡 WebSocket broadcast: %d/%d clients received message for IMEI %s",
				successfulSends, totalClients, imei)

			// Remove disconnected clients
			for _, client := range clientsToRemove {
				colors.PrintConnection("📱", "Removing disconnected client for IMEI %s", imei)
				go func(c *websocket.Conn) {
					h.unregister <- c
				}(client)
			}

			h.mutex.RUnlock()
		}
	}
}

// monitorConnections monitors connection health and cleans up stale connections
func (h *WebSocketHub) monitorConnections() {
	ticker := time.NewTicker(30 * time.Second) // FIXED: Check every 30 seconds instead of 2 minutes
	defer ticker.Stop()

	for range ticker.C {
		h.mutex.Lock()
		now := time.Now()
		staleConnections := []*websocket.Conn{}
		activeConnections := 0

		for client, clientInfo := range h.clients {
			// FIXED: More lenient timeout - consider connection stale after 10 minutes instead of 5
			if now.Sub(clientInfo.LastActivity) > 10*time.Minute {
				colors.PrintWarning("Detected stale WebSocket connection for User ID %d (inactive for %v)",
					clientInfo.UserID, now.Sub(clientInfo.LastActivity))
				staleConnections = append(staleConnections, client)
			} else {
				activeConnections++

				// FIXED: Send periodic ping to keep connections alive
				if now.Sub(clientInfo.LastActivity) > 1*time.Minute {
					go func(c *websocket.Conn, uid uint) {
						c.SetWriteDeadline(time.Now().Add(5 * time.Second))
						if err := c.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
							colors.PrintDebug("Failed to send ping to User ID %d: %v", uid, err)
						}
					}(client, clientInfo.UserID)
				}
			}
		}

		// Remove stale connections
		for _, client := range staleConnections {
			colors.PrintConnection("🧹", "Cleaning up stale WebSocket connection")
			delete(h.clients, client)
			client.Close()
		}

		if len(staleConnections) > 0 {
			colors.PrintInfo("Cleaned up %d stale WebSocket connections. Active clients: %d",
				len(staleConnections), activeConnections)
		} else if activeConnections > 0 {
			colors.PrintDebug("WebSocket health check: %d active connections", activeConnections)
		}

		h.mutex.Unlock()
	}
}

// isClientAuthorizedForIMEI checks if client has access to the specific IMEI
func (h *WebSocketHub) isClientAuthorizedForIMEI(clientInfo *ClientInfo, imei string) bool {
	// Check if the client has access to this IMEI
	for _, accessibleIMEI := range clientInfo.AccessibleIMEIs {
		if accessibleIMEI == imei {
			return true
		}
	}
	return false
}

// BroadcastGPSUpdate broadcasts GPS data to all authorized clients
func (h *WebSocketHub) BroadcastGPSUpdate(gpsData *models.GPSData, vehicleName, regNo string) {
	if h == nil {
		return
	}

	// Get vehicle type
	var vehicleType string
	if err := db.GetDB().Model(&models.Vehicle{}).Where("imei = ?", gpsData.IMEI).Select("vehicle_type").Scan(&vehicleType).Error; err != nil {
		vehicleType = "unknown"
	}

	// Create GPS update message
	gpsUpdate := GPSUpdate{
		IMEI:          gpsData.IMEI,
		VehicleName:   vehicleName,
		RegNo:         regNo,
		VehicleType:   vehicleType,
		Latitude:      gpsData.Latitude,
		Longitude:     gpsData.Longitude,
		Speed:         gpsData.Speed,
		Course:        gpsData.Course,
		Altitude:      gpsData.Altitude,
		Ignition:      gpsData.Ignition,
		Timestamp:     gpsData.Timestamp.Format("2006-01-02T15:04:05Z"),
		ProtocolName:  gpsData.ProtocolName,
		IsMoving:      gpsData.Speed != nil && *gpsData.Speed > 0,
		LastSeen:      time.Now().Format("2006-01-02T15:04:05Z"),
		LocationValid: gpsData.IsValidLocation(),
	}

	// Add enhanced status information
	if gpsData.VoltageLevel != nil {
		gpsUpdate.Battery = &BatteryInfo{
			Level:    getVoltagePercentage(*gpsData.VoltageLevel),
			Voltage:  *gpsData.VoltageLevel,
			Status:   getVoltageStatus(*gpsData.VoltageLevel),
			Charging: gpsData.Charger == "CONNECTED",
		}
	}

	if gpsData.GSMSignal != nil {
		gpsUpdate.Signal = &SignalInfo{
			Level:      *gpsData.GSMSignal,
			Bars:       getSignalBars(*gpsData.GSMSignal),
			Status:     getGsmStatus(*gpsData.GSMSignal),
			Percentage: getSignalPercentage(*gpsData.GSMSignal),
		}
	}

	// Add device status
	if gpsData.DeviceStatus != "" {
		satellites := 0
		if gpsData.Satellites != nil {
			satellites = *gpsData.Satellites
		}
		gpsUpdate.DeviceStatus = &DeviceInfo{
			Activated:     gpsData.DeviceStatus == "ACTIVATED",
			GPSTracking:   gpsData.GPSTracking == "ENABLED",
			OilConnected:  gpsData.OilElectricity == "CONNECTED",
			EngineRunning: gpsData.Ignition == "ON",
			Satellites:    satellites,
		}
	}

	// Add alarm status
	if gpsData.AlarmActive {
		gpsUpdate.AlarmStatus = &AlarmInfo{
			Active:    gpsData.AlarmActive,
			Type:      gpsData.AlarmType,
			Code:      gpsData.AlarmCode,
			Emergency: gpsData.AlarmCode == 1,
			Overspeed: gpsData.AlarmCode == 2,
			LowPower:  gpsData.AlarmCode == 3,
			Shock:     gpsData.AlarmCode == 4,
		}
	}

	// Determine connection status
	if gpsData.Speed != nil && *gpsData.Speed > 0 {
		gpsUpdate.ConnectionStatus = "connected"
	} else if gpsData.Ignition == "ON" {
		gpsUpdate.ConnectionStatus = "stopped"
	} else {
		gpsUpdate.ConnectionStatus = "inactive"
	}

	message := WebSocketMessage{
		Type:      "gps_update",
		Timestamp: time.Now().Format("2006-01-02T15:04:05Z"),
		Data:      gpsUpdate,
	}

	if data, err := json.Marshal(message); err == nil {
		h.broadcast <- data
		colors.PrintConnection("📡", "Broadcasted GPS update for IMEI %s: %s (%s)", gpsData.IMEI, vehicleName, regNo)
	}
}

// BroadcastLocationUpdate broadcasts location data to all authorized clients
func (h *WebSocketHub) BroadcastLocationUpdate(gpsData *models.GPSData, vehicleName, regNo string) {
	if h == nil {
		return
	}

	// Get vehicle type
	var vehicleType string
	if err := db.GetDB().Model(&models.Vehicle{}).Where("imei = ?", gpsData.IMEI).Select("vehicle_type").Scan(&vehicleType).Error; err != nil {
		vehicleType = "unknown"
	}

	// Create location update message
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
		LocationValid: gpsData.IsValidLocation(),
	}

	message := WebSocketMessage{
		Type:      "location_update",
		Timestamp: time.Now().Format("2006-01-02T15:04:05Z"),
		Data:      locationUpdate,
	}

	if data, err := json.Marshal(message); err == nil {
		h.broadcast <- data
		colors.PrintConnection("📍", "Broadcasted location update for IMEI %s: %s (%s)", gpsData.IMEI, vehicleName, regNo)
	}
}

// BroadcastStatusUpdate broadcasts status data to all authorized clients
func (h *WebSocketHub) BroadcastStatusUpdate(gpsData *models.GPSData, vehicleName, regNo string) {
	if h == nil {
		return
	}

	// Get vehicle type
	var vehicleType string
	if err := db.GetDB().Model(&models.Vehicle{}).Where("imei = ?", gpsData.IMEI).Select("vehicle_type").Scan(&vehicleType).Error; err != nil {
		vehicleType = "unknown"
	}

	// Create status update message
	statusUpdate := StatusUpdate{
		IMEI:         gpsData.IMEI,
		VehicleName:  vehicleName,
		RegNo:        regNo,
		VehicleType:  vehicleType,
		Speed:        gpsData.Speed,
		Ignition:     gpsData.Ignition,
		Timestamp:    gpsData.Timestamp.Format("2006-01-02T15:04:05Z"),
		ProtocolName: gpsData.ProtocolName,
		IsMoving:     gpsData.Speed != nil && *gpsData.Speed > 0,
		LastSeen:     time.Now().Format("2006-01-02T15:04:05Z"),
	}

	// Add enhanced status information
	if gpsData.VoltageLevel != nil {
		statusUpdate.Battery = &BatteryInfo{
			Level:    getVoltagePercentage(*gpsData.VoltageLevel),
			Voltage:  *gpsData.VoltageLevel,
			Status:   getVoltageStatus(*gpsData.VoltageLevel),
			Charging: gpsData.Charger == "CONNECTED",
		}
	}

	if gpsData.GSMSignal != nil {
		statusUpdate.Signal = &SignalInfo{
			Level:      *gpsData.GSMSignal,
			Bars:       getSignalBars(*gpsData.GSMSignal),
			Status:     getGsmStatus(*gpsData.GSMSignal),
			Percentage: getSignalPercentage(*gpsData.GSMSignal),
		}
	}

	// Add device status
	if gpsData.DeviceStatus != "" {
		satellites := 0
		if gpsData.Satellites != nil {
			satellites = *gpsData.Satellites
		}
		statusUpdate.DeviceStatus = &DeviceInfo{
			Activated:     gpsData.DeviceStatus == "ACTIVATED",
			GPSTracking:   gpsData.GPSTracking == "ENABLED",
			OilConnected:  gpsData.OilElectricity == "CONNECTED",
			EngineRunning: gpsData.Ignition == "ON",
			Satellites:    satellites,
		}
	}

	// Add alarm status
	if gpsData.AlarmActive {
		statusUpdate.AlarmStatus = &AlarmInfo{
			Active:    gpsData.AlarmActive,
			Type:      gpsData.AlarmType,
			Code:      gpsData.AlarmCode,
			Emergency: gpsData.AlarmCode == 1,
			Overspeed: gpsData.AlarmCode == 2,
			LowPower:  gpsData.AlarmCode == 3,
			Shock:     gpsData.AlarmCode == 4,
		}
	}

	// Determine connection status
	if gpsData.Speed != nil && *gpsData.Speed > 0 {
		statusUpdate.ConnectionStatus = "connected"
	} else if gpsData.Ignition == "ON" {
		statusUpdate.ConnectionStatus = "stopped"
	} else {
		statusUpdate.ConnectionStatus = "inactive"
	}

	message := WebSocketMessage{
		Type:      "status_update",
		Timestamp: time.Now().Format("2006-01-02T15:04:05Z"),
		Data:      statusUpdate,
	}

	if data, err := json.Marshal(message); err == nil {
		h.broadcast <- data
		colors.PrintConnection("📊", "Broadcasted status update for IMEI %s: %s (%s)", gpsData.IMEI, vehicleName, regNo)
	}
}

// BroadcastDeviceStatus broadcasts device status to all authorized clients
func (h *WebSocketHub) BroadcastDeviceStatus(imei, status, vehicleReg string) {
	if h == nil {
		return
	}

	// Get vehicle info
	var vehicleName, vehicleType string
	type VehicleInfo struct {
		Name        string
		VehicleType string
	}
	var vehicleInfo VehicleInfo
	if err := db.GetDB().Model(&models.Vehicle{}).Where("imei = ?", imei).Select("name, vehicle_type").Scan(&vehicleInfo).Error; err != nil {
		vehicleName = "Unknown Vehicle"
		vehicleType = "unknown"
	} else {
		vehicleName = vehicleInfo.Name
		vehicleType = vehicleInfo.VehicleType
	}

	// Create battery and signal info (placeholder values)
	battery := &BatteryInfo{
		Level:    75,
		Voltage:  4,
		Status:   "Normal",
		Charging: false,
	}

	signal := &SignalInfo{
		Level:      4,
		Bars:       4,
		Status:     "Good",
		Percentage: 80,
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
	if err := db.GetDB().Where("token = ?", token).First(&user).Error; err != nil {
		colors.PrintError("WebSocket connection attempted with invalid token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Check if token is valid (exists)
	if !user.IsTokenValid() {
		colors.PrintError("WebSocket connection attempted with expired token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
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

	colors.PrintConnection("🔗", "User ID %d has access to %d vehicles: %v", user.ID, len(accessibleIMEIs), accessibleIMEIs)

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
			// Ensure proper cleanup
			colors.PrintConnection("📱", "WebSocket cleanup for User ID %d", user.ID)
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
			if err := conn.WriteMessage(websocket.TextMessage, welcomeData); err != nil {
				colors.PrintError("Failed to send welcome message to User ID %d: %v", user.ID, err)
			}
		}

		// Set up ping/pong for connection health monitoring
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		// Keep connection alive and handle incoming messages
		for {
			// Set read deadline to detect stale connections
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) {
					colors.PrintError("WebSocket error for User ID %d: %v", user.ID, err)
				} else {
					colors.PrintConnection("📱", "WebSocket closed normally for User ID %d", user.ID)
				}
				break
			}

			// Handle ping messages
			if string(message) == "ping" {
				if err := conn.WriteMessage(websocket.TextMessage, []byte("pong")); err != nil {
					colors.PrintError("Failed to send pong to User ID %d: %v", user.ID, err)
					break
				}
			}

			// Update last activity
			WSHub.mutex.Lock()
			if clientInfo, exists := WSHub.clients[conn]; exists {
				clientInfo.LastActivity = time.Now()
			}
			WSHub.mutex.Unlock()
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

// getVoltageStatus returns the voltage status string
func getVoltageStatus(level int) string {
	if level <= 1 {
		return "Critical"
	} else if level <= 3 {
		return "Low"
	} else {
		return "Normal"
	}
}

// getSignalBars converts signal level to bars (0-5)
func getSignalBars(level int) int {
	if level <= 0 {
		return 0
	} else if level <= 2 {
		return 1
	} else if level <= 4 {
		return 2
	} else if level <= 6 {
		return 3
	} else if level <= 8 {
		return 4
	} else {
		return 5
	}
}

// getSignalPercentage converts signal level to percentage (0-100)
func getSignalPercentage(level int) int {
	if level <= 0 {
		return 0
	}
	if level >= 10 {
		return 100
	}
	return (level * 100) / 10
}

// getGsmStatus returns the GSM signal status string
func getGsmStatus(level int) string {
	if level <= 0 {
		return "No Signal"
	} else if level <= 2 {
		return "Poor"
	} else if level <= 4 {
		return "Fair"
	} else if level <= 6 {
		return "Good"
	} else {
		return "Excellent"
	}
}

// BroadcastFullGPSUpdate broadcasts complete GPS data
func (h *WebSocketHub) BroadcastFullGPSUpdate(gpsData *models.GPSData) {
	if h == nil {
		return
	}

	// Get vehicle info
	var vehicle models.Vehicle
	if err := db.GetDB().Where("imei = ?", gpsData.IMEI).First(&vehicle).Error; err != nil {
		colors.PrintWarning("Vehicle not found for IMEI %s", gpsData.IMEI)
		return
	}

	h.BroadcastGPSUpdate(gpsData, vehicle.Name, vehicle.RegNo)
}

// BroadcastLogoutNotification sends a logout notification to all clients of a specific user
func (h *WebSocketHub) BroadcastLogoutNotification(userID uint, reason string) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	logoutMessage := WebSocketMessage{
		Type:      "logout_notification",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"reason":  reason,
			"user_id": userID,
		},
	}

	messageBytes, err := json.Marshal(logoutMessage)
	if err != nil {
		colors.PrintError("Failed to marshal logout notification: %v", err)
		return
	}

	// Send logout notification to all clients of this user
	for conn, clientInfo := range h.clients {
		if clientInfo.UserID == userID {
			colors.PrintInfo("Sending logout notification to client for user %d", userID)
			err := conn.WriteMessage(websocket.TextMessage, messageBytes)
			if err != nil {
				colors.PrintError("Failed to send logout notification: %v", err)
				// The client is likely disconnected, so we unregister them
				go func(c *websocket.Conn) {
					h.unregister <- c
				}(conn)
			}
		}
	}
}

// BroadcastLogoutNotificationGlobal sends a logout notification using the global WSHub
func BroadcastLogoutNotificationGlobal(userID uint, reason string) {
	if WSHub != nil {
		WSHub.BroadcastLogoutNotification(userID, reason)
	} else {
		colors.PrintWarning("WebSocket hub not initialized - skipping logout notification")
	}
}
