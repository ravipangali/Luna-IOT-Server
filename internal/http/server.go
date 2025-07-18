package http

import (
	"luna_iot_server/internal/http/controllers"
	"luna_iot_server/pkg/colors"
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
	return s.router.Run(":" + s.port)
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
