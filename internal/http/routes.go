package http

import (
	"luna_iot_server/internal/http/controllers"
	"luna_iot_server/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all API routes
func SetupRoutes(router *gin.Engine) {
	SetupRoutesWithControlController(router, nil)
}

// SetupRoutesWithControlController configures all API routes with a shared control controller
func SetupRoutesWithControlController(router *gin.Engine, sharedControlController *controllers.ControlController) {
	// Initialize controllers
	authController := controllers.NewAuthController()
	userController := controllers.NewUserController()
	deviceController := controllers.NewDeviceController()
	vehicleController := controllers.NewVehicleController()
	gpsController := controllers.NewGPSController()

	// Use shared control controller if provided, otherwise create new one
	var controlController *controllers.ControlController
	if sharedControlController != nil {
		controlController = sharedControlController
	} else {
		controlController = controllers.NewControlController()
	}

	// WebSocket endpoint for real-time data (no auth required for now)
	router.GET("/ws", HandleWebSocket)

	// API version 1
	v1 := router.Group("/api/v1")
	{
		// Public authentication routes (no middleware)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authController.Login)
			auth.POST("/register", authController.Register)
		}

		// Protected authentication routes (require auth)
		authProtected := v1.Group("/auth")
		authProtected.Use(middleware.AuthMiddleware())
		{
			authProtected.POST("/logout", authController.Logout)
			authProtected.GET("/me", authController.Me)
			authProtected.POST("/refresh", authController.RefreshToken)
		}

		// User routes (admin only for most operations)
		users := v1.Group("/users")
		users.Use(middleware.AuthMiddleware()) // All user routes require authentication
		{
			users.GET("", middleware.AdminOnlyMiddleware(), userController.GetUsers)
			users.GET("/:id", userController.GetUser) // Users can view their own profile
			users.POST("", middleware.AdminOnlyMiddleware(), userController.CreateUser)
			users.PUT("/:id", userController.UpdateUser) // Users can update their own profile
			users.DELETE("/:id", middleware.AdminOnlyMiddleware(), userController.DeleteUser)

			// User image routes
			users.GET("/:id/image", userController.GetUserImage)
			users.DELETE("/:id/image", userController.DeleteUserImage)
		}

		// Device routes (authenticated users only)
		devices := v1.Group("/devices")
		devices.Use(middleware.AuthMiddleware())
		{
			devices.GET("", deviceController.GetDevices)
			devices.GET("/:id", deviceController.GetDevice)
			devices.GET("/imei/:imei", deviceController.GetDeviceByIMEI)
			devices.POST("", middleware.AdminOnlyMiddleware(), deviceController.CreateDevice)       // Admin only
			devices.PUT("/:id", middleware.AdminOnlyMiddleware(), deviceController.UpdateDevice)    // Admin only
			devices.DELETE("/:id", middleware.AdminOnlyMiddleware(), deviceController.DeleteDevice) // Admin only
		}

		// Vehicle routes (authenticated users only)
		vehicles := v1.Group("/vehicles")
		vehicles.Use(middleware.AuthMiddleware())
		{
			vehicles.GET("", vehicleController.GetVehicles)
			vehicles.GET("/:imei", vehicleController.GetVehicle)
			vehicles.GET("/reg/:reg_no", vehicleController.GetVehicleByRegNo)
			vehicles.GET("/type/:type", vehicleController.GetVehiclesByType)
			vehicles.POST("", middleware.AdminOnlyMiddleware(), vehicleController.CreateVehicle)         // Admin only
			vehicles.PUT("/:imei", middleware.AdminOnlyMiddleware(), vehicleController.UpdateVehicle)    // Admin only
			vehicles.DELETE("/:imei", middleware.AdminOnlyMiddleware(), vehicleController.DeleteVehicle) // Admin only
		}

		// GPS tracking routes (authenticated users only)
		gps := v1.Group("/gps")
		gps.Use(middleware.AuthMiddleware())
		{
			gps.GET("", gpsController.GetGPSData)
			gps.GET("/latest", gpsController.GetLatestGPSData)
			gps.GET("/latest-valid", gpsController.GetLatestValidGPSData)

			// NEW: Separate location and status data endpoints
			gps.GET("/latest-location", gpsController.GetLatestLocationData)
			gps.GET("/latest-status", gpsController.GetLatestStatusData)

			gps.GET("/:imei", gpsController.GetGPSDataByIMEI)
			gps.GET("/:imei/latest", gpsController.GetLatestGPSDataByIMEI)
			gps.GET("/:imei/latest-valid", gpsController.GetLatestValidGPSDataByIMEI)

			// NEW: Individual device location and status endpoints
			gps.GET("/:imei/location", gpsController.GetLocationDataByIMEI)
			gps.GET("/:imei/status", gpsController.GetStatusDataByIMEI)

			// NEW: Combined endpoint for individual tracking with historical fallback
			gps.GET("/:imei/individual-tracking", gpsController.GetIndividualTrackingData)

			gps.GET("/:imei/route", gpsController.GetGPSRoute)
			gps.DELETE("/:id", middleware.AdminOnlyMiddleware(), gpsController.DeleteGPSData) // Admin only
		}

		// Control routes for oil and electricity (authenticated users only)
		control := v1.Group("/control")
		control.Use(middleware.AuthMiddleware())
		{
			control.POST("/cut-oil", controlController.CutOilAndElectricity)
			control.POST("/connect-oil", controlController.ConnectOilAndElectricity)
			control.POST("/get-location", controlController.GetLocation)
			control.GET("/active-devices", controlController.GetActiveDevices)
			control.POST("/quick-cut/:id", controlController.QuickCutOil)
			control.POST("/quick-connect/:id", controlController.QuickConnectOil)
			control.POST("/quick-cut-imei/:imei", controlController.QuickCutOil)
			control.POST("/quick-connect-imei/:imei", controlController.QuickConnectOil)
		}
	}

	// Health check endpoint (public)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"message":   "Luna IoT Server is running",
			"websocket": "/ws",
			"api":       "/api/v1",
			"auth": gin.H{
				"login":    "/api/v1/auth/login",
				"register": "/api/v1/auth/register",
				"me":       "/api/v1/auth/me",
				"logout":   "/api/v1/auth/logout",
			},
		})
	})
}
