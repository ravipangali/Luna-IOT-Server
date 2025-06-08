# Luna IoT Server - Complete Code Analysis & File Documentation

## 📋 Table of Contents

1. [Project Overview & Architecture](#project-overview--architecture)
2. [Root Directory Files](#root-directory-files)
3. [cmd/ Directory - Entry Points](#cmd-directory---entry-points)
4. [config/ Directory - Configuration](#config-directory---configuration)
5. [internal/db/ - Database Layer](#internaldb---database-layer)
6. [internal/models/ - Data Models](#internalmodels---data-models)
7. [internal/protocol/ - Protocol Implementation](#internalprotocol---protocol-implementation)
8. [internal/http/ - HTTP Server](#internalhttp---http-server)
9. [internal/tcp/ - TCP Server](#internaltcp---tcp-server)
10. [pkg/colors/ - Utilities](#pkgcolors---utilities)

---

## 📖 Project Overview & Architecture

**Luna IoT Server** is a comprehensive GPS tracking system that handles GT06 protocol devices. It consists of:

- **Dual Server Architecture**: TCP server for IoT devices + HTTP server for web API
- **Real-time Processing**: Live GPS data ingestion and device control
- **Database Integration**: PostgreSQL with automatic migrations
- **Protocol Support**: Complete GT06 binary protocol implementation
- **Fleet Management**: Vehicles, users, and device management

**System Flow**:
```
IoT Devices → TCP Server → Protocol Decoder → Database
                    ↓
              Control Controller ← HTTP API ← Web Applications
```

---

## 📁 Root Directory Files

### 📄 `main.go` - Primary Application Entry Point

**Purpose**: Orchestrates the entire application startup and server coordination.

```go
package main

import (
    "fmt"
    "log"
    "os"
    "os/signal"
    "sync"
    "syscall"
    
    "github.com/joho/godotenv"
    "luna_iot_server/internal/db"
    "luna_iot_server/internal/http"
    "luna_iot_server/internal/http/controllers"
    "luna_iot_server/internal/tcp"
    "luna_iot_server/pkg/colors"
)
```

**Import Analysis**:
- `sync`: For WaitGroup to manage concurrent goroutines
- `os/signal`: Handles system shutdown signals (Ctrl+C, SIGTERM)
- `godotenv`: Loads environment variables from .env file
- Internal packages: Custom modules for database, servers, and utilities

**Main Function Breakdown**:

```go
func main() {
    // 1. Display professional banner
    colors.PrintBanner()
```

**Banner Display**: Shows ASCII art logo and system information for professional startup experience.

```go
    // 2. Load environment configuration
    if err := godotenv.Load(); err != nil {
        colors.PrintWarning("No .env file found, using system environment variables")
    } else {
        colors.PrintSuccess("Environment configuration loaded from .env file")
    }
```

**Configuration Loading**:
- Attempts to load `.env` file for local development
- Falls back to system environment variables in production
- Provides clear feedback about configuration source

```go
    // 3. Initialize database connection and migrations
    colors.PrintInfo("Initializing database connection...")
    if err := db.Initialize(); err != nil {
        colors.PrintError("Failed to initialize database: %v", err)
        log.Fatalf("Database initialization failed: %v", err)
    }
    defer db.Close()
```

**Database Initialization**:
- Establishes PostgreSQL connection using GORM
- Runs automatic migrations to create/update database schema
- Uses `defer` to ensure proper connection cleanup on exit
- Fatal error if database unavailable (app cannot function without DB)

```go
    // 4. Create shared control controller
    sharedControlController := controllers.NewControlController()
    colors.PrintSuccess("Shared control controller initialized")
```

**Shared State Management**:
- Creates single `ControlController` instance for device management
- Both TCP and HTTP servers will reference this same instance
- Enables HTTP API to control devices connected via TCP server

```go
    // 5. Get server ports from environment
    tcpPort := os.Getenv("TCP_PORT")
    if tcpPort == "" {
        tcpPort = "5000"  // Default TCP port for IoT devices
    }
    
    httpPort := os.Getenv("HTTP_PORT")
    if httpPort == "" {
        httpPort = "8080"  // Default HTTP port for web API
    }
```

**Port Configuration**:
- TCP Port 5000: For GT06 protocol device connections
- HTTP Port 8080: For REST API and web interface
- Environment variable override capability

```go
    // 6. Start servers concurrently
    var wg sync.WaitGroup
    errorChan := make(chan error, 2)
    
    // TCP Server goroutine for IoT devices
    wg.Add(1)
    go func() {
        defer wg.Done()
        tcpServer := tcp.NewServerWithController(tcpPort, sharedControlController)
        colors.PrintServer("📡", "Starting TCP Server on port %s for IoT devices", tcpPort)
        if err := tcpServer.Start(); err != nil {
            errorChan <- fmt.Errorf("TCP server error: %v", err)
        }
    }()
    
    // HTTP Server goroutine for REST API
    wg.Add(1)
    go func() {
        defer wg.Done()
        httpServer := http.NewServerWithController(httpPort, sharedControlController)
        colors.PrintServer("🌐", "Starting HTTP Server on port %s for REST API", httpPort)
        if err := httpServer.Start(); err != nil {
            errorChan <- fmt.Errorf("HTTP server error: %v", err)
        }
    }()
```

**Concurrent Server Architecture**:
- **WaitGroup**: Manages two independent goroutines
- **Error Channel**: Collects startup errors from either server
- **TCP Server**: Handles binary GT06 protocol from GPS devices
- **HTTP Server**: Provides REST API for web applications
- **Shared Controller**: Both servers use same device control instance

```go
    // 7. Graceful shutdown handling
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    
    colors.PrintSuccess("🚀 Luna IoT Server started successfully!")
    colors.PrintInfo("📡 TCP Server: localhost:%s (IoT Devices)", tcpPort)
    colors.PrintInfo("🌐 HTTP Server: localhost:%s (REST API)", httpPort)
    colors.PrintInfo("Press Ctrl+C to shutdown gracefully")
    
    // Wait for either error or shutdown signal
    select {
    case err := <-errorChan:
        colors.PrintError("Server startup failed: %v", err)
        return
    case <-quit:
        colors.PrintShutdown()
        return
    }
}
```

**Shutdown Management**:
- Listens for SIGINT (Ctrl+C) and SIGTERM signals
- `select` statement blocks until error or shutdown signal
- Provides clean application termination
- Shows server status and access information

### 📄 `go.mod` - Go Module Dependencies

```go
module luna_iot_server

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1           // High-performance HTTP web framework
    github.com/joho/godotenv v1.5.1           // Environment variable loader
    golang.org/x/crypto v0.17.0               // Cryptographic functions (bcrypt)
    gorm.io/driver/postgres v1.5.4            // PostgreSQL database driver
    gorm.io/gorm v1.25.5                      // Object-Relational Mapping library
)

require (
    // Indirect dependencies automatically managed
    github.com/bytedance/sonic v1.9.1
    github.com/chenzhuoyu/base64x v0.0.0-20221115062448-fe3a3abad311
    github.com/gabriel-vasile/mimetype v1.4.2
    github.com/gin-contrib/sse v0.1.0
    github.com/go-playground/locales v0.14.1
    github.com/go-playground/universal-translator v0.18.1
    github.com/go-playground/validator/v10 v10.14.0
    // ... more indirect dependencies
)
```

**Dependency Analysis**:
- **Gin Framework**: Fast HTTP router with middleware support
- **Godotenv**: Development environment configuration
- **Crypto**: Secure password hashing with bcrypt
- **GORM**: Database ORM with relationship management
- **PostgreSQL Driver**: Production-ready database connectivity

### 📄 `config.example.env` - Environment Configuration Template

```bash
# =============================================================================
# Luna IoT Server Configuration
# =============================================================================
# Copy this file to .env and modify values for your environment

# Database Configuration
# =============================================================================
DB_HOST=localhost                    # PostgreSQL server hostname/IP
DB_PORT=5432                        # PostgreSQL port (default: 5432)
DB_USER=postgres                    # Database username
DB_PASSWORD=your_secure_password    # Database password (change this!)
DB_NAME=luna_iot                    # Database name
DB_SSL_MODE=disable                 # SSL mode: disable/require/verify-full

# Server Configuration
# =============================================================================
HTTP_PORT=8080                      # REST API server port
TCP_PORT=5000                       # IoT device connection port

# Optional: Application Settings
# =============================================================================
LOG_LEVEL=info                      # Logging verbosity: debug/info/warn/error
LOG_HTTP=false                      # Enable detailed HTTP request logging
LOG_SQL=false                       # Enable SQL query logging

# Optional: Performance Settings
# =============================================================================
MAX_TCP_CONNECTIONS=1000            # Maximum concurrent TCP connections
DB_MAX_OPEN_CONNS=25               # Maximum database connections
DB_MAX_IDLE_CONNS=5                # Maximum idle database connections
```

**Configuration Sections**:
- **Database**: All PostgreSQL connection parameters
- **Servers**: Port configuration for both TCP and HTTP servers
- **Logging**: Debug and monitoring controls
- **Performance**: Connection limits and optimization settings

---

## 📂 `cmd/` Directory - Entry Points

### 📄 `cmd/http-server/main.go` - Standalone HTTP Server

**Purpose**: HTTP-only server for web API without IoT device support.

```go
package main

import (
    "log"
    "os"
    
    "github.com/joho/godotenv"
    "luna_iot_server/internal/db"
    "luna_iot_server/internal/http"
    "luna_iot_server/pkg/colors"
)

func main() {
    colors.PrintBanner()
    
    // Load environment configuration
    if err := godotenv.Load(); err != nil {
        colors.PrintWarning("No .env file found, using system environment variables")
    }
    
    // Initialize database
    if err := db.Initialize(); err != nil {
        colors.PrintError("Failed to initialize database: %v", err)
        log.Fatalf("Failed to initialize database: %v", err)
    }
    defer db.Close()
    
    // Get HTTP port from environment
    port := os.Getenv("HTTP_PORT")
    if port == "" {
        port = "8080"
    }
    
    // Create HTTP server without control controller
    server := http.NewServer(port)
    
    colors.PrintServer("🌐", "Luna IoT HTTP Server starting on port %s", port)
    colors.PrintEndpoints()  // Display available API endpoints
    
    // Start server
    if err := server.Start(); err != nil {
        colors.PrintError("Failed to start HTTP server: %v", err)
        log.Fatalf("Failed to start HTTP server: %v", err)
    }
}
```

**Use Cases**:
- **Microservice Deployment**: API-only service in distributed architecture
- **Development**: Testing REST endpoints without IoT devices
- **Load Balancing**: Multiple HTTP instances behind load balancer
- **Web-only Applications**: Dashboard and management interfaces

### 📄 `cmd/tcp-server/main.go` - Standalone TCP Server

**Purpose**: TCP-only server for IoT device communication without web API.

```go
package main

import (
    "log"
    "os"
    
    "github.com/joho/godotenv"
    "luna_iot_server/internal/db"
    "luna_iot_server/internal/tcp"
    "luna_iot_server/pkg/colors"
)

func main() {
    colors.PrintBanner()
    
    // Environment and database setup
    if err := godotenv.Load(); err != nil {
        colors.PrintWarning("No .env file found")
    }
    
    if err := db.Initialize(); err != nil {
        log.Fatalf("Failed to initialize database: %v", err)
    }
    defer db.Close()
    
    // Get TCP port from environment
    port := os.Getenv("TCP_PORT")
    if port == "" {
        port = "5000"
    }
    
    // Create TCP server
    server := tcp.NewServer(port)
    
    colors.PrintServer("📡", "Luna IoT TCP Server starting on port %s", port)
    colors.PrintInfo("Waiting for GT06 device connections...")
    
    // Start server
    if err := server.Start(); err != nil {
        log.Fatalf("Failed to start TCP server: %v", err)
    }
}
```

**Use Cases**:
- **Edge Computing**: IoT gateway at remote locations
- **High Throughput**: Dedicated device connection handling
- **Protocol Development**: Testing GT06 protocol implementations
- **Specialized Deployments**: Custom IoT infrastructure

---

## 📂 `config/` Directory - Configuration

### 📄 `config/database.go` - Database Configuration Management

**Purpose**: Centralized database configuration with environment variable support.

```go
package config

import (
    "fmt"
    "os"
)

// DatabaseConfig holds all database connection parameters
type DatabaseConfig struct {
    Host     string    // Database server hostname or IP address
    Port     string    // Database port number (usually 5432 for PostgreSQL)
    User     string    // Username for database authentication
    Password string    // Password for database authentication
    Role     string    // PostgreSQL role for role-based access control
    DBName   string    // Name of the database to connect to
    SSLMode  string    // SSL connection mode (disable/require/verify-full)
}
```

**Configuration Structure**:
- **Host**: Supports both IP addresses and hostnames
- **Port**: String format for flexibility (can include custom ports)
- **Role**: PostgreSQL-specific role-based access control
- **SSLMode**: Security configuration for production deployments

```go
// GetDatabaseConfig creates database configuration from environment variables
func GetDatabaseConfig() *DatabaseConfig {
    return &DatabaseConfig{
        Host:     getEnv("DB_HOST", "84.247.131.246"),         // Production server default
        Port:     getEnv("DB_PORT", "5433"),                   // Custom port
        User:     getEnv("DB_USER", "luna"),                   // Application user
        Role:     getEnv("DB_ROLE", "luna"),                   // PostgreSQL role
        Password: getEnv("DB_PASSWORD", "Luna@#$321"),         // Secure default
        DBName:   getEnv("DB_NAME", "luna_iot"),               // Database name
        SSLMode:  getEnv("DB_SSL_MODE", "disable"),            // Development default
    }
}
```

**Environment Integration**:
- Production-ready defaults for immediate deployment
- Each parameter can be overridden via environment variables
- Secure password pattern following complexity requirements
- SSL disabled for development, can be enabled for production

```go
// GetDSN returns PostgreSQL connection string
func (c *DatabaseConfig) GetDSN() string {
    return fmt.Sprintf("host=%s port=%s user=%s role=%s password=%s dbname=%s sslmode=%s",
        c.Host, c.Port, c.User, c.Role, c.Password, c.DBName, c.SSLMode)
}
```

**DSN Generation**:
- Creates properly formatted PostgreSQL connection string
- Includes role parameter for PostgreSQL role-based security
- Compatible with GORM PostgreSQL driver requirements

```go
// getEnv retrieves environment variable with fallback default
func getEnv(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}
```

**Utility Function**:
- Simple environment variable accessor with defaults
- Prevents null/empty value issues
- Consistent behavior across all configuration parameters

---

## 📂 `internal/db/` - Database Layer

### 📄 `internal/db/connection.go` - Database Connection & Migrations

**Purpose**: Manages PostgreSQL database connections and handles automatic schema migrations.

```go
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
```

**Global Database Instance**:
- Single global `*gorm.DB` instance shared across entire application
- Connection pooling handled automatically by GORM
- Thread-safe access for concurrent operations

```go
// Initialize establishes database connection and runs migrations
func Initialize() error {
    dbConfig := config.GetDatabaseConfig()
    dsn := dbConfig.GetDSN()
    colors.PrintDebug("Connecting to database...")

    var err error
    DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Silent),  // Reduce SQL noise
    })

    if err != nil {
        return fmt.Errorf("failed to connect to database: %v", err)
    }

    colors.PrintSuccess("Database connection established successfully")

    // Run migrations automatically
    if err := RunMigrations(); err != nil {
        return fmt.Errorf("failed to run migrations: %v", err)
    }

    return nil
}
```

**Connection Initialization**:
- Uses configuration from `config` package
- PostgreSQL driver with GORM ORM
- Silent logging mode to reduce console noise
- Automatic migration execution on startup

```go
// RunMigrations handles database schema migrations intelligently
func RunMigrations() error {
    colors.PrintSubHeader("Running Database Migrations")

    // Smart migration: detect if tables need reset due to constraint issues
    shouldReset := false

    if DB.Migrator().HasTable(&models.Vehicle{}) {
        // Test query to detect constraint problems
        var count int64
        err := DB.Model(&models.Vehicle{}).Count(&count).Error
        if err != nil && (strings.Contains(err.Error(), "does not exist") || 
                         strings.Contains(err.Error(), "constraint")) {
            shouldReset = true
            colors.PrintWarning("Detected schema conflicts, will reset tables...")
        }
    }
```

**Intelligent Migration Strategy**:
- Detects existing tables and schema conflicts
- Only resets tables when necessary to preserve data
- Handles foreign key constraint issues gracefully

```go
    if shouldReset {
        // Drop tables in reverse dependency order to handle foreign keys
        colors.PrintInfo("Resetting database schema...")
        
        if DB.Migrator().HasTable(&models.GPSData{}) {
            colors.PrintInfo("Dropping gps_data table...")
            DB.Migrator().DropTable(&models.GPSData{})
        }

        if DB.Migrator().HasTable(&models.Vehicle{}) {
            colors.PrintInfo("Dropping vehicles table...")
            DB.Migrator().DropTable(&models.Vehicle{})
        }

        if DB.Migrator().HasTable(&models.Device{}) {
            colors.PrintInfo("Dropping devices table...")
            DB.Migrator().DropTable(&models.Device{})
        }

        if DB.Migrator().HasTable(&models.User{}) {
            colors.PrintInfo("Dropping users table...")
            DB.Migrator().DropTable(&models.User{})
        }
    }
```

**Table Dropping Logic**:
- Reverse dependency order: GPS_Data → Vehicle → Device → User
- Prevents foreign key constraint violations
- Only drops tables that exist

```go
    // Create/update tables in correct dependency order
    colors.PrintInfo("Creating/updating database schema...")

    // Base tables first (no foreign key dependencies)
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

    // Tables with foreign key dependencies
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

    colors.PrintHeader("DATABASE MIGRATIONS COMPLETED SUCCESSFULLY")
    return nil
}
```

**Table Creation Order**:
- Independent tables first: User, Device
- Dependent tables last: Vehicle (references Device), GPSData (references both)
- Clear success feedback for each table
- Handles migration failures gracefully

```go
// GetDB provides safe access to database instance
func GetDB() *gorm.DB {
    return DB
}

// Close properly closes database connection
func Close() error {
    sqlDB, err := DB.DB()
    if err != nil {
        return err
    }
    return sqlDB.Close()
}
```

**Utility Functions**:
- **GetDB()**: Thread-safe access to global database instance
- **Close()**: Proper connection cleanup for graceful shutdown

---

## 📂 `internal/models/` - Data Models

### 📄 `internal/models/user.go` - User Management Model

**Purpose**: Handles system users with authentication, roles, and security.

```go
package models

import (
    "time"
    "golang.org/x/crypto/bcrypt"
    "gorm.io/gorm"
)

// UserRole represents user permission levels
type UserRole int

const (
    UserRoleAdmin  UserRole = 0 // Full system access
    UserRoleClient UserRole = 1 // Limited client access
)
```

**Role-Based Access Control**:
- Integer enum for database efficiency
- Admin (0): Full system management capabilities
- Client (1): Limited to assigned vehicles/devices
- Easily extensible for additional roles

```go
// User model with comprehensive fields and security
type User struct {
    ID        uint           `json:"id" gorm:"primarykey"`
    Name      string         `json:"name" gorm:"size:100;not null" validate:"required,min=2,max=100"`
    Phone     string         `json:"phone" gorm:"size:15;uniqueIndex" validate:"required,min=10,max=15"`
    Email     string         `json:"email" gorm:"size:100;uniqueIndex" validate:"required,email"`
    Password  string         `json:"-" gorm:"size:255;not null" validate:"required,min=6"`
    Role      UserRole       `json:"role" gorm:"type:integer;not null;default:1" validate:"required,oneof=0 1"`
    Image     string         `json:"image" gorm:"size:255"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
```

**User Model Features**:
- **Primary Key**: Auto-incrementing ID
- **Unique Constraints**: Phone and email with database indexes
- **Security**: Password excluded from JSON serialization (`json:"-"`)
- **Validation**: Comprehensive validation tags for input validation
- **Soft Delete**: `DeletedAt` field preserves user history
- **Timestamps**: Automatic creation and update tracking

```go
// BeforeCreate hook - automatically hash password on user creation
func (u *User) BeforeCreate(tx *gorm.DB) error {
    if u.Password != "" {
        hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
        if err != nil {
            return err
        }
        u.Password = string(hashedPassword)
    }
    return nil
}

// BeforeUpdate hook - hash password only if changed
func (u *User) BeforeUpdate(tx *gorm.DB) error {
    if tx.Statement.Changed("Password") && u.Password != "" {
        hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
        if err != nil {
            return err
        }
        u.Password = string(hashedPassword)
    }
    return nil
}
```

**Security Hooks**:
- **BeforeCreate**: Automatically hashes password using bcrypt
- **BeforeUpdate**: Only hashes password if it's being changed
- **Change Detection**: Prevents unnecessary hashing on user updates
- **Bcrypt**: Industry-standard password hashing with default cost (10)

```go
// CheckPassword securely verifies user password
func (u *User) CheckPassword(password string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
    return err == nil
}

// GetRoleString returns human-readable role name
func (u *User) GetRoleString() string {
    switch u.Role {
    case UserRoleAdmin:
        return "admin"
    case UserRoleClient:
        return "client"
    default:
        return "unknown"
    }
}
```

**Utility Methods**:
- **CheckPassword**: Secure password verification using bcrypt
- **GetRoleString**: Human-readable role names for display
- Clean API for authentication and user management

### 📄 `internal/models/device.go` - GPS Device Registry

**Purpose**: Manages GPS tracking device information and authentication.

```go
package models

import (
    "time"
    "gorm.io/gorm"
)

// SimOperator enum for mobile network operators
type SimOperator string

const (
    SimOperatorNcell SimOperator = "Ncell"  // Nepal's Ncell network
    SimOperatorNtc   SimOperator = "Ntc"    // Nepal Telecom
)

// Protocol enum for device communication protocols
type Protocol string

const (
    ProtocolGT06 Protocol = "GT06"  // Current supported protocol
    // Future protocols can be added here
)
```

**Type Safety**:
- String-based enums prevent invalid values
- Nepal-specific SIM operators (Ncell, Ntc)
- Extensible protocol support for future devices

```go
// Device model for GPS tracking devices
type Device struct {
    ID          uint           `json:"id" gorm:"primarykey"`
    IMEI        string         `json:"imei" gorm:"uniqueIndex;not null;size:16" validate:"required,len=16"`
    SimNo       string         `json:"sim_no" gorm:"size:20" validate:"required"`
    SimOperator SimOperator    `json:"sim_operator" gorm:"type:varchar(10);not null" validate:"required,oneof=Ncell Ntc"`
    Protocol    Protocol       `json:"protocol" gorm:"type:varchar(10);not null;default:'GT06'" validate:"required"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
    DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
```

**Device Model Features**:
- **IMEI Validation**: Exactly 16 digits (international standard)
- **Unique IMEI**: Database-level uniqueness constraint
- **SIM Information**: Tracks SIM card number and operator
- **Protocol Support**: Currently GT06, easily extensible
- **Soft Delete**: Preserves device history

```go
// BeforeCreate hook for device validation
func (d *Device) BeforeCreate(tx *gorm.DB) error {
    // Custom validation logic can be added here
    // e.g., IMEI format validation, SIM number validation
    return nil
}

// IsValidIMEI checks if IMEI follows international format
func (d *Device) IsValidIMEI() bool {
    return len(d.IMEI) == 16 && d.IMEI != ""
}
```

**Validation Methods**:
- **BeforeCreate**: Extensible validation hook
- **IsValidIMEI**: IMEI format validation
- Foundation for additional business logic

### 📄 `internal/models/vehicle.go` - Fleet Management

**Purpose**: Manages vehicle information and fleet tracking capabilities.

```go
package models

// VehicleType enum for different vehicle categories
type VehicleType string

const (
    VehicleTypeBike      VehicleType = "bike"
    VehicleTypeCar       VehicleType = "car"
    VehicleTypeTruck     VehicleType = "truck"
    VehicleTypeBus       VehicleType = "bus"
    VehicleTypeSchoolBus VehicleType = "school_bus"
)
```

**Vehicle Classification**:
- Comprehensive vehicle type support
- Different tracking profiles per vehicle type
- Easy categorization for fleet management

```go
// Vehicle model with comprehensive fleet management features
type Vehicle struct {
    IMEI        string         `json:"imei" gorm:"primaryKey;size:16;not null" validate:"required,len=16"`
    RegNo       string         `json:"reg_no" gorm:"size:20;uniqueIndex;not null" validate:"required"`
    Name        string         `json:"name" gorm:"size:100;not null" validate:"required"`
    Odometer    float64        `json:"odometer" gorm:"type:decimal(10,2);default:0"`
    Mileage     float64        `json:"mileage" gorm:"type:decimal(5,2)"`
    MinFuel     float64        `json:"min_fuel" gorm:"type:decimal(5,2)"`
    Overspeed   int            `json:"overspeed" gorm:"type:integer;default:60"`
    VehicleType VehicleType    `json:"vehicle_type" gorm:"type:varchar(20);not null" validate:"required"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
    DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

    // Foreign key relationship to Device
    Device Device `json:"device,omitempty" gorm:"foreignKey:IMEI;references:IMEI;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}
```

**Fleet Management Features**:
- **IMEI Primary Key**: Direct link to GPS device
- **Registration Uniqueness**: Unique vehicle registration numbers
- **Odometer Tracking**: Mileage and distance monitoring
- **Fuel Management**: Minimum fuel level alerts
- **Speed Monitoring**: Configurable overspeed limits
- **Device Relationship**: Foreign key with cascade updates

```go
// BeforeCreate hook for vehicle business logic
func (v *Vehicle) BeforeCreate(tx *gorm.DB) error {
    // Set default overspeed limit if not provided
    if v.Overspeed <= 0 {
        v.Overspeed = 60  // 60 km/h default
    }
    return nil
}

// IsOverspeed checks if current speed exceeds limit
func (v *Vehicle) IsOverspeed(currentSpeed int) bool {
    return currentSpeed > v.Overspeed
}

// GetFuelStatus returns fuel level status
func (v *Vehicle) GetFuelStatus(currentFuel float64) string {
    if currentFuel <= v.MinFuel {
        return "Low"
    } else if currentFuel <= v.MinFuel*1.2 {
        return "Warning"
    }
    return "Normal"
}
```

**Business Logic Methods**:
- **BeforeCreate**: Sets sensible defaults
- **IsOverspeed**: Speed limit checking
- **GetFuelStatus**: Fuel level monitoring
- Foundation for fleet management rules

### 📄 `internal/models/gps_data.go` - GPS Tracking Data

**Purpose**: Stores comprehensive GPS tracking data and device status information.

```go
package models

import (
    "fmt"
    "time"
    "gorm.io/gorm"
)

// GPSData model - comprehensive tracking data storage
type GPSData struct {
    ID        uint      `json:"id" gorm:"primarykey"`
    IMEI      string    `json:"imei" gorm:"size:16;not null;index"`
    Timestamp time.Time `json:"timestamp" gorm:"not null;index"`

    // GPS Location Data
    Latitude  *float64  `json:"latitude" gorm:"type:decimal(10,7)"`   // 7 decimal precision
    Longitude *float64  `json:"longitude" gorm:"type:decimal(10,7)"`  // ~1cm accuracy
    Speed     *int      `json:"speed"`     // Speed in km/h
    Course    *int      `json:"course"`    // Direction in degrees (0-359)
    Altitude  *int      `json:"altitude"`  // Altitude in meters

    // GPS Status Information
    GPSRealTime   *bool `json:"gps_real_time"`   // Real-time GPS fix
    GPSPositioned *bool `json:"gps_positioned"`  // GPS positioning active
    Satellites    *int  `json:"satellites"`      // Number of GPS satellites

    // Device Status
    Ignition       string `json:"ignition"`        // ON/OFF
    Charger        string `json:"charger"`         // CONNECTED/DISCONNECTED
    GPSTracking    string `json:"gps_tracking"`    // ENABLED/DISABLED
    OilElectricity string `json:"oil_electricity"` // CONNECTED/DISCONNECTED
    DeviceStatus   string `json:"device_status"`   // ACTIVATED/DEACTIVATED

    // Power and Signal Information
    VoltageLevel  *int   `json:"voltage_level"`   // Battery voltage level (0-100)
    VoltageStatus string `json:"voltage_status"`  // Battery status description
    GSMSignal     *int   `json:"gsm_signal"`      // GSM signal strength (0-100)
    GSMStatus     string `json:"gsm_status"`      // Signal status description

    // LBS (Location Based Services) Data - Cell Tower Information
    MCC    *int `json:"mcc"`     // Mobile Country Code
    MNC    *int `json:"mnc"`     // Mobile Network Code
    LAC    *int `json:"lac"`     // Location Area Code
    CellID *int `json:"cell_id"` // Cell Tower ID

    // Alarm and Alert System
    AlarmActive bool   `json:"alarm_active"`  // Is alarm currently active
    AlarmType   string `json:"alarm_type"`    // Type of alarm
    AlarmCode   int    `json:"alarm_code"`    // Numeric alarm code

    // Raw Protocol Data (for debugging and analysis)
    ProtocolName string `json:"protocol_name"` // Protocol used (GT06, etc.)
    RawPacket    string `json:"raw_packet"`    // Original hex packet data

    // Database Relationships
    Device  Device  `json:"device,omitempty" gorm:"foreignKey:IMEI;references:IMEI"`
    Vehicle Vehicle `json:"vehicle,omitempty" gorm:"foreignKey:IMEI;references:IMEI"`
}
```

**Comprehensive Data Model**:
- **Location**: High-precision GPS coordinates with 7 decimal places
- **Movement**: Speed, direction, and altitude tracking
- **Device Status**: Complete device state monitoring
- **Power Management**: Battery and charging status
- **Network Info**: Cell tower location data for backup positioning
- **Alarm System**: Emergency and alert handling
- **Raw Data**: Complete packet preservation for analysis

```go
// IsValidLocation checks if GPS coordinates are valid
func (g *GPSData) IsValidLocation() bool {
    if g.Latitude == nil || g.Longitude == nil {
        return false
    }
    
    // Validate coordinate ranges
    lat, lng := *g.Latitude, *g.Longitude
    return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

// GetLocationString returns formatted location string
func (g *GPSData) GetLocationString() string {
    if !g.IsValidLocation() {
        return "No valid location"
    }
    return fmt.Sprintf("%.6f,%.6f", *g.Latitude, *g.Longitude)
}

// GetDistanceFromPoint calculates distance to another point in kilometers
func (g *GPSData) GetDistanceFromPoint(lat, lng float64) float64 {
    if !g.IsValidLocation() {
        return -1
    }
    
    // Haversine formula for calculating distance between two points
    const earthRadius = 6371 // Earth's radius in kilometers
    
    lat1, lng1 := *g.Latitude, *g.Longitude
    lat2, lng2 := lat, lng
    
    // Convert degrees to radians
    lat1Rad := lat1 * math.Pi / 180
    lng1Rad := lng1 * math.Pi / 180
    lat2Rad := lat2 * math.Pi / 180
    lng2Rad := lng2 * math.Pi / 180
    
    // Haversine formula
    deltaLat := lat2Rad - lat1Rad
    deltaLng := lng2Rad - lng1Rad
    
    a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) + 
         math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
    c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
    
    return earthRadius * c
}

// IsMoving determines if device is currently moving
func (g *GPSData) IsMoving() bool {
    return g.Speed != nil && *g.Speed > 0
}

// GetAlarmSeverity returns alarm severity level
func (g *GPSData) GetAlarmSeverity() string {
    if !g.AlarmActive {
        return "None"
    }
    
    switch g.AlarmCode {
    case 1, 2, 3:
        return "Critical"
    case 4, 5, 6:
        return "High"
    case 7, 8, 9:
        return "Medium"
    default:
        return "Low"
    }
}
```

**Utility Methods**:
- **Location Validation**: Ensures GPS coordinates are within valid ranges
- **Distance Calculation**: Haversine formula for accurate distance measurement
- **Movement Detection**: Determines if device is stationary or moving
- **Alarm Management**: Categorizes alarm severity levels
- **Formatted Output**: Human-readable location strings

---

This detailed documentation continues with the protocol implementation, HTTP controllers, TCP server, and utilities. Would you like me to continue with the remaining sections? 
