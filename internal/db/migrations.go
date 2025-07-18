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
	if err := db.AutoMigrate(&models.User{}, &models.Device{}, &models.Vehicle{}, &models.Notification{}, &models.NotificationUser{}); err != nil {
		colors.PrintError("Failed to run migrations: %v", err)
		return err
	}

	// Update the image column in the users table to TEXT type
	if err := updateImageColumnToText(DB); err != nil {
		return fmt.Errorf("failed to update image column: %v", err)
	}
	colors.PrintSuccess("✓ User image column updated")

	// The ensureUserVehicleColumns function now uses AutoMigrate, so this serves as a redundant check.
	// This is safe to keep.
	if err := ensureUserVehicleColumns(DB); err != nil {
		return fmt.Errorf("failed to ensure user_vehicles table structure: %v", err)
	}
	colors.PrintSuccess("✓ User-Vehicle permissions table structure verified")

	// Add token columns to users table if they don't exist
	if err := addTokenColumnsToUsers(db); err != nil {
		colors.PrintError("Failed to add token columns: %v", err)
		return err
	}

	// Add FCM token column to users table if it doesn't exist
	if err := addFCMTokenColumn(db); err != nil {
		colors.PrintError("Failed to add FCM token column: %v", err)
		return err
	}

	// Add image_data column to notifications table if it doesn't exist
	if err := addImageDataColumnToNotifications(db); err != nil {
		colors.PrintError("Failed to add image_data column: %v", err)
		return err
	}

	// Fix the foreign key constraint between vehicles and devices
	if err := fixVehicleDeviceConstraint(db); err != nil {
		colors.PrintError("Failed to fix vehicle-device constraint: %v", err)
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

	// Update token system to remove expiration
	if err := updateTokenSystem(db); err != nil {
		colors.PrintError("Failed to update token system: %v", err)
		return err
	}

	// Update notification image URLs to use new public endpoint
	if err := updateNotificationImageURLs(db); err != nil {
		colors.PrintError("Failed to update notification image URLs: %v", err)
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

	// First, drop ALL foreign key constraints from both tables to ensure clean state
	colors.PrintInfo("Removing ALL foreign key constraints from devices and vehicles tables...")

	// Get all foreign key constraints from devices table
	var deviceConstraints []string
	db.Raw(`
		SELECT constraint_name 
		FROM information_schema.table_constraints 
		WHERE table_name = 'devices' 
		AND constraint_type = 'FOREIGN KEY'
	`).Pluck("constraint_name", &deviceConstraints)

	for _, constraint := range deviceConstraints {
		colors.PrintInfo("Removing foreign key constraint '%s' from devices table", constraint)
		if err := db.Exec("ALTER TABLE devices DROP CONSTRAINT IF EXISTS " + constraint).Error; err != nil {
			colors.PrintWarning("Could not remove constraint '%s' from devices: %v", constraint, err)
		} else {
			colors.PrintSuccess("Removed constraint '%s' from devices table", constraint)
		}
	}

	// Get all foreign key constraints from vehicles table
	var vehicleConstraints []string
	db.Raw(`
		SELECT constraint_name 
		FROM information_schema.table_constraints 
		WHERE table_name = 'vehicles' 
		AND constraint_type = 'FOREIGN KEY'
	`).Pluck("constraint_name", &vehicleConstraints)

	for _, constraint := range vehicleConstraints {
		colors.PrintInfo("Removing foreign key constraint '%s' from vehicles table", constraint)
		if err := db.Exec("ALTER TABLE vehicles DROP CONSTRAINT IF EXISTS " + constraint).Error; err != nil {
			colors.PrintWarning("Could not remove constraint '%s' from vehicles: %v", constraint, err)
		} else {
			colors.PrintSuccess("Removed constraint '%s' from vehicles table", constraint)
		}
	}

	// Also try to remove common constraint names that might exist
	commonConstraints := []string{
		"fk_vehicles_device",
		"fk_devices_vehicle",
		"fk_device_vehicle",
		"fk_vehicles_imei",
		"devices_vehicle_fkey",
		"devices_imei_fkey",
		"vehicles_device_fkey",
		"vehicles_imei_fkey",
	}

	for _, constraint := range commonConstraints {
		// Try removing from devices table
		db.Exec("ALTER TABLE devices DROP CONSTRAINT IF EXISTS " + constraint)
		// Try removing from vehicles table
		db.Exec("ALTER TABLE vehicles DROP CONSTRAINT IF EXISTS " + constraint)
	}

	colors.PrintSuccess("✓ All foreign key constraints removed from devices and vehicles tables")
	colors.PrintSuccess("✓ Devices table is now independent and can be created without constraints")
	colors.PrintInfo("✓ Vehicles will reference devices via IMEI, but devices are independent")
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

// updateTokenSystem updates the token system to remove expiration
func updateTokenSystem(db *gorm.DB) error {
	colors.PrintInfo("Updating token system to remove expiration...")

	// Clear all existing token expiration times since tokens no longer expire
	colors.PrintInfo("Clearing all token expiration times...")
	if err := db.Exec("UPDATE users SET token_exp = NULL WHERE token_exp IS NOT NULL").Error; err != nil {
		colors.PrintWarning("Could not clear token expiration times: %v", err)
	} else {
		colors.PrintSuccess("Cleared all token expiration times")
	}

	// Add a comment to the token_exp column to indicate it's no longer used
	colors.PrintInfo("Adding comment to token_exp column...")
	if err := db.Exec("COMMENT ON COLUMN users.token_exp IS 'No longer used - tokens do not expire'").Error; err != nil {
		colors.PrintWarning("Could not add comment to token_exp column: %v", err)
	} else {
		colors.PrintSuccess("Added comment to token_exp column")
	}

	colors.PrintSuccess("Token system updated successfully")
	return nil
}

// addFCMTokenColumn adds FCM token column to users table
func addFCMTokenColumn(db *gorm.DB) error {
	colors.PrintInfo("Adding FCM token column to users table...")

	// Check if fcm_token column already exists
	var columnExists int64
	db.Raw(`
		SELECT COUNT(*) 
		FROM information_schema.columns 
		WHERE table_name = 'users' 
		AND column_name = 'fcm_token'
	`).Count(&columnExists)

	if columnExists > 0 {
		colors.PrintInfo("FCM token column already exists in users table")
		return nil
	}

	// Add fcm_token column
	if err := db.Exec("ALTER TABLE users ADD COLUMN fcm_token VARCHAR(255)").Error; err != nil {
		colors.PrintError("Failed to add fcm_token column: %v", err)
		return fmt.Errorf("failed to add fcm_token column: %v", err)
	}

	colors.PrintSuccess("✓ FCM token column added to users table")
	return nil
}

// addImageDataColumnToNotifications adds image_data column to notifications table
func addImageDataColumnToNotifications(db *gorm.DB) error {
	colors.PrintInfo("Adding image_data column to notifications table...")

	// Check if image_data column already exists
	var columnExists int64
	db.Raw(`
		SELECT COUNT(*) 
		FROM information_schema.columns 
		WHERE table_name = 'notifications' 
		AND column_name = 'image_data'
	`).Count(&columnExists)

	if columnExists > 0 {
		colors.PrintInfo("image_data column already exists in notifications table")
		return nil
	}

	// Add image_data column
	if err := db.Exec("ALTER TABLE notifications ADD COLUMN image_data TEXT").Error; err != nil {
		colors.PrintError("Failed to add image_data column: %v", err)
		return fmt.Errorf("failed to add image_data column: %v", err)
	}

	colors.PrintSuccess("✓ image_data column added to notifications table")
	return nil
}

// updateNotificationImageURLs updates existing notification image URLs to use the new public endpoint
func updateNotificationImageURLs(db *gorm.DB) error {
	colors.PrintInfo("Updating notification image URLs to use new public endpoint...")

	// Update image_data column
	result := db.Exec(`
		UPDATE notifications 
		SET image_data = REPLACE(image_data, '/api/v1/files/notifications/', '/api/v1/public/files/notifications/')
		WHERE image_data LIKE '%/api/v1/files/notifications/%'
	`)

	if result.Error != nil {
		colors.PrintWarning("Could not update image_data URLs: %v", result.Error)
	} else {
		colors.PrintInfo("Updated %d image_data URLs", result.RowsAffected)
	}

	// Update image_url column
	result = db.Exec(`
		UPDATE notifications 
		SET image_url = REPLACE(image_url, '/api/v1/files/notifications/', '/api/v1/public/files/notifications/')
		WHERE image_url LIKE '%/api/v1/files/notifications/%'
	`)

	if result.Error != nil {
		colors.PrintWarning("Could not update image_url URLs: %v", result.Error)
	} else {
		colors.PrintInfo("Updated %d image_url URLs", result.RowsAffected)
	}

	colors.PrintSuccess("✓ Notification image URLs updated to use public endpoint")
	return nil
}
