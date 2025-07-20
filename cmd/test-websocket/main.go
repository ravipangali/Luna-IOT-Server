package main

import (
	"flag"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	// Parse command line flags
	serverAddr := flag.String("server", "localhost:8080", "Server address")
	token := flag.String("token", "", "Authentication token")
	flag.Parse()

	if *token == "" {
		log.Fatal("Token is required. Use -token flag")
	}

	// Construct WebSocket URL
	u := url.URL{Scheme: "ws", Host: *serverAddr, Path: "/ws"}
	q := u.Query()
	q.Set("token", *token)
	u.RawQuery = q.Encode()

	log.Printf("Connecting to %s", u.String())

	// Connect to WebSocket
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	log.Println("✅ WebSocket connected successfully")

	// Set up ping/pong handlers
	c.SetPongHandler(func(string) error {
		log.Println("🏓 Received pong from server")
		return nil
	})

	// Start ping timer
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			err := c.WriteMessage(websocket.TextMessage, []byte("ping"))
			if err != nil {
				log.Printf("❌ Failed to send ping: %v", err)
				return
			}
			log.Println("🏓 Sent ping to server")
		}
	}()

	// Listen for messages
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Printf("❌ WebSocket error: %v", err)
			break
		}

		log.Printf("📨 Received message: %s", string(message))
	}

	log.Println("🔌 WebSocket connection closed")
}
