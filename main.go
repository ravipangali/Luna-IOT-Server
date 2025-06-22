package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/http"
	"luna_iot_server/internal/http/controllers"
	"luna_iot_server/internal/tcp"
	"luna_iot_server/pkg/colors"

	"github.com/joho/godotenv"
)

func main() {
	// Print attractive banner
	colors.PrintBanner()

	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		colors.PrintWarning("No .env file found, using system environment variables")
	} else {
		colors.PrintSuccess("Environment configuration loaded from .env file")
	}

	// Initialize database connection
	colors.PrintInfo("Initializing database connection...")
	if err := db.Initialize(); err != nil {
		colors.PrintError("Failed to initialize database: %v", err)
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	// Get ports from environment variables or use defaults
	tcpPort := os.Getenv("TCP_PORT")
	if tcpPort == "" {
		tcpPort = "5000"
	}

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	// Create a shared control controller instance that both servers will use
	sharedControlController := controllers.NewControlController()
	colors.PrintSuccess("Shared control controller initialized")

	// Print server startup information
	colors.PrintHeader("LUNA IOT SERVER INITIALIZATION")
	colors.PrintServer("📡", "TCP Server configured for port %s (IoT Device Connections)", tcpPort)
	colors.PrintServer("🌐", "HTTP Server configured for port %s (REST API Access)", httpPort)
	colors.PrintSuccess("Database connection established successfully")
	colors.PrintControl("Oil & Electricity control system enabled")

	// Create a wait group to manage both servers
	var wg sync.WaitGroup
	errorChan := make(chan error, 2)

	// Start TCP Server in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		tcpServer := tcp.NewServerWithController(tcpPort, sharedControlController)
		colors.PrintInfo("Starting TCP Server for IoT device connections...")
		if err := tcpServer.Start(); err != nil {
			errorChan <- fmt.Errorf("TCP server error: %v", err)
		}
	}()

	// Start HTTP Server in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		httpServer := http.NewServerWithController(httpPort, sharedControlController)
		colors.PrintInfo("Starting HTTP Server for REST API...")

		colors.PrintSubHeader("Available REST API Endpoints")
		colors.PrintEndpoint("GET", "/health", "Health check endpoint")
		colors.PrintEndpoint("POST", "/api/v1/auth/login", "User authentication")
		colors.PrintEndpoint("POST", "/api/v1/auth/register", "User registration")
		colors.PrintEndpoint("GET", "/api/v1/auth/me", "Get user profile")

		colors.PrintSubHeader("User-Based Client API Endpoints")
		colors.PrintEndpoint("GET", "/api/v1/my-vehicles", "Get user's vehicles")
		colors.PrintEndpoint("GET", "/api/v1/my-tracking", "Get user's vehicles tracking")
		colors.PrintEndpoint("GET", "/api/v1/my-tracking/:imei", "Get specific vehicle tracking")
		colors.PrintEndpoint("GET", "/api/v1/my-tracking/:imei/location", "Get vehicle location")
		colors.PrintEndpoint("GET", "/api/v1/my-tracking/:imei/status", "Get vehicle status")
		colors.PrintEndpoint("GET", "/api/v1/my-tracking/:imei/history", "Get vehicle history")
		colors.PrintEndpoint("GET", "/api/v1/my-tracking/:imei/route", "Get vehicle route")
		colors.PrintEndpoint("GET", "/api/v1/my-tracking/:imei/reports", "Get vehicle reports")
		colors.PrintEndpoint("POST", "/api/v1/my-control/:imei/cut-oil", "Cut oil & electricity")
		colors.PrintEndpoint("POST", "/api/v1/my-control/:imei/connect-oil", "Connect oil & electricity")
		colors.PrintEndpoint("POST", "/api/v1/my-control/:imei/get-location", "Request device location")
		colors.PrintEndpoint("GET", "/api/v1/my-control/active-devices", "Get user's active devices")
		colors.PrintEndpoint("GET", "/api/v1/my-gps", "Get user's GPS data")
		colors.PrintEndpoint("GET", "/api/v1/my-gps/:imei/location", "Get vehicle GPS location")
		colors.PrintEndpoint("GET", "/api/v1/my-gps/:imei/status", "Get vehicle GPS status")
		colors.PrintEndpoint("GET", "/api/v1/my-gps/:imei/history", "Get vehicle GPS history")
		colors.PrintEndpoint("GET", "/api/v1/my-gps/:imei/route", "Get vehicle GPS route")
		colors.PrintEndpoint("GET", "/api/v1/my-gps/:imei/report", "Get vehicle GPS report")

		colors.PrintSubHeader("Admin API Endpoints")
		colors.PrintEndpoint("GET", "/api/v1/users", "List all users (Admin)")
		colors.PrintEndpoint("POST", "/api/v1/users", "Create new user (Admin)")
		colors.PrintEndpoint("GET", "/api/v1/devices", "List all devices (Admin)")
		colors.PrintEndpoint("POST", "/api/v1/devices", "Register new device (Admin)")
		colors.PrintEndpoint("GET", "/api/v1/vehicles", "List all vehicles (Admin)")
		colors.PrintEndpoint("POST", "/api/v1/vehicles", "Register new vehicle (Admin)")
		colors.PrintEndpoint("GET", "/api/v1/gps", "Get all GPS tracking data (Admin)")
		colors.PrintEndpoint("POST", "/api/v1/control/cut-oil", "Cut oil & electricity (Admin)")
		colors.PrintEndpoint("POST", "/api/v1/control/connect-oil", "Connect oil & electricity (Admin)")
		colors.PrintEndpoint("POST", "/api/v1/control/get-location", "Get device location (Admin)")
		colors.PrintEndpoint("GET", "/api/v1/control/active-devices", "List active devices (Admin)")

		colors.PrintSubHeader("WebSocket Connection")
		colors.PrintEndpoint("GET", "/ws?token=<auth_token>", "Real-time user-based updates")

		if err := httpServer.Start(); err != nil {
			errorChan <- fmt.Errorf("HTTP server error: %v", err)
		}
	}()

	// Set up graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for either an error or shutdown signal
	select {
	case err := <-errorChan:
		colors.PrintError("Server startup failed: %v", err)
		return
	case <-quit:
		colors.PrintShutdown()
		return
	}
}
