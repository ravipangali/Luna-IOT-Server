package http

import (
	"crypto/tls"
	"luna_iot_server/internal/http/controllers"
	"luna_iot_server/pkg/colors"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// Server represents the HTTP server
type Server struct {
	router *gin.Engine
	port   string
}

// NewServer creates a new HTTP server instance
func NewServer(port string) *Server {
	// Set Gin to release mode to reduce debug output
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	router := gin.Default()

	// Add middleware conditionally
	// Only add logger middleware if LOG_HTTP is set to true
	if os.Getenv("LOG_HTTP") == "true" {
		router.Use(gin.Logger())
	}
	router.Use(gin.Recovery())
	router.Use(CORSMiddleware())

	// Initialize WebSocket hub
	InitializeWebSocket()

	// Setup routes
	SetupRoutes(router)

	// Add global OPTIONS handler for CORS preflight
	router.OPTIONS("/*path", func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		c.AbortWithStatus(204)
	})

	return &Server{
		router: router,
		port:   port,
	}
}

// NewServerWithController creates a new HTTP server instance with a shared control controller
func NewServerWithController(port string, sharedController *controllers.ControlController) *Server {
	// Set Gin to release mode to reduce debug output
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	router := gin.Default()

	// Add middleware conditionally
	// Only add logger middleware if LOG_HTTP is set to true
	if os.Getenv("LOG_HTTP") == "true" {
		router.Use(gin.Logger())
	}
	router.Use(gin.Recovery())
	router.Use(CORSMiddleware())

	// Initialize WebSocket hub
	InitializeWebSocket()

	// Setup routes with shared control controller
	SetupRoutesWithControlController(router, sharedController)

	return &Server{
		router: router,
		port:   port,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	colors.PrintServer("🌐", "HTTP REST API Server starting on port %s", s.port)
	colors.PrintServer("🔗", "WebSocket endpoint available at /ws for real-time data")

	// Check if HTTPS is enabled
	if os.Getenv("HTTPS_ENABLED") == "true" {
		return s.startHTTPS()
	}

	return s.router.Run(":" + s.port)
}

// startHTTPS starts the server with HTTPS
func (s *Server) startHTTPS() error {
	certFile := os.Getenv("SSL_CERT_FILE")
	keyFile := os.Getenv("SSL_KEY_FILE")

	if certFile == "" || keyFile == "" {
		colors.PrintError("SSL_CERT_FILE and SSL_KEY_FILE environment variables must be set for HTTPS")
		colors.PrintWarning("Falling back to HTTP mode")
		return s.router.Run(":" + s.port)
	}

	// Check if certificate files exist
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		colors.PrintError("SSL certificate file not found: %s", certFile)
		colors.PrintWarning("Falling back to HTTP mode")
		return s.router.Run(":" + s.port)
	}

	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		colors.PrintError("SSL key file not found: %s", keyFile)
		colors.PrintWarning("Falling back to HTTP mode")
		return s.router.Run(":" + s.port)
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	// Create HTTP server with TLS config
	server := &http.Server{
		Addr:      ":" + s.port,
		Handler:   s.router,
		TLSConfig: tlsConfig,
	}

	colors.PrintServer("🔒", "HTTPS server starting on port %s", s.port)
	colors.PrintServer("📜", "Using SSL certificate: %s", certFile)
	colors.PrintServer("🔑", "Using SSL key: %s", keyFile)

	return server.ListenAndServeTLS(certFile, keyFile)
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow all origins for development
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
