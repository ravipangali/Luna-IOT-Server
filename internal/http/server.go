package http

import (
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

	// Setup routes
	SetupRoutes(router)

	return &Server{
		router: router,
		port:   port,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	colors.PrintServer("🌐", "HTTP REST API Server starting on port %s", s.port)
	return s.router.Run(":" + s.port)
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
