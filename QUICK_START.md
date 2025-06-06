# Quick Start Guide - Fix Database Error

## The Error You're Seeing

```
dial tcp [::1]:5432: connectex: No connection could be made because the target machine actively refused it.
```

This means PostgreSQL is not running on your system.

## Step-by-Step Fix

### 1. Check PostgreSQL Installation

Open Command Prompt as Administrator and run:
```cmd
psql --version
```

If PostgreSQL is not installed, download it from: https://www.postgresql.org/download/windows/

### 2. Start PostgreSQL Service

#### Option A: Use the batch script (easiest)
```cmd
start_postgres.bat
```

#### Option B: Manual service start
```cmd
net start postgresql-x64-14
```
*(Replace '14' with your PostgreSQL version: 13, 15, 16, etc.)*

#### Option C: Check what PostgreSQL services are available
```cmd
sc query type=service state=all | findstr postgresql
```

### 3. Create Database

Once PostgreSQL is running:
```cmd
psql -U postgres
```

Then in PostgreSQL prompt:
```sql
CREATE DATABASE luna_iot;
\q
```

### 4. Create Environment File

Create `.env` file with your settings:
```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=Luna@#$321
DB_NAME=luna_iot
DB_SSL_MODE=disable

HTTP_PORT=9000
TCP_PORT=5000
```

### 5. Test the Setup

Run the test to verify everything compiles:
```cmd
go run test_compile.go
```

### 6. Start the Server

```cmd
go run cmd/http-server/main.go
```

You should see:
```
Database connection established successfully
✓ Users table ready
✓ Devices table ready  
✓ Vehicles table ready
✓ GPS data table ready
Database migrations completed successfully
Luna IoT HTTP Server starting on port 9000
```

### 7. Test API

Open another terminal:
```cmd
curl http://localhost:9000/health
```

Expected response:
```json
{"message":"Luna IoT Server is running","status":"ok"}
```

## Troubleshooting

### PostgreSQL Won't Start
1. Check if another instance is running
2. Check Windows Services (services.msc)
3. Look for PostgreSQL in the services list
4. Right-click → Start

### Wrong PostgreSQL Version
If you get "service not found", find your version:
```cmd
dir "C:\Program Files\PostgreSQL\"
```

### Password Issues
If you get authentication errors:
1. Reset PostgreSQL password
2. Update `.env` file with correct password

### Port Already in Use
Change the HTTP_PORT in `.env` to a different port (like 9001, 9002, etc.)

## Success Indicators

✅ PostgreSQL service running
✅ Database `luna_iot` created  
✅ Server starts without errors
✅ Health endpoint responds
✅ Tables created automatically

## Next Steps

Once everything is working:
1. Load sample data: `psql -d luna_iot -f setup.sql`
2. Test API endpoints (see README.md)
3. Start TCP server: `go run cmd/tcp-server/main.go` 