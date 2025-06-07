package main

import (
	"log"
	"luna_iot_server/internal/db"
	"luna_iot_server/internal/http"
	"luna_iot_server/pkg/colors"
	"os"

	"github.com/joho/godotenv"
)

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

	// Get port from environment variable or use default
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	// Create and start HTTP server
	server := http.NewServer(port)

	colors.PrintServer("🌐", "Luna IoT HTTP Server starting on port %s", port)
	colors.PrintSubHeader("Available REST API Endpoints")
	colors.PrintEndpoint("GET", "/health", "Health check endpoint")
	colors.PrintEndpoint("GET", "/api/v1/users", "List all users")
	colors.PrintEndpoint("POST", "/api/v1/users", "Create new user")
	colors.PrintEndpoint("GET", "/api/v1/devices", "List all devices")
	colors.PrintEndpoint("POST", "/api/v1/devices", "Register new device")
	colors.PrintEndpoint("GET", "/api/v1/vehicles", "List all vehicles")
	colors.PrintEndpoint("POST", "/api/v1/vehicles", "Register new vehicle")

	if err := server.Start(); err != nil {
		colors.PrintError("Failed to start HTTP server: %v", err)
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
