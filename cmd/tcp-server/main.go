package main

import (
	"log"
	"luna_iot_server/config"
	"luna_iot_server/internal/db"
	"luna_iot_server/internal/http/controllers"
	"luna_iot_server/internal/tcp"
	"luna_iot_server/pkg/colors"
	"os"

	"github.com/joho/godotenv"
)

// Global control controller instance to track active connections
var controlController *controllers.ControlController

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

	// Initialize Firebase
	if err := config.InitializeFirebase(); err != nil {
		colors.PrintWarning("Firebase initialization failed: %v", err)
		colors.PrintWarning("Push notifications will be disabled")
	} else {
		colors.PrintSuccess("Firebase initialized successfully")
	}

	// Initialize global control controller
	controlController = controllers.NewControlController()

	// Get TCP port from environment variable or use default
	port := os.Getenv("TCP_PORT")
	if port == "" {
		port = "5000"
	}

	// Use the enhanced TCP server from internal/tcp package
	colors.PrintServer("📡", "Starting Enhanced GT06 TCP Server on port %s", port)
	colors.PrintConnection("📶", "Features: GPS validation, device timeout monitoring, enhanced WebSocket broadcasting")
	colors.PrintData("💾", "Database connectivity enabled - GPS data will be saved")
	colors.PrintControl("Oil/Electricity control system enabled - Ready for commands")

	// Create and start the enhanced TCP server
	tcpServer := tcp.NewServerWithController(port, controlController)
	if err := tcpServer.Start(); err != nil {
		colors.PrintError("Failed to start TCP server: %v", err)
		log.Fatalf("Failed to start TCP server: %v", err)
	}
}
