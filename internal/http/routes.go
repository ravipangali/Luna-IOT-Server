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
	deviceModelController := controllers.NewDeviceModelController()
	vehicleController := controllers.NewVehicleController()
	userVehicleController := controllers.NewUserVehicleController()
	gpsController := controllers.NewGPSController()
	userTrackingController := controllers.NewUserTrackingController()
	dashboardController := controllers.NewDashboardController()

	// Use shared control controller if provided, otherwise create new one
	var controlController *controllers.ControlController
	if sharedControlController != nil {
		controlController = sharedControlController
	} else {
		controlController = controllers.NewControlController()
	}

	// Initialize user-based controllers
	userControlController := controllers.NewUserControlController(controlController)
	userGPSController := controllers.NewUserGPSController()

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
			auth.POST("/send-otp", authController.SendOTP)
		}

		// Protected authentication routes (require auth)
		authProtected := v1.Group("/auth")
		authProtected.Use(middleware.AuthMiddleware())
		{
			authProtected.POST("/logout", authController.Logout)
			authProtected.GET("/me", authController.Me)
			authProtected.POST("/refresh", authController.RefreshToken)
			authProtected.GET("/delete-account", authController.DeleteAccount)
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

		// Device Model routes (authenticated users only)
		deviceModels := v1.Group("/device-models")
		deviceModels.Use(middleware.AuthMiddleware())
		{
			deviceModels.GET("", deviceModelController.GetDeviceModels)
			deviceModels.GET("/:id", deviceModelController.GetDeviceModel)
			deviceModels.POST("", middleware.AdminOnlyMiddleware(), deviceModelController.CreateDeviceModel)       // Admin only
			deviceModels.PUT("/:id", middleware.AdminOnlyMiddleware(), deviceModelController.UpdateDeviceModel)    // Admin only
			deviceModels.DELETE("/:id", middleware.AdminOnlyMiddleware(), deviceModelController.DeleteDeviceModel) // Admin only
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

		// Customer vehicle routes (authenticated users can manage their own vehicles)
		customerVehicles := v1.Group("/my-vehicles")
		customerVehicles.Use(middleware.AuthMiddleware())
		{
			customerVehicles.GET("", vehicleController.GetMyVehicles)                              // Get user's own vehicles
			customerVehicles.GET("/:imei", vehicleController.GetMyVehicle)                         // Get user's specific vehicle
			customerVehicles.POST("", vehicleController.CreateMyVehicle)                           // Create vehicle for current user
			customerVehicles.PUT("/:imei", vehicleController.UpdateMyVehicle)                      // Update user's own vehicle
			customerVehicles.DELETE("/:imei", vehicleController.DeleteMyVehicle)                   // Delete user's own vehicle
			customerVehicles.GET("/:imei/share", vehicleController.GetVehicleShares)               // Get vehicle sharing info
			customerVehicles.POST("/:imei/share", vehicleController.ShareMyVehicle)                // Share vehicle with others
			customerVehicles.DELETE("/:imei/share/:shareId", vehicleController.RevokeVehicleShare) // Revoke vehicle share
		}

		// ===========================================
		// NEW: USER-BASED TRACKING ROUTES (CLIENT APP)
		// ===========================================
		userTracking := v1.Group("/my-tracking")
		userTracking.Use(middleware.AuthMiddleware())
		{
			// Get tracking data for all user's vehicles
			userTracking.GET("", userTrackingController.GetMyVehiclesTracking)

			// Get detailed tracking for a specific vehicle
			userTracking.GET("/:imei", userTrackingController.GetMyVehicleTracking)

			// Get only location data for a specific vehicle
			userTracking.GET("/:imei/location", userTrackingController.GetMyVehicleLocation)

			// Get only status data for a specific vehicle
			userTracking.GET("/:imei/status", userTrackingController.GetMyVehicleStatus)

			// Get GPS history for a specific vehicle
			userTracking.GET("/:imei/history", userTrackingController.GetMyVehicleHistory)

			// Get route data for a specific vehicle
			userTracking.GET("/:imei/route", userTrackingController.GetMyVehicleRoute)

			// Get reports for a specific vehicle
			userTracking.GET("/:imei/reports", userTrackingController.GetMyVehicleReports)
		}

		// ===========================================
		// NEW: USER-BASED CONTROL ROUTES (CLIENT APP)
		// ===========================================
		userControl := v1.Group("/my-control")
		userControl.Use(middleware.AuthMiddleware())
		{
			// Cut oil and electricity for user's vehicle
			userControl.POST("/:imei/cut-oil", userControlController.CutOilAndElectricity)

			// Connect oil and electricity for user's vehicle
			userControl.POST("/:imei/connect-oil", userControlController.ConnectOilAndElectricity)

			// Get location for user's vehicle
			userControl.POST("/:imei/get-location", userControlController.GetVehicleLocation)

			// Get user's active devices
			userControl.GET("/active-devices", userControlController.GetUserActiveDevices)
		}

		// ===========================================
		// NEW: USER-BASED GPS ROUTES (CLIENT APP)
		// ===========================================
		userGPS := v1.Group("/my-gps")
		userGPS.Use(middleware.AuthMiddleware())
		{
			// Get GPS data for all user's vehicles
			userGPS.GET("", userGPSController.GetUserVehicleTracking)

			// Get location data for a specific vehicle
			userGPS.GET("/:imei/location", userGPSController.GetUserVehicleLocation)

			// Get status data for a specific vehicle
			userGPS.GET("/:imei/status", userGPSController.GetUserVehicleStatus)

			// Get GPS history with pagination
			userGPS.GET("/:imei/history", userGPSController.GetUserVehicleHistory)

			// Get GPS route data
			userGPS.GET("/:imei/route", userGPSController.GetUserVehicleRoute)

			// Get GPS reports
			userGPS.GET("/:imei/report", userGPSController.GetUserVehicleReport)
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

		// User-Vehicle relationship routes (admin only for assignment, users can view their own access)
		userVehicles := v1.Group("/user-vehicles")
		userVehicles.Use(middleware.AuthMiddleware())
		{
			// Admin only routes for managing assignments
			userVehicles.POST("/assign", middleware.AdminOnlyMiddleware(), userVehicleController.AssignVehicleToUser)
			userVehicles.POST("/bulk-assign", middleware.AdminOnlyMiddleware(), userVehicleController.BulkAssignVehiclesToUser)
			userVehicles.PUT("/:id/permissions", middleware.AdminOnlyMiddleware(), userVehicleController.UpdateVehiclePermissions)
			userVehicles.PUT("/vehicle/:vehicle_id/set-main-user", middleware.AdminOnlyMiddleware(), userVehicleController.SetMainUser)
			userVehicles.DELETE("/:id", middleware.AdminOnlyMiddleware(), userVehicleController.RevokeVehicleAccess)

			// View routes (users can view their own access, admins can view all)
			userVehicles.GET("/user/:user_id", userVehicleController.GetUserVehicleAccess)       // Will be restricted by middleware
			userVehicles.GET("/vehicle/:vehicle_id", userVehicleController.GetVehicleUserAccess) // Will be restricted by middleware
		}

		// Dashboard routes
		dashboard := v1.Group("/dashboard")
		dashboard.Use(middleware.AuthMiddleware(), middleware.AdminOnlyMiddleware())
		{
			dashboard.GET("/stats", dashboardController.GetDashboardStats)
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
			"client_api": gin.H{
				"my_vehicles": "/api/v1/my-vehicles",
				"my_tracking": "/api/v1/my-tracking",
				"my_control":  "/api/v1/my-control",
				"my_gps":      "/api/v1/my-gps",
			},
		})
	})
}
