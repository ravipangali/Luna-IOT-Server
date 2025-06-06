package config

import (
	"fmt"
	"os"
)

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Role     string
	Password string
	DBName   string
	SSLMode  string
}

// GetDatabaseConfig returns database configuration from environment variables
func GetDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Host:     getEnv("DB_HOST", "84.247.131.246"),
		Port:     getEnv("DB_PORT", "5433"),
		User:     getEnv("DB_USER", "luna"),
		Role:     getEnv("DB_ROLE", "luna"),
		Password: getEnv("DB_PASSWORD", "Luna@#$321"),
		DBName:   getEnv("DB_NAME", "luna_iot"),
		SSLMode:  getEnv("DB_SSL_MODE", "disable"),
	}
}

// GetDSN returns the database connection string
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s role=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Role, c.Password, c.DBName, c.SSLMode)
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
