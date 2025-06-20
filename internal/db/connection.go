package db

import (
	"fmt"
	"luna_iot_server/config"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Initialize establishes database connection and runs migrations
func Initialize() error {
	dbConfig := config.GetDatabaseConfig()
	dsn := dbConfig.GetDSN()
	colors.PrintDebug("Database DSN: %s", dsn)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	colors.PrintSuccess("Database connection established successfully")

	// Run auto-migrations
	if err := RunMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %v", err)
	}

	return nil
}

// RunMigrations runs all database migrations
func RunMigrations() error {
	colors.PrintSubHeader("Running Database Migrations")

	// IMPORTANT: Force reset all tables to fix schema issues
	colors.PrintWarning("Forcefully resetting database schema to fix persistent issues...")

	// Drop all tables in the correct order to avoid foreign key constraint errors
	DB.Exec("DROP TABLE IF EXISTS gps_data CASCADE")
	DB.Exec("DROP TABLE IF EXISTS user_vehicles CASCADE")
	DB.Exec("DROP TABLE IF EXISTS vehicles CASCADE")
	DB.Exec("DROP TABLE IF EXISTS devices CASCADE")
	DB.Exec("DROP TABLE IF EXISTS device_models CASCADE")
	DB.Exec("DROP TABLE IF EXISTS users CASCADE")

	colors.PrintSuccess("Database tables dropped successfully")

	// Create tables in the correct order
	colors.PrintInfo("Creating database tables from scratch...")

	// Create base tables first (no foreign keys)
	err := DB.AutoMigrate(&models.User{})
	if err != nil {
		return fmt.Errorf("user table migration failed: %v", err)
	}
	colors.PrintSuccess("✓ Users table ready")

	err = DB.AutoMigrate(&models.DeviceModel{})
	if err != nil {
		return fmt.Errorf("device_model table migration failed: %v", err)
	}
	colors.PrintSuccess("✓ Device Models table ready")

	err = DB.AutoMigrate(&models.Device{})
	if err != nil {
		return fmt.Errorf("device table migration failed: %v", err)
	}
	colors.PrintSuccess("✓ Devices table ready")

	// Create tables with foreign keys
	err = DB.AutoMigrate(&models.Vehicle{})
	if err != nil {
		return fmt.Errorf("vehicle table migration failed: %v", err)
	}
	colors.PrintSuccess("✓ Vehicles table ready")

	// Create GPS data table
	err = DB.AutoMigrate(&models.GPSData{})
	if err != nil {
		return fmt.Errorf("gps_data table migration failed: %v", err)
	}
	colors.PrintSuccess("✓ GPS data table ready")

	// Create user-vehicle relationship table
	err = DB.AutoMigrate(&models.UserVehicle{})
	if err != nil {
		return fmt.Errorf("user_vehicle table migration failed: %v", err)
	}
	colors.PrintSuccess("✓ User-Vehicle relationship table ready")

	// Update the image column in the users table to TEXT type
	if err := updateImageColumnToText(DB); err != nil {
		return fmt.Errorf("failed to update image column: %v", err)
	}
	colors.PrintSuccess("✓ User image column updated")

	// Fix vehicle-device foreign key constraint
	if err := fixVehicleDeviceConstraint(DB); err != nil {
		return fmt.Errorf("failed to fix vehicle-device constraint: %v", err)
	}
	colors.PrintSuccess("✓ Vehicle-device relationship fixed")

	// Update latitude and longitude precision
	if err := updateLatLongPrecision(DB); err != nil {
		return fmt.Errorf("failed to update GPS precision: %v", err)
	}
	colors.PrintSuccess("✓ GPS coordinate precision enhanced")

	// Ensure user_vehicles table has all required permission columns
	if err := ensureUserVehicleColumns(DB); err != nil {
		return fmt.Errorf("failed to ensure user_vehicles table structure: %v", err)
	}
	colors.PrintSuccess("✓ User-Vehicle permissions table structure verified")

	colors.PrintHeader("DATABASE MIGRATIONS COMPLETED SUCCESSFULLY")
	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}

// Close closes the database connection
func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
