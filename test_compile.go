package main

import (
	"fmt"
	"luna_iot_server/config"
	"luna_iot_server/internal/models"
)

func main() {
	fmt.Println("Testing Luna IoT Server compilation...")

	// Test model creation
	user := models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Role:  models.UserRoleAdmin,
	}

	device := models.Device{
		IMEI:        "123456789012345",
		SimNo:       "9841234567",
		SimOperator: models.SimOperatorNcell,
		Protocol:    models.ProtocolGT06,
	}

	vehicle := models.Vehicle{
		IMEI:        "123456789012345",
		RegNo:       "BA-1-PA-1234",
		Name:        "Test Vehicle",
		VehicleType: models.VehicleTypeCar,
	}

	gpsData := models.GPSData{
		IMEI:         "123456789012345",
		ProtocolName: "GPS_LBS_STATUS",
	}

	// Test config
	dbConfig := config.GetDatabaseConfig()

	fmt.Printf("✓ User model: %s (%s)\n", user.Name, user.GetRoleString())
	fmt.Printf("✓ Device model: %s (%s)\n", device.IMEI, device.SimOperator)
	fmt.Printf("✓ Vehicle model: %s (%s)\n", vehicle.Name, vehicle.VehicleType)
	fmt.Printf("✓ GPS data model: %s\n", gpsData.ProtocolName)
	fmt.Printf("✓ Database config: %s:%s\n", dbConfig.Host, dbConfig.Port)

	fmt.Println("\n✅ All models and configurations compiled successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("1. Start PostgreSQL service (use start_postgres.bat)")
	fmt.Println("2. Create database: CREATE DATABASE luna_iot;")
	fmt.Println("3. Run: go run cmd/http-server/main.go")
}
