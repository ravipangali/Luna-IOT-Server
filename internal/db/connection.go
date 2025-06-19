package db

import (
	"fmt"
	"luna_iot_server/config"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"
	"strings"

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

	// Check if we need to reset tables (only if there are constraint conflicts)
	shouldReset := false

	// Check if vehicles table exists but has constraint issues
	if DB.Migrator().HasTable(&models.Vehicle{}) {
		// Try a simple query to check if table is problematic
		var count int64
		err := DB.Model(&models.Vehicle{}).Count(&count).Error
		if err != nil && (strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "constraint")) {
			shouldReset = true
			colors.PrintWarning("Detected constraint issues, will reset tables...")
		}
	}

	if shouldReset {
		// Drop tables in reverse order to handle foreign key constraints
		if DB.Migrator().HasTable(&models.GPSData{}) {
			colors.PrintInfo("Dropping existing gps_data table...")
			DB.Migrator().DropTable(&models.GPSData{})
		}

		if DB.Migrator().HasTable(&models.Vehicle{}) {
			colors.PrintInfo("Dropping existing vehicles table...")
			DB.Migrator().DropTable(&models.Vehicle{})
		}

		if DB.Migrator().HasTable(&models.Device{}) {
			colors.PrintInfo("Dropping existing devices table...")
			DB.Migrator().DropTable(&models.Device{})
		}

		if DB.Migrator().HasTable(&models.User{}) {
			colors.PrintInfo("Dropping existing users table...")
			DB.Migrator().DropTable(&models.User{})
		}
	}

	// Create tables in the correct order
	colors.PrintInfo("Creating/updating database tables...")

	// Create base tables first (no foreign keys)
	err := DB.AutoMigrate(&models.User{})
	if err != nil {
		return fmt.Errorf("user table migration failed: %v", err)
	}
	colors.PrintSuccess("✓ Users table ready")

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

	err = DB.AutoMigrate(&models.GPSData{})
	if err != nil {
		return fmt.Errorf("gps_data table migration failed: %v", err)
	}
	colors.PrintSuccess("✓ GPS data table ready")

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

	// Add bearing column for map rotation support
	if err := addBearingColumn(DB); err != nil {
		return fmt.Errorf("failed to add bearing column: %v", err)
	}
	colors.PrintSuccess("✓ Bearing column added for map rotation")

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

// addBearingColumn adds bearing column to gps_data table for map rotation support
func addBearingColumn(db *gorm.DB) error {
	colors.PrintInfo("Adding bearing column to gps_data table for map rotation support...")

	// Check if bearing column already exists
	var bearingColumnExists int64
	db.Raw(`
		SELECT COUNT(*) 
		FROM information_schema.columns 
		WHERE table_name = 'gps_data' 
		AND column_name = 'bearing'
	`).Count(&bearingColumnExists)

	if bearingColumnExists == 0 {
		colors.PrintInfo("Adding bearing column to gps_data table...")
		if err := db.Exec("ALTER TABLE gps_data ADD COLUMN bearing DECIMAL(6,3)").Error; err != nil {
			return err
		}

		// Add index on bearing column for performance
		if err := db.Exec("CREATE INDEX idx_gps_data_bearing ON gps_data(bearing) WHERE bearing IS NOT NULL").Error; err != nil {
			colors.PrintWarning("Could not create bearing index (might already exist): %v", err)
		} else {
			colors.PrintSuccess("Created index on bearing column")
		}

		colors.PrintSuccess("✓ Added bearing column to gps_data table")
	} else {
		colors.PrintInfo("Bearing column already exists in gps_data table")
	}

	return nil
}
