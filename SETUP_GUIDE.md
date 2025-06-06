# Luna IoT Server Setup Guide

This guide will help you set up the Luna IoT Server with PostgreSQL database on your local machine.

## Prerequisites

1. **Go 1.24.3 or higher**
   - Download from: https://golang.org/dl/

2. **PostgreSQL 12 or higher**
   - Download from: https://www.postgresql.org/download/

## Step 1: Database Setup

1. Install and start PostgreSQL
2. Create a new database:
```sql
CREATE DATABASE luna_iot;
```

3. Create a user (optional):
```sql
CREATE USER luna_user WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE luna_iot TO luna_user;
```

## Step 2: Environment Configuration

Create a `.env` file in the project root directory:

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password_here
DB_NAME=luna_iot
DB_SSL_MODE=disable

# Server Configuration
HTTP_PORT=8080
TCP_PORT=5000
```

**Note:** Replace `your_password_here` with your actual PostgreSQL password.

## Step 3: Install Dependencies

```bash
go mod tidy
```

## Step 4: Start the Servers

### Option 1: Start HTTP API Server
```bash
go run cmd/http-server/main.go
```
This will:
- Connect to PostgreSQL database
- Run automatic migrations to create tables
- Start HTTP server on port 8080 (or HTTP_PORT from .env)

### Option 2: Start TCP GPS Server
```bash
go run cmd/tcp-server/main.go
```
This will:
- Connect to PostgreSQL database
- Start TCP server on port 5000 (or TCP_PORT from .env)
- Accept GT06 device connections and save GPS data

### Option 3: Run Both Servers (Recommended)
Open two terminal windows and run both commands above.

## Step 5: Verify Installation

### Test HTTP API
```bash
# Check server health
curl http://localhost:8080/health

# Expected response:
# {"message":"Luna IoT Server is running","status":"ok"}
```

### Test Database Connection
If the servers start without errors and you see:
```
Database connection established successfully
Database migrations completed successfully
```
Then your database is set up correctly.

## Step 6: Load Sample Data (Optional)

Run the setup script to load sample data:
```bash
psql -d luna_iot -f setup.sql
```

This will create:
- 3 sample users (including admin)
- 3 sample devices
- 3 sample vehicles
- Sample GPS tracking data

## Step 7: Test API Endpoints

### Create a Device
```bash
curl -X POST http://localhost:8080/api/v1/devices \
  -H "Content-Type: application/json" \
  -d '{
    "imei": "123456789012348",
    "sim_no": "9841234573",
    "sim_operator": "Ncell",
    "protocol": "GT06"
  }'
```

### Create a Vehicle
```bash
curl -X POST http://localhost:8080/api/v1/vehicles \
  -H "Content-Type: application/json" \
  -d '{
    "imei": "123456789012348",
    "reg_no": "BA-4-PA-5678",
    "name": "Test Vehicle",
    "vehicle_type": "car"
  }'
```

### Get Latest GPS Data
```bash
curl http://localhost:8080/api/v1/gps/latest
```

## Project Structure Overview

```
luna_iot_server/
├── cmd/
│   ├── http-server/main.go    # HTTP API server entry point
│   └── tcp-server/main.go     # TCP GPS server entry point
├── config/
│   └── database.go            # Database configuration
├── internal/
│   ├── db/
│   │   └── connection.go      # Database connection & migrations
│   ├── http/
│   │   ├── controllers/       # API controllers (MVC pattern)
│   │   ├── routes.go         # Route definitions
│   │   └── server.go         # HTTP server setup
│   ├── models/               # Database models (GORM)
│   │   ├── device.go
│   │   ├── user.go
│   │   ├── vehicle.go
│   │   └── gps_data.go
│   └── protocol/
│       └── gt06_decoder.go   # GT06 protocol implementation
├── .env                      # Environment configuration
├── setup.sql                 # Sample data script
├── README.md                 # Main documentation
└── SETUP_GUIDE.md           # This setup guide
```

## Available API Endpoints

### Users Management
- `GET /api/v1/users` - List all users
- `POST /api/v1/users` - Create user
- `GET /api/v1/users/:id` - Get user by ID
- `PUT /api/v1/users/:id` - Update user
- `DELETE /api/v1/users/:id` - Delete user

### Device Management
- `GET /api/v1/devices` - List all devices
- `POST /api/v1/devices` - Create device
- `GET /api/v1/devices/:id` - Get device by ID
- `GET /api/v1/devices/imei/:imei` - Get device by IMEI
- `PUT /api/v1/devices/:id` - Update device
- `DELETE /api/v1/devices/:id` - Delete device

### Vehicle Management
- `GET /api/v1/vehicles` - List all vehicles
- `POST /api/v1/vehicles` - Create vehicle
- `GET /api/v1/vehicles/:imei` - Get vehicle by IMEI
- `GET /api/v1/vehicles/reg/:reg_no` - Get vehicle by registration
- `GET /api/v1/vehicles/type/:type` - Get vehicles by type
- `PUT /api/v1/vehicles/:imei` - Update vehicle
- `DELETE /api/v1/vehicles/:imei` - Delete vehicle

### GPS Tracking
- `GET /api/v1/gps` - Get GPS data (with filtering)
- `GET /api/v1/gps/latest` - Latest GPS data for all devices
- `GET /api/v1/gps/:imei` - GPS data for specific device
- `GET /api/v1/gps/:imei/latest` - Latest GPS data for device
- `GET /api/v1/gps/:imei/route` - GPS route data
- `DELETE /api/v1/gps/:id` - Delete GPS data

## Troubleshooting

### Database Connection Issues
1. Verify PostgreSQL is running
2. Check database credentials in `.env`
3. Ensure database `luna_iot` exists
4. Check firewall settings

### Port Already in Use
1. Change ports in `.env` file
2. Or kill existing processes:
```bash
# Windows
netstat -ano | findstr :8080
taskkill /PID <PID> /F

# Linux/Mac
lsof -ti:8080 | xargs kill
```

### Missing Dependencies
```bash
go mod tidy
go mod download
```

## Default Credentials (Sample Data)

If you loaded sample data:
- **Admin User**: admin@lunaiot.com / admin123
- **Regular User**: john@example.com / admin123

## Next Steps

1. Connect GT06 GPS devices to TCP port 5000
2. Build a frontend application using the API
3. Set up real-time WebSocket connections
4. Implement user authentication and authorization
5. Add more device protocols as needed

## Support

For issues or questions, check the main README.md file or review the code documentation. 