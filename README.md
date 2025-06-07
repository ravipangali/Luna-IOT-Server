# Luna IoT Server

A comprehensive GPS tracking server built with Go, supporting GT06 protocol devices with PostgreSQL database and RESTful API.

## Features

- **Unified Server Architecture**: Single command runs both TCP and HTTP servers together
- **GT06 Protocol Support**: Complete implementation of GT06 GPS tracking protocol
- **PostgreSQL Database**: Robust data storage with GORM ORM
- **RESTful API**: Full CRUD operations for users, devices, and vehicles
- **MVC Architecture**: Clean separation of concerns with Models, Views, and Controllers
- **Real-time GPS Tracking**: TCP server for live GPS data reception from IoT devices
- **HTTP API Server**: RESTful endpoints for data management and control
- **Oil & Electricity Control**: Remote vehicle control via GPS devices (cut/connect fuel and electrical systems)
- **Real-time Device Control**: Send commands to connected GPS devices for immediate response
- **Concurrent Processing**: Both servers run simultaneously with shared device connections
- **Graceful Shutdown**: Proper cleanup when stopping the server

## Database Schema

### Tables

1. **devices**
   - `id` (Primary Key)
   - `imei` (Unique, 15 digits)
   - `sim_no` (SIM card number)
   - `sim_operator` (Enum: Ncell, Ntc)
   - `protocol` (Enum: GT06)

2. **users**
   - `id` (Primary Key)
   - `name`
   - `phone` (Unique)
   - `email` (Unique)
   - `password` (Hashed)
   - `role` (Enum: 0=admin, 1=client)
   - `image`

3. **vehicles**
   - `imei` (Primary Key, Foreign Key to devices)
   - `reg_no` (Unique registration number)
   - `name`
   - `odometer`
   - `mileage`
   - `min_fuel`
   - `overspeed`
   - `vehicle_type` (Enum: bike, car, truck, bus, school_bus)

## Environment Variables

Create a `.env` file in the root directory:

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=luna_iot
DB_SSL_MODE=disable

# Server Configuration
HTTP_PORT=8080
TCP_PORT=5000
```

## Installation & Setup

### Prerequisites

- Go 1.24.3 or higher
- PostgreSQL 12 or higher

### Database Setup

1. Create PostgreSQL database:
```sql
CREATE DATABASE luna_iot;
```

2. Install dependencies:
```bash
go mod tidy
```

3. Run migrations (automatic on server start):
```bash
go run cmd/http-server/main.go
```

### Running the Unified Server

The Luna IoT Server now runs both TCP and HTTP servers together in a single command:

#### Using the Run Scripts (Recommended)

**Windows:**
```batch
run_server.bat
```

**Linux/Mac:**
```bash
./run_server.sh
```

#### Manual Build and Run
```bash
# Build
go build -o luna_server main.go

# Run (both TCP and HTTP servers start together)
./luna_server
```

**What happens when you start the server:**
- 📡 TCP Server starts on port 5000 (for IoT device connections)
- 🌐 HTTP Server starts on port 8080 (for API access)
- 💾 Database connection is established
- ⚡ Device control system is enabled
- 🔄 Both servers run simultaneously and share data

## API Endpoints

### Base URL: `http://localhost:8080/api/v1`

### Health Check
- `GET /health` - Server health status

### Users
- `GET /users` - Get all users
- `GET /users/:id` - Get user by ID
- `POST /users` - Create new user
- `PUT /users/:id` - Update user
- `DELETE /users/:id` - Delete user

### Devices
- `GET /devices` - Get all devices
- `GET /devices/:id` - Get device by ID
- `GET /devices/imei/:imei` - Get device by IMEI
- `POST /devices` - Create new device
- `PUT /devices/:id` - Update device
- `DELETE /devices/:id` - Delete device

### Vehicles
- `GET /vehicles` - Get all vehicles
- `GET /vehicles/:imei` - Get vehicle by IMEI
- `GET /vehicles/reg/:reg_no` - Get vehicle by registration number
- `GET /vehicles/type/:type` - Get vehicles by type
- `POST /vehicles` - Create new vehicle
- `PUT /vehicles/:imei` - Update vehicle
- `DELETE /vehicles/:imei` - Delete vehicle

### GPS Tracking
- `GET /gps` - Get GPS data with optional filtering (imei, from, to, page, limit)
- `GET /gps/latest` - Get latest GPS data for all devices
- `GET /gps/:imei` - Get GPS data for specific device
- `GET /gps/:imei/latest` - Get latest GPS data for specific device
- `GET /gps/:imei/route` - Get GPS route between two timestamps
- `DELETE /gps/:id` - Delete GPS data (admin only)

