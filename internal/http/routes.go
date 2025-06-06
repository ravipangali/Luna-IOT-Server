package http

import (
	"luna_iot_server/internal/http/controllers"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all API routes
func SetupRoutes(router *gin.Engine) {
	// Initialize controllers
	userController := controllers.NewUserController()
	deviceController := controllers.NewDeviceController()
	vehicleController := controllers.NewVehicleController()
	gpsController := controllers.NewGPSController()

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
	}

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Luna IoT Server is running",
		})
	})
}
