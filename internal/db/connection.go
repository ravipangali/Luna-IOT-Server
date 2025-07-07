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
		Logger: logger.Default.LogMode(logger.Info),
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

	// IMPORTANT: Fix vehicle-device foreign key constraints BEFORE running AutoMigrate
	// This prevents constraint violations during table creation
	colors.PrintInfo("Pre-migration: Fixing vehicle-device constraints...")
	if err := fixVehicleDeviceConstraint(DB); err != nil {
		return fmt.Errorf("failed to fix vehicle-device constraint: %v", err)
	}
	colors.PrintSuccess("✓ Vehicle-device relationship constraints cleaned up")

	// Use AutoMigrate for all models. It will create tables, add missing columns,
	// and change column types, but it will NOT delete data.
	colors.PrintInfo("Running Auto-Migrations for all models...")
	err := DB.AutoMigrate(
		&models.User{},
		&models.DeviceModel{},
		&models.Device{},
		&models.Vehicle{},
		&models.GPSData{},
		&models.UserVehicle{},
	)
	if err != nil {
		return fmt.Errorf("auto-migration failed: %v", err)
	}
	colors.PrintSuccess("✓ All models migrated successfully")

	// The functions below are for manual migrations that might not be fully
	// covered by AutoMigrate, such as changing column types or fixing constraints.
	// They will be executed safely without dropping tables.

	// Update the image column in the users table to TEXT type
	if err := updateImageColumnToText(DB); err != nil {
		return fmt.Errorf("failed to update image column: %v", err)
	}
	colors.PrintSuccess("✓ User image column updated")

	// Update latitude and longitude precision
	if err := updateLatLongPrecision(DB); err != nil {
		return fmt.Errorf("failed to update GPS precision: %v", err)
	}
	colors.PrintSuccess("✓ GPS coordinate precision enhanced")

	// The ensureUserVehicleColumns function now uses AutoMigrate, so this serves as a redundant check.
	// This is safe to keep.
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