### Oil & Electricity Control
- `POST /control/cut-oil` - Cut oil and electricity for a device
- `POST /control/connect-oil` - Connect oil and electricity for a device
- `POST /control/get-location` - Get current location from device
- `GET /control/active-devices` - Get list of currently connected devices
- `POST /control/quick-cut/:id` - Quick cut oil by device ID
- `POST /control/quick-connect/:id` - Quick connect oil by device ID
- `POST /control/quick-cut-imei/:imei` - Quick cut oil by IMEI
- `POST /control/quick-connect-imei/:imei` - Quick connect oil by IMEI

## API Examples

### Create User
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "phone": "9841234567",
    "email": "john@example.com",
    "password": "password123",
    "role": 1
  }'
```

### Create Device
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

### Create Vehicle
```bash
curl -X POST http://localhost:8080/api/v1/vehicles \
  -H "Content-Type: application/json" \
  -d '{
    "imei": "123456789012345",
    "reg_no": "BA-1-PA-1234",
    "name": "Company Car",
    "mileage": 15.5,
    "min_fuel": 10.0,
    "overspeed": 80,
    "vehicle_type": "car"
  }'
```

### Get Latest GPS Data
```bash
curl http://localhost:8080/api/v1/gps/latest
```

### Get GPS Route
```bash
curl "http://localhost:8080/api/v1/gps/123456789012345/route?from=2024-01-01T00:00:00Z&to=2024-01-01T23:59:59Z"
```

### Cut Oil and Electricity
```bash
curl -X POST http://localhost:8080/api/v1/control/cut-oil \
  -H "Content-Type: application/json" \
  -d '{"imei": "123456789012345"}'
```

### Connect Oil and Electricity
```bash
curl -X POST http://localhost:8080/api/v1/control/connect-oil \
  -H "Content-Type: application/json" \
  -d '{"imei": "123456789012345"}'
```

### Get Active Devices
```bash
curl http://localhost:8080/api/v1/control/active-devices
```

## Project Structure

```
luna_iot_server/
├── cmd/
│   ├── http-server/          # HTTP API server
│   └── tcp-server/           # TCP GPS server
├── config/
│   └── database.go           # Database configuration
├── internal/
│   ├── db/
│   │   └── connection.go     # Database connection
│   ├── http/
│   │   ├── controllers/      # HTTP controllers
│   │   │   ├── control_controller.go  # Oil & electricity control
│   │   │   ├── device_controller.go
│   │   │   ├── gps_controller.go
│   │   │   ├── user_controller.go
│   │   │   └── vehicle_controller.go
│   │   ├── routes.go        # Route definitions
│   │   └── server.go        # HTTP server setup
│   ├── models/              # Database models
│   │   ├── device.go
│   │   ├── gps_data.go
│   │   ├── user.go
│   │   └── vehicle.go
│   ├── protocol/            # GT06 protocol implementation
│   │   ├── gt06_decoder.go  # Protocol decoder
│   │   └── gps_tracker_control.go  # Device control commands
│   ├── tracker/             # GPS tracking logic
│   └── websocket/           # WebSocket support
├── examples/                # Example code and tests
│   └── oil_control_test.go  # Oil & electricity control test
├── pkg/                     # Public packages
├── OIL_ELECTRICITY_CONTROL.md  # Detailed control system documentation
├── test_oil_control.sh      # Bash script for testing control endpoints
├── go.mod
└── go.sum
```

## GPS Device Integration

The TCP server (port 5000) accepts GT06 protocol connections from GPS devices. Supported packet types:

- **LOGIN (0x01)**: Device authentication
- **GPS_LBS_STATUS (0x12)**: GPS location with LBS data
- **STATUS_INFO (0x13)**: Device status information
- **ALARM_DATA (0x16)**: Alarm notifications

## Development

### Adding New Models

1. Create model in `internal/models/`
2. Add to migrations in `internal/db/connection.go`
3. Create controller in `internal/http/controllers/`
4. Add routes in `internal/http/routes.go`

### Database Migrations

Migrations run automatically using GORM AutoMigrate. For custom migrations, modify the `RunMigrations()` function in `internal/db/connection.go`.

## Testing

```bash
# Test database connection
go run cmd/http-server/main.go

# Test TCP server
go run cmd/tcp-server/main.go

# Test API endpoints
curl http://localhost:8080/health

# Test oil & electricity control
./test_oil_control.sh

# Run Go test for control functionality
go run examples/oil_control_test.go
```

## Oil & Electricity Control

The server now supports remote vehicle control through GPS devices. For detailed information about the control system, see [OIL_ELECTRICITY_CONTROL.md](OIL_ELECTRICITY_CONTROL.md).

### Quick Start for Control Features

1. Ensure both servers are running (HTTP and TCP)
2. Register a device in the database
3. Connect the GPS device to the TCP server
4. Use the control API endpoints to send commands

### Safety Features

- **Speed Limitation**: Oil cutting is disabled when vehicle speed > 20 km/h
- **GPS Requirement**: Commands require active GPS tracking
- **Device Validation**: Only registered devices can receive commands
- **Real-time Communication**: Commands are sent immediately over active TCP connections

## License

This project is licensed under the MIT License. 