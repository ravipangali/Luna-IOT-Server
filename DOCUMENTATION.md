# Luna IoT Server - Complete Technical Documentation

## 📋 Table of Contents

- [Luna IoT Server - Complete Technical Documentation](#luna-iot-server---complete-technical-documentation)
  - [📋 Table of Contents](#-table-of-contents)
  - [📖 Project Overview](#-project-overview)
    - [🎯 Key Features](#-key-features)
  - [🏗️ Architecture Overview](#️-architecture-overview)
  - [📁 Directory Structure Analysis](#-directory-structure-analysis)
  - [🚀 Core Application Files](#-core-application-files)
    - [📄 `main.go` - Unified Server Entry Point](#-maingo---unified-server-entry-point)
  - [📂 internal/protocol/ - Protocol Implementation](#-internalprotocol---protocol-implementation)
    - [📄 `internal/protocol/gt06_decoder.go` - GT06 Protocol Decoder](#-internalprotocolgt06_decodergo---gt06-protocol-decoder)
      - [Data Processing Pipeline](#data-processing-pipeline)
      - [GPS Data Decoding](#gps-data-decoding)
      - [Status Information Decoding](#status-information-decoding)
    - [📄 `internal/protocol/gps_tracker_control.go` - Device Control System](#-internalprotocolgps_tracker_controlgo---device-control-system)
      - [Oil/Electricity Control](#oilelectricity-control)
      - [Command Transmission](#command-transmission)
      - [Response Analysis](#response-analysis)
  - [📂 internal/http/ - HTTP Server Implementation](#-internalhttp---http-server-implementation)
    - [📄 `internal/http/server.go` - HTTP Server Setup](#-internalhttpservergo---http-server-setup)
    - [📄 `internal/http/routes.go` - API Route Configuration](#-internalhttproutesgo---api-route-configuration)
  - [📂 internal/http/controllers/ - HTTP Controllers](#-internalhttpcontrollers---http-controllers)
    - [📄 `control_controller.go` - Device Control Controller](#-control_controllergo---device-control-controller)
    - [📄 `device_controller.go` - Device Management Controller](#-device_controllergo---device-management-controller)
    - [📄 `gps_controller.go` - GPS Data Controller](#-gps_controllergo---gps-data-controller)
  - [📂 internal/tcp/ - TCP Server Implementation](#-internaltcp---tcp-server-implementation)
    - [📄 `internal/tcp/server.go` - TCP Server](#-internaltcpservergo---tcp-server)
  - [📂 pkg/colors/ - Utility Package](#-pkgcolors---utility-package)
    - [📄 `pkg/colors/colors.go` - Console Output Formatting](#-pkgcolorscolorsgo---console-output-formatting)
  - [📄 Configuration Files](#-configuration-files)
    - [`config.example.env` - Environment Configuration Template](#configexampleenv---environment-configuration-template)
    - [`setup.sql` - Database Schema Setup](#setupsql---database-schema-setup)
  - [🚀 Quick Start Guide](#-quick-start-guide)
    - [Installation Steps](#installation-steps)
    - [API Usage Examples](#api-usage-examples)

---

## 📖 Project Overview

**Luna IoT Server** is a professional-grade GPS tracking and fleet management system built with Go. It's designed to handle GT06 protocol-based GPS tracking devices, providing real-time location tracking, vehicle management, and remote device control capabilities.

### 🎯 Key Features
- **Dual Server Architecture**: Concurrent TCP and HTTP servers
- **Real-time GPS Tracking**: Live device location monitoring
- **Remote Device Control**: Oil/electricity cut-off commands
- **Fleet Management**: Vehicle and user management system
- **Protocol Support**: Complete GT06 protocol implementation
- **Database Integration**: PostgreSQL with GORM ORM
- **RESTful API**: Comprehensive REST endpoints
- **Connection Management**: Active device connection tracking

---

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Luna IoT Server                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐              ┌─────────────────┐          │
│  │   TCP Server    │              │   HTTP Server   │          │
│  │   (Port 5000)   │              │   (Port 8080)   │          │
│  │                 │              │                 │          │
│  │ • GT06 Protocol │              │ • REST API      │          │
│  │ • Device Auth   │              │ • CRUD Ops      │          │
│  │ • GPS Data      │              │ • Device Control│          │
│  │ • Status Update │              │ • Web Interface │          │
│  └─────────────────┘              └─────────────────┘          │
│           │                                │                   │
│           └────────────────┬───────────────┘                   │
│                            │                                   │
│  ┌─────────────────────────▼─────────────────────────┐         │
│  │              Shared Components                    │         │
│  │                                                   │         │
│  │ • Control Controller (Connection Management)      │         │
│  │ • GT06 Protocol Decoder                          │         │
│  │ • Database Layer (GORM)                          │         │
│  │ • Models & Business Logic                        │         │
│  │ • Configuration Management                       │         │
│  └─────────────────────────┬─────────────────────────┘         │
│                            │                                   │
└────────────────────────────┼───────────────────────────────────┘
                             │
                    ┌────────▼────────┐
                    │   PostgreSQL    │
                    │    Database     │
                    │                 │
                    │ • Users         │
                    │ • Devices       │
                    │ • Vehicles      │
                    │ • GPS Data      │
                    └─────────────────┘
```

---

## 📁 Directory Structure Analysis

```
luna_iot_server/
├── 📂 cmd/                         # Application entry points
│   ├── 📂 http-server/             # Standalone HTTP server
│   │   └── 📄 main.go              # HTTP-only server entry point
│   └── 📂 tcp-server/              # Standalone TCP server
│       └── 📄 main.go              # TCP-only server entry point
│
├── 📂 config/                      # Configuration management
│   └── 📄 database.go              # Database configuration utilities
│
├── 📂 internal/                    # Private application code
│   ├── 📂 db/                      # Database layer
│   │   └── 📄 connection.go        # DB connection & migrations
│   │
│   ├── 📂 http/                    # HTTP server components
│   │   ├── 📂 controllers/         # Request handlers
│   │   │   ├── 📄 control_controller.go    # Device control operations
│   │   │   ├── 📄 device_controller.go     # Device management CRUD
│   │   │   ├── 📄 gps_controller.go        # GPS data operations
│   │   │   ├── 📄 user_controller.go       # User management CRUD
│   │   │   └── 📄 vehicle_controller.go    # Vehicle management CRUD
│   │   ├── 📄 routes.go            # API route definitions
│   │   └── 📄 server.go            # HTTP server setup
│   │
│   ├── 📂 models/                  # Database models
│   │   ├── 📄 device.go            # GPS device model
│   │   ├── 📄 gps_data.go          # GPS tracking data model
│   │   ├── 📄 user.go              # User model
│   │   └── 📄 vehicle.go           # Vehicle model
│   │
│   ├── 📂 protocol/                # Protocol implementations
│   │   ├── 📄 gt06_decoder.go      # GT06 protocol decoder
│   │   └── 📄 gps_tracker_control.go # Device control commands
│   │
│   └── 📂 tcp/                     # TCP server components
│       └── 📄 server.go            # TCP server implementation
│
├── 📂 pkg/                         # Public packages
│   └── 📂 colors/                  # Console output utilities
│       └── 📄 colors.go            # Colored console output functions
│
├── 📄 config.example.env           # Environment configuration template
├── 📄 DEPLOYMENT.md               # Deployment instructions
├── 📄 go.mod                      # Go module definition
├── 📄 go.sum                      # Go module checksums
├── 📄 main.go                     # Unified server entry point
├── 📄 README.md                   # Project README
└── 📄 setup.sql                   # Database schema setup
```

---

## 🚀 Core Application Files

### 📄 `main.go` - Unified Server Entry Point

**Purpose**: Primary application entry point that orchestrates both TCP and HTTP servers

**Key Responsibilities**:
- Environment configuration loading
- Database initialization
- Shared component creation
- Concurrent server startup
- Graceful shutdown handling

**Code Breakdown**:

```go
func main() {
    // 1. Print application banner
    colors.PrintBanner()
    
    // 2. Load environment variables
    if err := godotenv.Load(); err != nil {
        colors.PrintWarning("No .env file found, using system environment variables")
    }
    
    // 3. Initialize database
    if err := db.Initialize(); err != nil {
        log.Fatalf("Database initialization failed: %v", err)
    }
    defer db.Close()
    
    // 4. Create shared control controller
    sharedControlController := controllers.NewControlController()
    
    // 5. Start servers concurrently
    var wg sync.WaitGroup
    errorChan := make(chan error, 2)
    
    // TCP Server goroutine
    wg.Add(1)
    go func() {
        defer wg.Done()
        tcpServer := tcp.NewServerWithController(tcpPort, sharedControlController)
        if err := tcpServer.Start(); err != nil {
            errorChan <- fmt.Errorf("TCP server error: %v", err)
        }
    }()
    
    // HTTP Server goroutine
    wg.Add(1)
    go func() {
        defer wg.Done()
        httpServer := http.NewServerWithController(httpPort, sharedControlController)
        if err := httpServer.Start(); err != nil {
            errorChan <- fmt.Errorf("HTTP server error: %v", err)
        }
    }()
    
    // 6. Handle shutdown signals
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    
    select {
    case err := <-errorChan:
        colors.PrintError("Server startup failed: %v", err)
    case <-quit:
        colors.PrintShutdown()
    }
}
```

**Key Features**:
- **Concurrent Execution**: Both servers run simultaneously
- **Shared State**: Control controller shared between servers
- **Error Handling**: Comprehensive error management
- **Graceful Shutdown**: Proper cleanup on termination

---

## 📂 internal/protocol/ - Protocol Implementation

### 📄 `internal/protocol/gt06_decoder.go` - GT06 Protocol Decoder

**Purpose**: Complete implementation of GT06 protocol for GPS tracking devices

**GT06 Protocol Overview**:
The GT06 protocol is a binary communication protocol used by GPS tracking devices. It defines packet structure, data encoding, and command formats.

**Packet Structure**:
```
┌─────────┬────────┬──────────┬─────────┬──────────┬─────────┬─────────┐
│ Start   │ Length │ Protocol │ Data    │ Serial   │ CRC     │ Stop    │
│ (2B)    │ (1B)   │ (1B)     │ (nB)    │ (2B)     │ (2B)    │ (2B)    │
├─────────┼────────┼──────────┼─────────┼──────────┼─────────┼─────────┤
│ 78 78   │ XX     │ XX       │ ...     │ XX XX    │ XX XX   │ 0D 0A   │
│ 79 79   │        │          │         │          │         │         │
└─────────┴────────┴──────────┴─────────┴──────────┴─────────┴─────────┘
```

**Key Structures**:

```go
// Main decoder structure
type GT06Decoder struct {
    buffer           []byte              // Data buffer
    startBits78      []byte              // Standard packets (78 78)
    startBits79      []byte              // Extended packets (79 79)
    stopBits         []byte              // Stop bits (0D 0A)
    protocolNumbers  map[byte]string     // Protocol number mappings
    responseRequired []byte              // Protocols requiring responses
}

// Decoded packet result
type DecodedPacket struct {
    Raw           string      `json:"raw"`
    Timestamp     time.Time   `json:"timestamp"`
    Length        byte        `json:"length"`
    Protocol      byte        `json:"protocol"`
    ProtocolName  string      `json:"protocolName"`
    SerialNumber  byte        `json:"serialNumber"`
    Checksum      byte        `json:"checksum"`
    NeedsResponse bool        `json:"needsResponse"`
    
    // Login data
    TerminalID     string  `json:"terminalId,omitempty"`
    DeviceType     *uint16 `json:"deviceType,omitempty"`
    TimezoneOffset *int16  `json:"timezoneOffset,omitempty"`
    
    // GPS data
    GPSTime       *time.Time `json:"gpsTime,omitempty"`
    Latitude      *float64   `json:"latitude,omitempty"`
    Longitude     *float64   `json:"longitude,omitempty"`
    Speed         *byte      `json:"speed,omitempty"`
    Course        *uint16    `json:"course,omitempty"`
    
    // Device status
    Ignition      string     `json:"ignition,omitempty"`
    Charger       string     `json:"charger,omitempty"`
    Voltage       *VoltageInfo `json:"voltage,omitempty"`
    GSMSignal     *GSMInfo   `json:"gsmSignal,omitempty"`
    // ... more fields
}
```

**Protocol Numbers**:
- **0x01 LOGIN**: Device authentication and identification
- **0x12 GPS_LBS_STATUS**: GPS coordinates with cell tower info
- **0x13 STATUS_INFO**: Device status (ignition, battery, signals)
- **0x16 ALARM_DATA**: Emergency alerts and alarms
- **0x1A GPS_LBS_DATA**: Alternative GPS data format
- **0xA0 GPS_LBS_STATUS_A0**: Extended GPS format

**Key Processing Functions**:

#### Data Processing Pipeline
```go
func (d *GT06Decoder) AddData(data []byte) ([]*DecodedPacket, error) {
    // 1. Clear previous buffer
    d.clearBuffer()
    
    // 2. Add new data to buffer
    d.buffer = append(d.buffer, data...)
    
    // 3. Process complete packets
    return d.processBuffer()
}

func (d *GT06Decoder) processBuffer() ([]*DecodedPacket, error) {
    var packets []*DecodedPacket
    
    for len(d.buffer) >= 5 {
        // Find start bits (78 78 or 79 79)
        startInfo := d.findStartBits()
        if startInfo.Index == -1 {
            break
        }
        
        // Extract packet length
        lengthByte := d.buffer[2]
        totalLength := int(lengthByte) + 5
        
        // Verify complete packet
        if len(d.buffer) < totalLength {
            break
        }
        
        // Extract and decode packet
        packet := d.buffer[0:totalLength]
        if packet[totalLength-2] == 0x0D && packet[totalLength-1] == 0x0A {
            decoded, err := d.decodePacket(packet)
            if err == nil && decoded != nil {
                packets = append(packets, decoded)
            }
        }
        
        // Remove processed packet from buffer
        d.buffer = d.buffer[totalLength:]
    }
    
    return packets, nil
}
```

#### GPS Data Decoding
```go
func (d *GT06Decoder) decodeGPSLBS(data []byte, result *DecodedPacket) {
    offset := 0
    
    // 1. Decode timestamp (6 bytes: YY MM DD HH MM SS)
    if len(data) >= 6 {
        year := 2000 + int(data[offset])
        month := int(data[offset+1])
        day := int(data[offset+2])
        hour := int(data[offset+3])
        minute := int(data[offset+4])
        second := int(data[offset+5])
        
        gpsTime := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC)
        result.GPSTime = &gpsTime
        offset += 6
    }
    
    // 2. Decode GPS coordinates
    if result.Protocol == 0xA0 {
        // Extended format (A0 protocol)
        if offset+12 <= len(data) {
            // Latitude (4 bytes, degrees * 1800000)
            latRaw := binary.BigEndian.Uint32(data[offset:offset+4])
            lat := float64(latRaw) / 1800000.0
            result.Latitude = &lat
            offset += 4
            
            // Longitude (4 bytes, degrees * 1800000)
            lngRaw := binary.BigEndian.Uint32(data[offset:offset+4])
            lng := float64(lngRaw) / 1800000.0
            result.Longitude = &lng
            offset += 4
            
            // Speed and course
            speed := data[offset]
            result.Speed = &speed
            
            courseStatus := binary.BigEndian.Uint16(data[offset+1:offset+3])
            course := courseStatus & 0x03FF
            result.Course = &course
            
            // Status flags
            gpsRealTime := (courseStatus & 0x2000) == 0
            result.GPSRealTime = &gpsRealTime
            
            gpsPositioned := (courseStatus & 0x1000) == 0
            result.GPSPositioned = &gpsPositioned
        }
    }
    
    // 3. Decode LBS (Location Based Services) data
    if offset+9 <= len(data) {
        mcc := binary.BigEndian.Uint16(data[offset:offset+2])     // Mobile Country Code
        result.MCC = &mcc
        
        mnc := data[offset+2]                                      // Mobile Network Code
        result.MNC = &mnc
        
        lac := binary.BigEndian.Uint16(data[offset+3:offset+5])   // Location Area Code
        result.LAC = &lac
        
        // Cell ID (3 bytes)
        cellId := (uint32(data[offset+5]) << 16) | 
                 (uint32(data[offset+6]) << 8) | 
                 uint32(data[offset+7])
        result.CellID = &cellId
    }
}
```

#### Status Information Decoding
```go
func (d *GT06Decoder) decodeStatusInfo(data []byte, result *DecodedPacket) {
    if len(data) < 3 {
        return
    }
    
    // Terminal Information Byte analysis
    terminalInfoByte := data[0]
    
    // Extract status bits
    // Bit 1 (0x02): ACC/Ignition status
    if (terminalInfoByte & 0x02) != 0 {
        result.Ignition = "ON"
    } else {
        result.Ignition = "OFF"
    }
    
    // Bit 2 (0x04): Charging status
    if (terminalInfoByte & 0x04) != 0 {
        result.Charger = "CONNECTED"
    } else {
        result.Charger = "DISCONNECTED"
    }
    
    // Bit 6 (0x40): GPS tracking
    if (terminalInfoByte & 0x40) != 0 {
        result.GPSTracking = "ENABLED"
    } else {
        result.GPSTracking = "DISABLED"
    }
    
    // Bit 7 (0x80): Oil/Electricity status
    if (terminalInfoByte & 0x80) == 0 {
        result.OilElectricity = "CONNECTED"
    } else {
        result.OilElectricity = "DISCONNECTED"
    }
    
    // Voltage level (second byte)
    voltageLevel := data[1]
    result.Voltage = &VoltageInfo{
        Level:      voltageLevel,
        Status:     d.getVoltageStatus(voltageLevel),
        Percentage: d.getVoltagePercentage(voltageLevel),
    }
    
    // GSM signal strength (third byte)
    gsmLevel := data[2]
    result.GSMSignal = &GSMInfo{
        Level:  gsmLevel,
        Status: d.getGsmStatus(gsmLevel),
        Bars:   min(int(gsmLevel), 4),
    }
}
```

### 📄 `internal/protocol/gps_tracker_control.go` - Device Control System

**Purpose**: Send control commands to connected GPS devices

**Control Commands**:
- **DYD#**: Cut oil and electricity (disable vehicle)
- **HFYD#**: Connect oil and electricity (enable vehicle)
- **DWXX#**: Get location information

**Command Packet Structure**:
```go
type ControlPacket struct {
    StartBit         uint16    // Always 0x7878
    PacketLength     uint8     // Total packet length
    ProtocolNumber   uint8     // 0x80 (Server to terminal)
    CommandLength    uint8     // Command data length
    ServerFlag       [4]byte   // Server identification
    CommandContent   string    // Command string (DYD#, HFYD#, etc.)
    Language         uint16    // Language setting (English/Chinese)
    InfoSerialNumber uint16    // Sequence number
    ErrorCheck       uint16    // Checksum
    StopBit          uint16    // Always 0x0D0A
}

type GPSTrackerController struct {
    conn         net.Conn    // TCP connection to device
    serverFlag   [4]byte     // Server identification
    serialNumber uint16      // Command sequence number
    language     uint16      // Language preference
    deviceIMEI   string      // Target device IMEI
}
```

**Command Processing**:

#### Oil/Electricity Control
```go
func (g *GPSTrackerController) CutOilAndElectricity() (*ControlResponse, error) {
    colors.PrintSubHeader("CUTTING OIL AND ELECTRICITY for device %s", g.deviceIMEI)
    
    response, err := g.sendCommand(CmdCutOil) // "DYD#"
    if err != nil {
        return response, fmt.Errorf("failed to cut oil and electricity: %v", err)
    }
    
    if response.Success {
        colors.PrintSuccess("Oil and electricity successfully cut for device %s", g.deviceIMEI)
    } else {
        colors.PrintError("Failed to cut oil and electricity for device %s: %s", 
                         g.deviceIMEI, response.Message)
    }
    
    return response, nil
}

func (g *GPSTrackerController) ConnectOilAndElectricity() (*ControlResponse, error) {
    colors.PrintSubHeader("CONNECTING OIL AND ELECTRICITY for device %s", g.deviceIMEI)
    
    response, err := g.sendCommand(CmdConnectOil) // "HFYD#"
    if err != nil {
        return response, fmt.Errorf("failed to connect oil and electricity: %v", err)
    }
    
    return response, nil
}
```

#### Command Transmission
```go
func (g *GPSTrackerController) sendCommand(command string) (*ControlResponse, error) {
    response := &ControlResponse{
        Command:    command,
        Timestamp:  time.Now(),
        DeviceIMEI: g.deviceIMEI,
    }
    
    // 1. Build control packet
    packet := g.buildControlPacket(command)
    data := g.packetToBytes(packet)
    
    colors.PrintControl("Sending command %s to device %s", command, g.deviceIMEI)
    
    // 2. Send command to device
    _, err := g.conn.Write(data)
    if err != nil {
        response.Success = false
        response.Message = fmt.Sprintf("Failed to send command: %v", err)
        return response, err
    }
    
    // 3. Read response with timeout
    responseData := make([]byte, 256)
    g.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
    n, err := g.conn.Read(responseData)
    if err != nil {
        response.Success = false
        response.Message = fmt.Sprintf("Failed to read response: %v", err)
        return response, err
    }
    
    // 4. Parse response
    responseStr, parseErr := g.parseResponse(responseData[:n])
    response.Response = responseStr
    
    // 5. Determine success/failure
    response.Success = g.isSuccessfulResponse(command, responseStr)
    response.Message = g.getResponseMessage(command, responseStr)
    
    return response, parseErr
}
```

#### Response Analysis
```go
func (g *GPSTrackerController) isSuccessfulResponse(command, response string) bool {
    switch command {
    case CmdCutOil:     // "DYD#"
        return contains(response, "Success")
    case CmdConnectOil: // "HFYD#"
        return contains(response, "Success")
    case CmdLocation:   // "DWXX#"
        return !contains(response, "Fail")
    default:
        return contains(response, "Success")
    }
}

func (g *GPSTrackerController) getResponseMessage(command, response string) string {
    switch command {
    case CmdCutOil:
        switch {
        case contains(response, "Success"):
            return "Oil and electricity successfully cut"
        case contains(response, "Speed Limit"):
            return "Cannot cut oil - vehicle speed too high (>20km/h)"
        case contains(response, "Unvalued Fix"):
            return "Cannot cut oil - GPS tracking is off"
        default:
            return fmt.Sprintf("Unknown response: %s", response)
        }
    case CmdConnectOil:
        switch {
        case contains(response, "Success"):
            return "Oil and electricity successfully connected"
        case contains(response, "Fail"):
            return "Failed to connect oil and electricity"
        default:
            return fmt.Sprintf("Unknown response: %s", response)
        }
    case CmdLocation:
        return fmt.Sprintf("Location response: %s", response)
    default:
        return response
    }
}
```

---

## 📂 internal/http/ - HTTP Server Implementation

### 📄 `internal/http/server.go` - HTTP Server Setup

**Purpose**: Configures and initializes the HTTP REST API server

**Key Components**:

```go
type Server struct {
    router *gin.Engine
    port   string
}

func NewServer(port string) *Server {
    // Set Gin to release mode
    gin.SetMode(gin.ReleaseMode)
    
    // Create router with middleware
    router := gin.Default()
    
    // Conditional middleware
    if os.Getenv("LOG_HTTP") == "true" {
        router.Use(gin.Logger())
    }
    router.Use(gin.Recovery())
    router.Use(CORSMiddleware())
    
    // Setup routes
    SetupRoutes(router)
    
    return &Server{router: router, port: port}
}

// CORS middleware for cross-origin requests
func CORSMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
        c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
        c.Writer.Header().Set("Access-Control-Allow-Headers", 
            "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
        c.Writer.Header().Set("Access-Control-Allow-Methods", 
            "POST, OPTIONS, GET, PUT, DELETE")
        
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        c.Next()
    }
}
```

### 📄 `internal/http/routes.go` - API Route Configuration

**Purpose**: Defines all HTTP endpoints and their mappings

**Route Structure**:
```go
func SetupRoutesWithControlController(router *gin.Engine, sharedControlController *controllers.ControlController) {
    // Initialize controllers
    userController := controllers.NewUserController()
    deviceController := controllers.NewDeviceController()
    vehicleController := controllers.NewVehicleController()
    gpsController := controllers.NewGPSController()
    
    // Use shared control controller
    var controlController *controllers.ControlController
    if sharedControlController != nil {
        controlController = sharedControlController
    } else {
        controlController = controllers.NewControlController()
    }
    
    // API version 1 group
    v1 := router.Group("/api/v1")
    {
        // User management routes
        users := v1.Group("/users")
        {
            users.GET("", userController.GetUsers)           // List users
            users.GET("/:id", userController.GetUser)        // Get user by ID
            users.POST("", userController.CreateUser)        // Create user
            users.PUT("/:id", userController.UpdateUser)     // Update user
            users.DELETE("/:id", userController.DeleteUser)  // Delete user
        }
        
        // Device management routes
        devices := v1.Group("/devices")
        {
            devices.GET("", deviceController.GetDevices)
            devices.GET("/:id", deviceController.GetDevice)
            devices.GET("/imei/:imei", deviceController.GetDeviceByIMEI)
            devices.POST("", deviceController.CreateDevice)
            devices.PUT("/:id", deviceController.UpdateDevice)
            devices.DELETE("/:id", deviceController.DeleteDevice)
        }
        
        // Vehicle management routes
        vehicles := v1.Group("/vehicles")
        {
            vehicles.GET("", vehicleController.GetVehicles)
            vehicles.GET("/:imei", vehicleController.GetVehicle)
            vehicles.GET("/reg/:reg_no", vehicleController.GetVehicleByRegNo)
            vehicles.GET("/type/:type", vehicleController.GetVehiclesByType)
            vehicles.POST("", vehicleController.CreateVehicle)
            vehicles.PUT("/:imei", vehicleController.UpdateVehicle)
            vehicles.DELETE("/:imei", vehicleController.DeleteVehicle)
        }
        
        // GPS tracking routes
        gps := v1.Group("/gps")
        {
            gps.GET("", gpsController.GetGPSData)
            gps.GET("/latest", gpsController.GetLatestGPSData)
            gps.GET("/:imei", gpsController.GetGPSDataByIMEI)
            gps.GET("/:imei/latest", gpsController.GetLatestGPSDataByIMEI)
            gps.GET("/:imei/route", gpsController.GetGPSRoute)
            gps.DELETE("/:id", gpsController.DeleteGPSData)
        }
        
        // Device control routes
        control := v1.Group("/control")
        {
            control.POST("/cut-oil", controlController.CutOilAndElectricity)
            control.POST("/connect-oil", controlController.ConnectOilAndElectricity)
            control.POST("/get-location", controlController.GetLocation)
            control.GET("/active-devices", controlController.GetActiveDevices)
            control.POST("/quick-cut/:id", controlController.QuickCutOil)
            control.POST("/quick-connect/:id", controlController.QuickConnectOil)
            control.POST("/quick-cut-imei/:imei", controlController.QuickCutOil)
            control.POST("/quick-connect-imei/:imei", controlController.QuickConnectOil)
        }
    }
    
    // Health check endpoint
    router.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "message": "Luna IoT Server is running",
        })
    })
}
```

---

## 📂 internal/http/controllers/ - HTTP Controllers

### 📄 `control_controller.go` - Device Control Controller

**Purpose**: Manages remote device control operations

**Key Features**:
- Active connection management
- Command validation and sending
- Response processing
- Real-time device status

**Core Structure**:
```go
type ControlController struct {
    activeConnections map[string]net.Conn // IMEI -> TCP connection mapping
}

type ControlRequest struct {
    DeviceID *uint  `json:"device_id,omitempty"`
    IMEI     string `json:"imei,omitempty"`
}

type ControlResponse struct {
    Success    bool                      `json:"success"`
    Message    string                    `json:"message"`
    DeviceInfo *models.Device            `json:"device_info,omitempty"`
    Response   *protocol.ControlResponse `json:"control_response,omitempty"`
    Error      string                    `json:"error,omitempty"`
}
```

**Connection Management**:
```go
func (cc *ControlController) RegisterConnection(imei string, conn net.Conn) {
    cc.activeConnections[imei] = conn
    colors.PrintConnection("🔗", "Registered connection for device %s", imei)
}

func (cc *ControlController) UnregisterConnection(imei string) {
    delete(cc.activeConnections, imei)
    colors.PrintConnection("🔌", "Unregistered connection for device %s", imei)
}

func (cc *ControlController) GetActiveConnection(imei string) (net.Conn, bool) {
    conn, exists := cc.activeConnections[imei]
    return conn, exists
}
```

**Control Operations**:
```go
func (cc *ControlController) CutOilAndElectricity(c *gin.Context) {
    // 1. Validate request
    device, errorResponse, err := cc.validateControlRequest(c)
    if err != nil {
        c.JSON(http.StatusBadRequest, errorResponse)
        return
    }
    
    // 2. Check active connection
    conn, exists := cc.GetActiveConnection(device.IMEI)
    if !exists {
        c.JSON(http.StatusServiceUnavailable, ControlResponse{
            Success:    false,
            Error:      "Device not connected",
            Message:    fmt.Sprintf("Device %s is not currently connected", device.IMEI),
            DeviceInfo: device,
        })
        return
    }
    
    // 3. Send control command
    controller := protocol.NewGPSTrackerController(conn, device.IMEI)
    controlResponse, err := controller.CutOilAndElectricity()
    
    if err != nil {
        c.JSON(http.StatusInternalServerError, ControlResponse{
            Success:    false,
            Error:      "Command failed",
            Message:    err.Error(),
            DeviceInfo: device,
        })
        return
    }
    
    // 4. Return success response
    c.JSON(http.StatusOK, ControlResponse{
        Success:    controlResponse.Success,
        Message:    controlResponse.Message,
        DeviceInfo: device,
        Response:   controlResponse,
    })
}
```

### 📄 `device_controller.go` - Device Management Controller

**Purpose**: CRUD operations for GPS tracking devices

**Key Operations**:

```go
func (dc *DeviceController) CreateDevice(c *gin.Context) {
    var device models.Device
    
    // 1. Parse JSON request
    if err := c.ShouldBindJSON(&device); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
        return
    }
    
    // 2. Validate IMEI
    if len(device.IMEI) != 16 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "IMEI must be exactly 16 digits"})
        return
    }
    
    // 3. Check uniqueness
    var existingDevice models.Device
    if err := db.GetDB().Where("imei = ?", device.IMEI).First(&existingDevice).Error; err == nil {
        c.JSON(http.StatusConflict, gin.H{"error": "Device with this IMEI already exists"})
        return
    }
    
    // 4. Save to database
    if err := db.GetDB().Create(&device).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create device"})
        return
    }
    
    c.JSON(http.StatusCreated, gin.H{
        "data":    device,
        "message": "Device created successfully",
    })
}
```

### 📄 `gps_controller.go` - GPS Data Controller

**Purpose**: GPS tracking data access and route generation

**Key Features**:
- Pagination support
- Time-based filtering
- Route generation
- Latest position tracking

```go
func (gc *GPSController) GetGPSRoute(c *gin.Context) {
    imei := c.Param("imei")
    from := c.Query("from")
    to := c.Query("to")
    
    // Parse time range
    fromTime, err := time.Parse("2006-01-02T15:04:05Z", from)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid from time format. Use: 2006-01-02T15:04:05Z",
        })
        return
    }
    
    toTime, err := time.Parse("2006-01-02T15:04:05Z", to)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid to time format. Use: 2006-01-02T15:04:05Z",
        })
        return
    }
    
    // Query GPS data
    var gpsData []models.GPSData
    if err := db.GetDB().Where(
        "imei = ? AND timestamp BETWEEN ? AND ? AND latitude IS NOT NULL AND longitude IS NOT NULL",
        imei, fromTime, toTime,
    ).Order("timestamp ASC").Find(&gpsData).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch GPS route data"})
        return
    }
    
    // Generate route points
    routePoints := make([]gin.H, len(gpsData))
    for i, data := range gpsData {
        routePoints[i] = gin.H{
            "latitude":  data.Latitude,
            "longitude": data.Longitude,
            "timestamp": data.Timestamp,
            "speed":     data.Speed,
            "course":    data.Course,
        }
    }
    
    c.JSON(http.StatusOK, gin.H{
        "imei":         imei,
        "from":         fromTime,
        "to":           toTime,
        "route":        routePoints,
        "total_points": len(routePoints),
        "message":      "GPS route retrieved successfully",
    })
}
```

---

## 📂 internal/tcp/ - TCP Server Implementation

### 📄 `internal/tcp/server.go` - TCP Server

**Purpose**: Handles IoT device connections and real-time data processing

**Server Structure**:
```go
type Server struct {
    port              string
    listener          net.Listener
    controlController *controllers.ControlController
}

func NewServerWithController(port string, sharedController *controllers.ControlController) *Server {
    return &Server{
        port:              port,
        controlController: sharedController,
    }
}
```

**Connection Handling**:
```go
func (s *Server) Start() error {
    listener, err := net.Listen("tcp", ":"+s.port)
    if err != nil {
        return fmt.Errorf("failed to start TCP server: %v", err)
    }
    
    s.listener = listener
    defer listener.Close()
    
    colors.PrintServer("📡", "GT06 TCP Server is running on port %s", s.port)
    
    for {
        conn, err := listener.Accept()
        if err != nil {
            colors.PrintError("Error accepting TCP connection: %v", err)
            continue
        }
        
        // Handle each device connection in separate goroutine
        go s.handleConnection(conn)
    }
}
```

**Device Connection Processing**:
```go
func (s *Server) handleConnection(conn net.Conn) {
    var deviceIMEI string
    
    defer func() {
        conn.Close()
        if deviceIMEI != "" {
            s.controlController.UnregisterConnection(deviceIMEI)
        }
    }()
    
    colors.PrintConnection("📱", "IoT Device connected from %s", conn.RemoteAddr())
    
    // Create GT06 decoder for this connection
    decoder := protocol.NewGT06Decoder()
    buffer := make([]byte, 1024)
    
    for {
        n, err := conn.Read(buffer)
        if err != nil {
            if err.Error() == "EOF" {
                colors.PrintConnection("📱", "IoT Device disconnected: %s", conn.RemoteAddr())
                break
            }
            colors.PrintError("Error reading from connection %s: %v", conn.RemoteAddr(), err)
            break
        }
        
        if n > 0 {
            // Log raw data
            colors.PrintData("📦", "Raw data from %s: %X", conn.RemoteAddr(), buffer[:n])
            
            // Process through GT06 decoder
            packets, err := decoder.AddData(buffer[:n])
            if err != nil {
                colors.PrintError("Error decoding data: %v", err)
                continue
            }
            
            // Process each decoded packet
            for _, packet := range packets {
                switch packet.ProtocolName {
                case "LOGIN":
                    deviceIMEI = s.handleLoginPacket(packet, conn)
                case "GPS_LBS_STATUS", "GPS_LBS_DATA", "GPS_LBS_STATUS_A0":
                    s.handleGPSPacket(packet, conn, deviceIMEI)
                case "STATUS_INFO":
                    s.handleStatusPacket(packet, conn, deviceIMEI)
                case "ALARM_DATA":
                    s.handleAlarmPacket(packet, conn)
                }
                
                // Send response if required
                if packet.NeedsResponse {
                    s.sendResponse(packet, conn, decoder)
                }
            }
        }
    }
}
```

**Packet Type Handlers**:
```go
func (s *Server) handleLoginPacket(packet *protocol.DecodedPacket, conn net.Conn) string {
    if len(packet.TerminalID) >= 16 {
        potentialIMEI := packet.TerminalID[:16]
        
        // Validate device registration
        if !s.isDeviceRegistered(potentialIMEI) {
            colors.PrintError("Unauthorized device: %s", potentialIMEI)
            conn.Close()
            return ""
        }
        
        colors.PrintSuccess("Authorized device login: %s", potentialIMEI)
        s.controlController.RegisterConnection(potentialIMEI, conn)
        return potentialIMEI
    }
    return ""
}

func (s *Server) handleGPSPacket(packet *protocol.DecodedPacket, conn net.Conn, deviceIMEI string) {
    if packet.Latitude != nil && packet.Longitude != nil {
        colors.PrintData("📍", "GPS Location: Lat=%.6f, Lng=%.6f, Speed=%v km/h",
            *packet.Latitude, *packet.Longitude, packet.Speed)
    }
    
    // Save GPS data
    if deviceIMEI != "" && s.isDeviceRegistered(deviceIMEI) {
        gpsData := s.buildGPSData(packet, deviceIMEI)
        if err := db.GetDB().Create(&gpsData).Error; err != nil {
            colors.PrintError("Error saving GPS data: %v", err)
        } else {
            colors.PrintSuccess("GPS data saved for device %s", deviceIMEI)
        }
    }
}
```

---

## 📂 pkg/colors/ - Utility Package

### 📄 `pkg/colors/colors.go` - Console Output Formatting

**Purpose**: Provides colored console output for better readability and debugging

**Key Functions**:
```go
// Color constants
const (
    Reset   = "\033[0m"
    Red     = "\033[31m"
    Green   = "\033[32m"
    Yellow  = "\033[33m"
    Blue    = "\033[34m"
    Purple  = "\033[35m"
    Cyan    = "\033[36m"
    White   = "\033[37m"
    Bold    = "\033[1m"
)

// Formatted output functions
func PrintSuccess(format string, args ...interface{}) {
    fmt.Printf(Green+"✅ "+format+Reset+"\n", args...)
}

func PrintError(format string, args ...interface{}) {
    fmt.Printf(Red+"❌ "+format+Reset+"\n", args...)
}

func PrintWarning(format string, args ...interface{}) {
    fmt.Printf(Yellow+"⚠️  "+format+Reset+"\n", args...)
}

func PrintInfo(format string, args ...interface{}) {
    fmt.Printf(Blue+"ℹ️  "+format+Reset+"\n", args...)
}

func PrintConnection(icon, format string, args ...interface{}) {
    fmt.Printf(Cyan+icon+" "+format+Reset+"\n", args...)
}

func PrintData(icon, format string, args ...interface{}) {
    fmt.Printf(Purple+icon+" "+format+Reset+"\n", args...)
}

func PrintControl(format string, args ...interface{}) {
    fmt.Printf(Yellow+"🎮 "+format+Reset+"\n", args...)
}
```

**Application Banner**:
```go
func PrintBanner() {
    banner := `
    ██╗     ██╗   ██╗███╗   ██╗ █████╗     ██╗ ██████╗ ████████╗
    ██║     ██║   ██║████╗  ██║██╔══██╗    ██║██╔═══██╗╚══██╔══╝
    ██║     ██║   ██║██╔██╗ ██║███████║    ██║██║   ██║   ██║   
    ██║     ██║   ██║██║╚██╗██║██╔══██║    ██║██║   ██║   ██║   
    ███████╗╚██████╔╝██║ ╚████║██║  ██║    ██║╚██████╔╝   ██║   
    ╚══════╝ ╚═════╝ ╚═╝  ╚═══╝╚═╝  ╚═╝    ╚═╝ ╚═════╝    ╚═╝   
                                                                  
    ███████╗███████╗██████╗ ██╗   ██╗███████╗██████╗              
    ██╔════╝██╔════╝██╔══██╗██║   ██║██╔════╝██╔══██╗             
    ███████╗█████╗  ██████╔╝██║   ██║█████╗  ██████╔╝             
    ╚════██║██╔══╝  ██╔══██╗╚██╗ ██╔╝██╔══╝  ██╔══██╗             
    ███████║███████╗██║  ██║ ╚████╔╝ ███████╗██║  ██║             
    ╚══════╝╚══════╝╚═╝  ╚═╝  ╚═══╝  ╚══════╝╚═╝  ╚═╝             
    `
    fmt.Println(Cyan + banner + Reset)
    fmt.Println(Bold + "    🚀 GPS Tracking & Fleet Management System" + Reset)
    fmt.Println("    📡 GT06 Protocol Support | 🌐 REST API | 🎮 Remote Control")
    fmt.Println("")
}
```

---

## 📄 Configuration Files

### `config.example.env` - Environment Configuration Template

**Purpose**: Template for environment variables configuration

```bash
# Luna IoT Server Configuration
# Copy this file to .env and modify the values as needed

# Database Configuration
DB_HOST=localhost                    # Database server address
DB_PORT=5432                        # PostgreSQL port  
DB_USER=postgres                    # Database username
DB_PASSWORD=your_password_here      # Database password
DB_NAME=luna_iot                    # Database name
DB_SSL_MODE=disable                 # SSL connection mode

# Server Ports
HTTP_PORT=8080                      # REST API server port
TCP_PORT=5000                       # IoT device connection port

# Optional: Logging Configuration
LOG_LEVEL=info                      # Logging verbosity (debug, info, warn, error)
LOG_HTTP=false                      # Enable HTTP request logging

# Optional: Performance Settings
MAX_TCP_CONNECTIONS=1000            # Maximum concurrent TCP connections
```

### `setup.sql` - Database Schema Setup

**Purpose**: SQL script for manual database setup and optimization

```sql
-- Luna IoT Database Schema Setup

-- Performance indexes for GPS data queries
CREATE INDEX IF NOT EXISTS idx_gps_data_imei_timestamp ON gps_data(imei, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_gps_data_timestamp ON gps_data(timestamp DESC);

-- Device and vehicle indexes
CREATE INDEX IF NOT EXISTS idx_devices_imei ON devices(imei);
CREATE INDEX IF NOT EXISTS idx_vehicles_imei ON vehicles(imei);
CREATE INDEX IF NOT EXISTS idx_vehicles_reg_no ON vehicles(reg_no);

-- User indexes
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_gps_data_imei_lat_lng ON gps_data(imei, latitude, longitude) 
  WHERE latitude IS NOT NULL AND longitude IS NOT NULL;
```

---

## 🚀 Quick Start Guide

### Installation Steps

1. **Clone Repository**:
   ```bash
   git clone <repository-url>
   cd luna_iot_server
   ```

2. **Install Dependencies**:
   ```bash
   go mod download
   ```

3. **Database Setup**:
   ```sql
   -- PostgreSQL setup
   CREATE DATABASE luna_iot;
   CREATE USER luna_user WITH PASSWORD 'secure_password';
   GRANT ALL PRIVILEGES ON DATABASE luna_iot TO luna_user;
   ```

4. **Environment Configuration**:
   ```bash
   cp config.example.env .env
   # Edit .env with your database credentials
   ```

5. **Start Application**:
   ```bash
   go run main.go
   ```

### API Usage Examples

**Register Device**:
```bash
curl -X POST http://localhost:8080/api/v1/devices \
  -H "Content-Type: application/json" \
  -d '{
    "imei": "123456789012345",
    "sim_no": "9841234567",
    "sim_operator": "Ncell",
    "protocol": "GT06"
  }'
```

**Control Device**:
```bash
curl -X POST http://localhost:8080/api/v1/control/cut-oil \
  -H "Content-Type: application/json" \
  -d '{"imei": "123456789012345"}'
```

**Check Health**:
```bash
curl http://localhost:8080/health
```

---

This comprehensive documentation covers every aspect of the Luna IoT Server project, from the overall architecture down to individual code functions. Each component is explained with its purpose, key features, and implementation details to help developers understand and work with the system effectively. 