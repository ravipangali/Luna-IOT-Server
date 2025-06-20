package main

import (
	"fmt"
	"log"
	"luna_iot_server/config"
	"luna_iot_server/pkg/colors"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		colors.PrintWarning("No .env file found, using system environment variables")
	}

	// Initialize database connection
	dbConfig := config.GetDatabaseConfig()
	dsn := dbConfig.GetDSN()
	colors.PrintDebug("Database DSN: %s", dsn)

	var err error
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	colors.PrintSuccess("Database connection established successfully")

	// Check user_vehicles table structure
	colors.PrintHeader("Checking user_vehicles table structure")

	// Check if table exists
	if !db.Migrator().HasTable("user_vehicles") {
		colors.PrintError("user_vehicles table does not exist!")
		return
	}
	colors.PrintSuccess("✓ user_vehicles table exists")

	// Check columns
	var columns []struct {
		ColumnName string `gorm:"column:column_name"`
		DataType   string `gorm:"column:data_type"`
	}

	db.Raw(`
		SELECT column_name, data_type 
		FROM information_schema.columns 
		WHERE table_name = 'user_vehicles'
		ORDER BY ordinal_position
	`).Scan(&columns)

	colors.PrintInfo("Columns in user_vehicles table:")
	for _, col := range columns {
		fmt.Printf("  - %s (%s)\n", col.ColumnName, col.DataType)
	}

	// Check primary key
	var pk []struct {
		ColumnName string `gorm:"column:column_name"`
	}

	db.Raw(`
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		ON tc.constraint_name = kcu.constraint_name
		WHERE tc.table_name = 'user_vehicles'
		AND tc.constraint_type = 'PRIMARY KEY'
	`).Scan(&pk)

	if len(pk) == 0 {
		colors.PrintError("No primary key found on user_vehicles table!")
	} else {
		colors.PrintSuccess("✓ Primary key found on column(s): ")
		for _, col := range pk {
			fmt.Printf("  - %s\n", col.ColumnName)
		}
	}

	// Check foreign keys
	var fks []struct {
		ConstraintName   string `gorm:"column:constraint_name"`
		ColumnName       string `gorm:"column:column_name"`
		ReferencedTable  string `gorm:"column:referenced_table"`
		ReferencedColumn string `gorm:"column:referenced_column"`
	}

	db.Raw(`
		SELECT
			tc.constraint_name,
			kcu.column_name,
			ccu.table_name AS referenced_table,
			ccu.column_name AS referenced_column
		FROM
			information_schema.table_constraints AS tc
			JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
			JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
		WHERE
			tc.constraint_type = 'FOREIGN KEY' AND
			tc.table_name = 'user_vehicles'
	`).Scan(&fks)

	if len(fks) == 0 {
		colors.PrintWarning("No foreign keys found on user_vehicles table")
	} else {
		colors.PrintSuccess("✓ Foreign keys found:")
		for _, fk := range fks {
			fmt.Printf("  - %s: %s references %s.%s\n",
				fk.ConstraintName, fk.ColumnName, fk.ReferencedTable, fk.ReferencedColumn)
		}
	}

	// Try a simple query
	var count int64
	err = db.Table("user_vehicles").Count(&count).Error
	if err != nil {
		colors.PrintError("Failed to query user_vehicles table: %v", err)
	} else {
		colors.PrintSuccess("✓ Successfully queried user_vehicles table. Record count: %d", count)
	}

	colors.PrintHeader("Database check completed")
}
