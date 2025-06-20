package db

import (
	"fmt"
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

	// Add token columns to users table if they don't exist
	if err := addTokenColumnsToUsers(db); err != nil {
		colors.PrintError("Failed to add token columns: %v", err)
		return err
	}

	// Fix the foreign key constraint between vehicles and devices
	if err := fixVehicleDeviceConstraint(db); err != nil {
		colors.PrintError("Failed to fix vehicle-device constraint: %v", err)
		return err
	}

	// Update latitude and longitude precision
	if err := updateLatLongPrecision(db); err != nil {
		colors.PrintError("Failed to update latitude and longitude precision: %v", err)
		return err
	}

	// Ensure user_vehicles table has all required permission columns
	if err := ensureUserVehicleColumns(db); err != nil {
		colors.PrintError("Failed to ensure user_vehicles table: %v", err)
		return err
	}

	// Fix user_vehicles foreign key constraints
	if err := fixUserVehicleConstraints(db); err != nil {
		colors.PrintError("Failed to fix user_vehicles constraints: %v", err)
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

// addTokenColumnsToUsers adds token and token_exp columns to users table if they don't exist
func addTokenColumnsToUsers(db *gorm.DB) error {
	colors.PrintInfo("Checking for token columns in users table...")

	// Check if token column exists
	var tokenColumnExists int64
	db.Raw(`
		SELECT COUNT(*) 
		FROM information_schema.columns 
		WHERE table_name = 'users' 
		AND column_name = 'token'
	`).Count(&tokenColumnExists)

	if tokenColumnExists == 0 {
		colors.PrintInfo("Adding token column to users table...")
		if err := db.Exec("ALTER TABLE users ADD COLUMN token VARCHAR(255)").Error; err != nil {
			return err
		}
		colors.PrintSuccess("Added token column to users table")
	} else {
		colors.PrintInfo("Token column already exists in users table")
	}

	// Check if token_exp column exists
	var tokenExpColumnExists int64
	db.Raw(`
		SELECT COUNT(*) 
		FROM information_schema.columns 
		WHERE table_name = 'users' 
		AND column_name = 'token_exp'
	`).Count(&tokenExpColumnExists)

	if tokenExpColumnExists == 0 {
		colors.PrintInfo("Adding token_exp column to users table...")
		if err := db.Exec("ALTER TABLE users ADD COLUMN token_exp TIMESTAMP").Error; err != nil {
			return err
		}
		colors.PrintSuccess("Added token_exp column to users table")
	} else {
		colors.PrintInfo("Token_exp column already exists in users table")
	}

	// Add unique index on token column if it doesn't exist
	var tokenIndexExists int64
	db.Raw(`
		SELECT COUNT(*) 
		FROM pg_indexes 
		WHERE tablename = 'users' 
		AND indexname LIKE '%token%'
	`).Count(&tokenIndexExists)

	if tokenIndexExists == 0 {
		colors.PrintInfo("Creating unique index on token column...")
		if err := db.Exec("CREATE UNIQUE INDEX idx_users_token ON users(token) WHERE token IS NOT NULL").Error; err != nil {
			// Index might already exist with a different name, log warning but continue
			colors.PrintWarning("Could not create token index (might already exist): %v", err)
		} else {
			colors.PrintSuccess("Created unique index on token column")
		}
	} else {
		colors.PrintInfo("Token index already exists")
	}

	// Add index on token_exp column if it doesn't exist
	var tokenExpIndexExists int64
	db.Raw(`
		SELECT COUNT(*) 
		FROM pg_indexes 
		WHERE tablename = 'users' 
		AND indexname LIKE '%token_exp%'
	`).Count(&tokenExpIndexExists)

	if tokenExpIndexExists == 0 {
		colors.PrintInfo("Creating index on token_exp column...")
		if err := db.Exec("CREATE INDEX idx_users_token_exp ON users(token_exp)").Error; err != nil {
			colors.PrintWarning("Could not create token_exp index (might already exist): %v", err)
		} else {
			colors.PrintSuccess("Created index on token_exp column")
		}
	} else {
		colors.PrintInfo("Token_exp index already exists")
	}

	return nil
}

// fixVehicleDeviceConstraint fixes the foreign key constraint between vehicles and devices
func fixVehicleDeviceConstraint(db *gorm.DB) error {
	colors.PrintInfo("Checking and fixing device-vehicle foreign key constraints...")

	// Remove any foreign key constraints from devices table that reference vehicles
	// This is wrong - devices should be independent
	colors.PrintInfo("Removing any constraints from devices table...")

	// Drop any constraint that might exist on devices table referencing vehicles
	constraints := []string{
		"fk_vehicles_device",
		"fk_devices_vehicle",
		"fk_device_vehicle",
		"devices_vehicle_fkey",
		"devices_imei_fkey",
	}

	for _, constraint := range constraints {
		// Check if constraint exists on devices table
		var existsOnDevices int64
		db.Raw(`
			SELECT COUNT(*) 
			FROM information_schema.table_constraints 
			WHERE constraint_name = ? 
			AND table_name = 'devices'
		`, constraint).Count(&existsOnDevices)

		if existsOnDevices > 0 {
			colors.PrintInfo("Found constraint '%s' on devices table, removing it...", constraint)
			db.Exec("ALTER TABLE devices DROP CONSTRAINT IF EXISTS " + constraint)
			colors.PrintSuccess("Removed constraint '%s' from devices table", constraint)
		}

		// Check if constraint exists on vehicles table
		var existsOnVehicles int64
		db.Raw(`
			SELECT COUNT(*) 
			FROM information_schema.table_constraints 
			WHERE constraint_name = ? 
			AND table_name = 'vehicles'
		`, constraint).Count(&existsOnVehicles)

		if existsOnVehicles > 0 {
			colors.PrintInfo("Found constraint '%s' on vehicles table, removing it...", constraint)
			db.Exec("ALTER TABLE vehicles DROP CONSTRAINT IF EXISTS " + constraint)
			colors.PrintSuccess("Removed constraint '%s' from vehicles table", constraint)
		}
	}

	// Make sure devices table has no foreign key constraints
	colors.PrintInfo("Ensuring devices table is completely independent...")

	// Get all foreign key constraints on devices table
	var fkConstraints []string
	db.Raw(`
		SELECT constraint_name 
		FROM information_schema.table_constraints 
		WHERE table_name = 'devices' 
		AND constraint_type = 'FOREIGN KEY'
	`).Pluck("constraint_name", &fkConstraints)

	for _, fk := range fkConstraints {
		colors.PrintInfo("Removing foreign key constraint '%s' from devices table", fk)
		db.Exec("ALTER TABLE devices DROP CONSTRAINT IF EXISTS " + fk)
		colors.PrintSuccess("Removed foreign key constraint '%s'", fk)
	}

	colors.PrintSuccess("✓ Devices table is now independent and can be created without constraints")
	colors.PrintInfo("✓ Vehicles will reference devices via IMEI, but devices are independent")
	return nil
}

// updateLatLongPrecision updates latitude and longitude columns to use higher precision
func updateLatLongPrecision(db *gorm.DB) error {
	colors.PrintInfo("Updating latitude and longitude precision to 15,12 for enhanced GPS accuracy...")

	// Check current data types
	var latDataType string
	var lngDataType string

	db.Raw(`
		SELECT data_type || '(' || 
		       COALESCE(numeric_precision::text, '') || ',' || 
		       COALESCE(numeric_scale::text, '') || ')' as data_type
		FROM information_schema.columns 
		WHERE table_name = 'gps_data' AND column_name = 'latitude'
	`).Scan(&latDataType)

	db.Raw(`
		SELECT data_type || '(' || 
		       COALESCE(numeric_precision::text, '') || ',' || 
		       COALESCE(numeric_scale::text, '') || ')' as data_type
		FROM information_schema.columns 
		WHERE table_name = 'gps_data' AND column_name = 'longitude'
	`).Scan(&lngDataType)

	colors.PrintInfo("Current latitude type: %s", latDataType)
	colors.PrintInfo("Current longitude type: %s", lngDataType)

	// Update latitude column
	if err := db.Exec("ALTER TABLE gps_data ALTER COLUMN latitude TYPE NUMERIC(15,12)").Error; err != nil {
		colors.PrintWarning("Failed to update latitude precision: %v", err)
	} else {
		colors.PrintSuccess("✓ Updated latitude column to NUMERIC(15,12)")
	}

	// Update longitude column
	if err := db.Exec("ALTER TABLE gps_data ALTER COLUMN longitude TYPE NUMERIC(15,12)").Error; err != nil {
		colors.PrintWarning("Failed to update longitude precision: %v", err)
	} else {
		colors.PrintSuccess("✓ Updated longitude column to NUMERIC(15,12)")
	}

	return nil
}

// ensureUserVehicleColumns ensures that the user_vehicles table has all required permission columns
func ensureUserVehicleColumns(db *gorm.DB) error {
	colors.PrintInfo("Ensuring user_vehicles table has all required permission columns via AutoMigrate...")

	// AutoMigrate will create the table if it doesn't exist,
	// and add any missing columns, indexes, or change column types.
	if err := db.AutoMigrate(&models.UserVehicle{}); err != nil {
		colors.PrintError("Failed to AutoMigrate user_vehicles table: %v", err)
		return fmt.Errorf("failed to auto-migrate user_vehicles table: %v", err)
	}

	colors.PrintSuccess("✓ User-Vehicle permissions table structure verified and synchronized")
	return nil
}

// fixUserVehicleConstraints ensures proper foreign key constraints for user_vehicles table
func fixUserVehicleConstraints(db *gorm.DB) error {
	colors.PrintInfo("Fixing user_vehicles foreign key constraints...")

	// Check if user_vehicles table exists
	if !db.Migrator().HasTable("user_vehicles") {
		colors.PrintInfo("user_vehicles table does not exist, skipping constraint fix")
		return nil
	}

	// Remove problematic foreign key constraints and recreate them properly
	constraints := []string{
		"fk_user_vehicles_user",
		"fk_user_vehicles_vehicle",
		"fk_user_vehicles_granted_by_user",
		"user_vehicles_user_id_fkey",
		"user_vehicles_vehicle_id_fkey",
		"user_vehicles_granted_by_fkey",
	}

	for _, constraint := range constraints {
		// Check if constraint exists
		var exists int64
		db.Raw(`
			SELECT COUNT(*) 
			FROM information_schema.table_constraints 
			WHERE constraint_name = ? 
			AND table_name = 'user_vehicles'
		`, constraint).Count(&exists)

		if exists > 0 {
			colors.PrintInfo("Removing constraint '%s' from user_vehicles table", constraint)
			db.Exec("ALTER TABLE user_vehicles DROP CONSTRAINT IF EXISTS " + constraint)
		}
	}

	// Add proper foreign key constraints with better error handling
	colors.PrintInfo("Adding proper foreign key constraints to user_vehicles table")

	// Check if users table exists before adding constraint
	if db.Migrator().HasTable("users") {
		// User ID foreign key
		if err := db.Exec(`
			ALTER TABLE user_vehicles 
			ADD CONSTRAINT fk_user_vehicles_user 
			FOREIGN KEY (user_id) REFERENCES users(id) 
			ON UPDATE CASCADE ON DELETE CASCADE
		`).Error; err != nil {
			colors.PrintWarning("Failed to add user foreign key constraint: %v", err)
		} else {
			colors.PrintSuccess("Added user foreign key constraint")
		}
	}

	// Check if vehicles table exists before adding constraint
	if db.Migrator().HasTable("vehicles") {
		// Vehicle ID foreign key (references IMEI)
		if err := db.Exec(`
			ALTER TABLE user_vehicles 
			ADD CONSTRAINT fk_user_vehicles_vehicle 
			FOREIGN KEY (vehicle_id) REFERENCES vehicles(imei) 
			ON UPDATE CASCADE ON DELETE CASCADE
		`).Error; err != nil {
			colors.PrintWarning("Failed to add vehicle foreign key constraint: %v", err)
		} else {
			colors.PrintSuccess("Added vehicle foreign key constraint")
		}
	}

	// Granted by foreign key (nullable, allows NULL)
	if db.Migrator().HasTable("users") {
		if err := db.Exec(`
			ALTER TABLE user_vehicles 
			ADD CONSTRAINT fk_user_vehicles_granted_by 
			FOREIGN KEY (granted_by) REFERENCES users(id) 
			ON UPDATE CASCADE ON DELETE SET NULL
		`).Error; err != nil {
			colors.PrintWarning("Failed to add granted_by foreign key constraint: %v", err)
		} else {
			colors.PrintSuccess("Added granted_by foreign key constraint")
		}
	}

	colors.PrintSuccess("✓ user_vehicles foreign key constraints fixed")
	return nil
}
