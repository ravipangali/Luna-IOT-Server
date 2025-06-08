package http

import (
	"encoding/json"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"
	"net/http"
	"sync"

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
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	Speed        *int     `json:"speed"`
	Course       *int     `json:"course"`
	Ignition     string   `json:"ignition"`
	Timestamp    string   `json:"timestamp"`
	ProtocolName string   `json:"protocol_name"`
}

// DeviceStatus represents device connection status
type DeviceStatus struct {
	IMEI       string `json:"imei"`
	Status     string `json:"status"` // "connected", "disconnected"
	LastSeen   string `json:"last_seen"`
	VehicleReg string `json:"vehicle_reg,omitempty"`
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

// BroadcastGPSUpdate sends GPS data to all connected clients
func (h *WebSocketHub) BroadcastGPSUpdate(gpsData *models.GPSData, vehicleName, regNo string) {
	update := GPSUpdate{
		IMEI:         gpsData.IMEI,
		VehicleName:  vehicleName,
		RegNo:        regNo,
		Latitude:     gpsData.Latitude,
		Longitude:    gpsData.Longitude,
		Speed:        gpsData.Speed,
		Course:       gpsData.Course,
		Ignition:     gpsData.Ignition,
		Timestamp:    gpsData.Timestamp.Format("2006-01-02T15:04:05Z"),
		ProtocolName: gpsData.ProtocolName,
	}

	message := WebSocketMessage{
		Type:      "gps_update",
		Timestamp: gpsData.Timestamp.Format("2006-01-02T15:04:05Z"),
		Data:      update,
	}

	if data, err := json.Marshal(message); err == nil {
		h.broadcast <- data
		colors.PrintData("📡", "Broadcasted GPS update for IMEI %s to %d clients", gpsData.IMEI, len(h.clients))
	} else {
		colors.PrintError("Error marshaling GPS update: %v", err)
	}
}

// BroadcastDeviceStatus sends device status updates
func (h *WebSocketHub) BroadcastDeviceStatus(imei, status, vehicleReg string) {
	statusUpdate := DeviceStatus{
		IMEI:       imei,
		Status:     status,
		LastSeen:   "", // Will be set by caller
		VehicleReg: vehicleReg,
	}

	message := WebSocketMessage{
		Type:      "device_status",
		Timestamp: "", // Will be set by caller
		Data:      statusUpdate,
	}

	if data, err := json.Marshal(message); err == nil {
		h.broadcast <- data
		colors.PrintConnection("📡", "Broadcasted device status for IMEI %s: %s", imei, status)
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
