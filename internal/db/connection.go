package db

import (
	"fmt"
	"log"
	"luna_iot_server/config"
	"luna_iot_server/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Initialize establishes database connection and runs migrations
func Initialize() error {
	dbConfig := config.GetDatabaseConfig()
	dsn := dbConfig.GetDSN()

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

	err := DB.AutoMigrate(
		&models.User{},
		&models.Device{},
		&models.Vehicle{},
		&models.GPSData{},
	)

	if err != nil {
		return fmt.Errorf("migration failed: %v", err)
	}

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
