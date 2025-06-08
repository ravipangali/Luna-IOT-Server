package db

import (
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"

	"gorm.io/gorm"
)

// MigrateDB performs database migrations
func MigrateDB(db *gorm.DB) error {
	colors.PrintInfo("Running database migrations...")

	// Auto migrate the schema
	if err := db.AutoMigrate(&models.User{}, &models.Device{}, &models.Vehicle{}); err != nil {
		colors.PrintError("Failed to run migrations: %v", err)
		return err
	}

	// Manually update the image column to TEXT type
	if err := updateImageColumnToText(db); err != nil {
		colors.PrintError("Failed to update image column: %v", err)
		return err
	}

	// Fix the foreign key constraint between vehicles and devices
	if err := fixVehicleDeviceConstraint(db); err != nil {
		colors.PrintError("Failed to fix vehicle-device constraint: %v", err)
		return err
	}

	colors.PrintSuccess("Database migrations completed successfully")
	return nil
}

// updateImageColumnToText updates the image column in the users table to TEXT type
func updateImageColumnToText(db *gorm.DB) error {
	// Check if we need to alter the column type
	var columnType string
	result := db.Raw("SELECT data_type FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'image'").Scan(&columnType)

	if result.Error != nil {
		return result.Error
	}

	// If column is not TEXT, update it
	if columnType != "text" {
		colors.PrintInfo("Updating users.image column from %s to TEXT type", columnType)
		if err := db.Exec("ALTER TABLE users ALTER COLUMN image TYPE TEXT").Error; err != nil {
			return err
		}
		colors.PrintSuccess("Successfully updated users.image column to TEXT type")
	} else {
		colors.PrintInfo("users.image column is already TEXT type, no update needed")
	}

	return nil
}

// fixVehicleDeviceConstraint fixes the foreign key constraint between vehicles and devices
func fixVehicleDeviceConstraint(db *gorm.DB) error {
	colors.PrintInfo("Checking device-vehicle foreign key constraint...")

	// First check if the constraint exists
	var constraintExists int64
	err := db.Raw(`
		SELECT COUNT(*) 
		FROM information_schema.table_constraints 
		WHERE constraint_name = 'fk_vehicles_device' 
		AND table_name = 'vehicles'
	`).Count(&constraintExists).Error

	if err != nil {
		return err
	}

	if constraintExists > 0 {
		colors.PrintInfo("Found problematic constraint 'fk_vehicles_device', removing it...")

		// Drop the existing constraint
		if err := db.Exec("ALTER TABLE vehicles DROP CONSTRAINT IF EXISTS fk_vehicles_device").Error; err != nil {
			colors.PrintError("Failed to drop constraint: %v", err)
			return err
		}

		colors.PrintSuccess("Successfully removed the constraint")
	} else {
		colors.PrintInfo("Constraint 'fk_vehicles_device' not found, nothing to drop")
	}

	// We'll let the models re-create the constraints properly
	colors.PrintInfo("Vehicle-device relationship will be recreated with proper constraints")
	return nil
}
