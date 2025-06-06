package db

import (
	"fmt"
	"log"
	"luna_iot_server/config"
	"luna_iot_server/internal/models"
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
	fmt.Println("DSN:", dsn)
	// dsn := "host=84.247.131.246 port=5433 user=luna role=luna password=Luna@#$321 dbname=luna_iot sslmode=disable"

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	log.Println("Database connection established successfully")

	// Run auto-migrations
	if err := RunMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %v", err)
	}

	return nil
}

// RunMigrations runs all database migrations
func RunMigrations() error {
	log.Println("Running database migrations...")

	// Check if we need to reset tables (only if there are constraint conflicts)
	shouldReset := false

	// Check if vehicles table exists but has constraint issues
	if DB.Migrator().HasTable(&models.Vehicle{}) {
		// Try a simple query to check if table is problematic
		var count int64
		err := DB.Model(&models.Vehicle{}).Count(&count).Error
		if err != nil && (strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "constraint")) {
			shouldReset = true
			log.Println("Detected constraint issues, will reset tables...")
		}
	}

	if shouldReset {
		// Drop tables in reverse order to handle foreign key constraints
		if DB.Migrator().HasTable(&models.GPSData{}) {
			log.Println("Dropping existing gps_data table...")
			DB.Migrator().DropTable(&models.GPSData{})
		}

		if DB.Migrator().HasTable(&models.Vehicle{}) {
			log.Println("Dropping existing vehicles table...")
			DB.Migrator().DropTable(&models.Vehicle{})
		}

		if DB.Migrator().HasTable(&models.Device{}) {
			log.Println("Dropping existing devices table...")
			DB.Migrator().DropTable(&models.Device{})
		}

		if DB.Migrator().HasTable(&models.User{}) {
			log.Println("Dropping existing users table...")
			DB.Migrator().DropTable(&models.User{})
		}
	}

	// Create tables in the correct order
	log.Println("Creating/updating tables...")

	// Create base tables first (no foreign keys)
	err := DB.AutoMigrate(&models.User{})
	if err != nil {
		return fmt.Errorf("user table migration failed: %v", err)
	}
	log.Println("✓ Users table ready")

	err = DB.AutoMigrate(&models.Device{})
	if err != nil {
		return fmt.Errorf("device table migration failed: %v", err)
	}
	log.Println("✓ Devices table ready")

	// Create tables with foreign keys
	err = DB.AutoMigrate(&models.Vehicle{})
	if err != nil {
		return fmt.Errorf("vehicle table migration failed: %v", err)
	}
	log.Println("✓ Vehicles table ready")

	err = DB.AutoMigrate(&models.GPSData{})
	if err != nil {
		return fmt.Errorf("gps_data table migration failed: %v", err)
	}
	log.Println("✓ GPS data table ready")

	log.Println("Database migrations completed successfully")
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
