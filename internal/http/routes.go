package http

import (
	"luna_iot_server/internal/http/controllers"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all API routes
func SetupRoutes(router *gin.Engine) {
	SetupRoutesWithControlController(router, nil)
}

// SetupRoutesWithControlController configures all API routes with a shared control controller
func SetupRoutesWithControlController(router *gin.Engine, sharedControlController *controllers.ControlController) {
	// Initialize controllers
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

	// WebSocket endpoint for real-time data
	router.GET("/ws", HandleWebSocket)

	// API version 1
	v1 := router.Group("/api/v1")
	{
		// User routes
		users := v1.Group("/users")
		{
			users.GET("", userController.GetUsers)
			users.GET("/:id", userController.GetUser)
			users.POST("", userController.CreateUser)
			users.PUT("/:id", userController.UpdateUser)
			users.DELETE("/:id", userController.DeleteUser)
		}

		// Device routes
		devices := v1.Group("/devices")
		{
			devices.GET("", deviceController.GetDevices)
			devices.GET("/:id", deviceController.GetDevice)
			devices.GET("/imei/:imei", deviceController.GetDeviceByIMEI)
			devices.POST("", deviceController.CreateDevice)
			devices.PUT("/:id", deviceController.UpdateDevice)
			devices.DELETE("/:id", deviceController.DeleteDevice)
		}

		// Vehicle routes
		vehicles := v1.Group("/vehicles")
		{
			vehicles.GET("", vehicleController.GetVehicles)
			vehicles.GET("/:imei", vehicleController.GetVehicle)
			vehicles.GET("/reg/:reg_no", vehicleController.GetVehicleByRegNo)
			vehicles.GET("/type/:type", vehicleController.GetVehiclesByType)
			vehicles.POST("", vehicleController.CreateVehicle)
			vehicles.PUT("/:imei", vehicleController.UpdateVehicle)
			vehicles.DELETE("/:imei", vehicleController.DeleteVehicle)
		}

		// GPS tracking routes
		gps := v1.Group("/gps")
		{
			gps.GET("", gpsController.GetGPSData)
			gps.GET("/latest", gpsController.GetLatestGPSData)
			gps.GET("/:imei", gpsController.GetGPSDataByIMEI)
			gps.GET("/:imei/latest", gpsController.GetLatestGPSDataByIMEI)
			gps.GET("/:imei/route", gpsController.GetGPSRoute)
			gps.DELETE("/:id", gpsController.DeleteGPSData)
		}

		// Control routes for oil and electricity
		control := v1.Group("/control")
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

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"message":   "Luna IoT Server is running",
			"websocket": "/ws",
			"api":       "/api/v1",
		})
	})
}
