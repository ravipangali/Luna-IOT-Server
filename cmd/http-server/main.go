package main

import (
	"log"
	"os"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/http"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize database connection
	if err := db.Initialize(); err != nil {
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

	log.Printf("Luna IoT HTTP Server starting on port %s", port)
	log.Println("Available endpoints:")
	log.Println("  GET    /health")
	log.Println("  GET    /api/v1/users")
	log.Println("  POST   /api/v1/users")
	log.Println("  GET    /api/v1/devices")
	log.Println("  POST   /api/v1/devices")
	log.Println("  GET    /api/v1/vehicles")
	log.Println("  POST   /api/v1/vehicles")

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
