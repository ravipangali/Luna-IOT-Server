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
	"luna_iot_server/internal/tcp"

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

	// Get ports from environment variables or use defaults
	tcpPort := os.Getenv("TCP_PORT")
	if tcpPort == "" {
		tcpPort = "5000"
	}

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	fmt.Println("🚀 Luna IoT Server Starting...")
	fmt.Printf("📡 TCP Server will listen on port %s (for IoT devices)\n", tcpPort)
	fmt.Printf("🌐 HTTP Server will listen on port %s (for API access)\n", httpPort)
	fmt.Println("💾 Database connection established")
	fmt.Println("⚡ Control system enabled - Oil/Electricity control available")

	// Create a wait group to manage both servers
	var wg sync.WaitGroup
	errorChan := make(chan error, 2)

	// Start TCP Server in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		tcpServer := tcp.NewServer(tcpPort)
		log.Printf("Starting TCP Server on port %s", tcpPort)
		if err := tcpServer.Start(); err != nil {
			errorChan <- fmt.Errorf("TCP server error: %v", err)
		}
	}()

	// Start HTTP Server in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		httpServer := http.NewServer(httpPort)
		log.Printf("Starting HTTP Server on port %s", httpPort)
		log.Println("Available HTTP endpoints:")
		log.Println("  GET    /health")
		log.Println("  GET    /api/v1/users")
		log.Println("  POST   /api/v1/users")
		log.Println("  GET    /api/v1/devices")
		log.Println("  POST   /api/v1/devices")
		log.Println("  GET    /api/v1/vehicles")
		log.Println("  POST   /api/v1/vehicles")
		log.Println("  POST   /api/v1/control/oil")
		log.Println("  POST   /api/v1/control/electricity")
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
		log.Printf("Server error: %v", err)
		return
	case <-quit:
		fmt.Println("\n🛑 Shutting down Luna IoT Server...")
		log.Println("Graceful shutdown initiated...")
		return
	}
}
