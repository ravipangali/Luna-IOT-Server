# Device IMEI Validation for GPS Data

## Overview

The Luna IoT Server now includes **device authentication and validation** for GPS tracking. Only devices that are registered in the database can connect and send GPS data to the server.

## How It Works

### 1. Device Registration Required

Before a GPS device can send data, it must be registered in the `devices` table through the HTTP API:

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

### 2. Login Validation

When a GPS device attempts to connect to the TCP server:

1. **Device sends LOGIN packet** with its Terminal ID (IMEI)
2. **Server extracts IMEI** from the Terminal ID
3. **Server checks database** to verify if device is registered
4. **If device is registered**: Connection is allowed ✅
5. **If device is NOT registered**: Connection is rejected and closed ❌

### 3. Data Processing Validation

For every GPS data packet received:

1. **Before saving GPS data**, the server re-validates that the device still exists in the database
2. **If device was deleted** after login: GPS data is ignored
3. **If device still exists**: GPS data is saved normally

## Security Features

### ✅ **What's Protected:**
- Only registered devices can connect
- Unregistered devices are immediately disconnected
- GPS data from deleted devices is ignored
- Real-time device validation

### 🔍 **Logging & Monitoring:**
- Unauthorized login attempts are logged
- Device registration status is displayed
- Clear visual indicators (✅ authorized, ⚠️ rejected)

### 🛡️ **Connection Security:**
```
✅ Authorized device login: 123456789012345
⚠️  Rejecting unregistered device: 999999999999999
```

## Implementation Details

### Database Function
```go
// Helper function to check device registration
func isDeviceRegistered(imei string) bool {
    var device models.Device
    err := db.GetDB().Where("imei = ?", imei).First(&device).Error
    return err == nil
}
```

### Login Validation
```go
case "LOGIN":
    potentialIMEI := packet.TerminalID[:15]
    
    // Check if device exists in database
    if !isDeviceRegistered(potentialIMEI) {
        log.Printf("Unauthorized device attempted login from %s - IMEI: %s (not registered)", 
            conn.RemoteAddr(), potentialIMEI)
        conn.Close()  // Close connection for unregistered devices
        return
    }
    
    // Device is registered, allow connection
    deviceIMEI = potentialIMEI
    fmt.Printf("✅ Authorized device login: %s\n", deviceIMEI)
```

### Data Validation
```go
// Before saving GPS data
if deviceIMEI != "" {
    // Verify device still exists before saving GPS data
    if !isDeviceRegistered(deviceIMEI) {
        log.Printf("Device %s no longer registered, ignoring GPS data", deviceIMEI)
        continue
    }
    
    // Save GPS data...
}
```

## Testing the Validation

### 1. Test with Registered Device
```bash
# Register a device first
curl -X POST http://localhost:8080/api/v1/devices \
  -H "Content-Type: application/json" \
  -d '{"imei": "123456789012345", "sim_no": "9841234567", "sim_operator": "Ncell", "protocol": "GT06"}'

# Device with IMEI 123456789012345 can now connect and send GPS data
```

### 2. Test with Unregistered Device
```bash
# Device with unregistered IMEI will be rejected:
# ⚠️  Rejecting unregistered device: 999999999999999
# Connection will be closed immediately
```

## Benefits

1. **Security**: Prevents unauthorized devices from sending fake GPS data
2. **Data Integrity**: Ensures only legitimate devices contribute to the tracking system
3. **Resource Protection**: Prevents spam and malicious data from consuming server resources
4. **Audit Trail**: Complete logging of authorized and unauthorized connection attempts
5. **Real-time Validation**: Continuous verification even after initial login

## Database Requirements

The validation depends on the `devices` table having proper IMEI records:

```sql
-- Devices table structure
CREATE TABLE devices (
    id SERIAL PRIMARY KEY,
    imei VARCHAR(15) UNIQUE NOT NULL,
    sim_no VARCHAR(20),
    sim_operator VARCHAR(10),
    protocol VARCHAR(10),
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

## Migration Notes

- **Existing devices**: Must be registered in the database before they can connect
- **No breaking changes**: HTTP API remains unchanged
- **Backward compatibility**: Server gracefully handles unregistered devices by logging and rejecting them 