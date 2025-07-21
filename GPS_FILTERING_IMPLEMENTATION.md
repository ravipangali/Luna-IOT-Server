# GPS Filtering Implementation

## Overview
This document describes the GPS filtering functionality that has been implemented in the Luna IoT TCP server to optimize data storage and reduce unnecessary location tracking.

## Filtering Conditions

The TCP server now applies filtering to GPS data based on the following conditions:

### 1. Ignition Status Filter
- **Condition**: When `ignition == "OFF"`
- **Action**: Ignore (don't save) latitude, longitude, and speed data
- **Rationale**: When the vehicle is turned off, location tracking is not meaningful for route analysis

### 2. Low Speed Filter  
- **Condition**: When `speed < 5` km/h
- **Action**: Ignore (don't save) latitude, longitude, and speed data
- **Rationale**: Very slow speeds indicate the vehicle is essentially stationary, so location updates create noise rather than useful tracking data

## What Gets Filtered vs. Preserved

### Filtered Out (Not Saved):
- ✅ `Latitude` - Set to `nil`
- ✅ `Longitude` - Set to `nil` 
- ✅ `Speed` - Set to `nil`
- ✅ `Course` - Set to `nil`
- ✅ `Altitude` - Set to `nil`

### Always Preserved (Still Saved):
- ✅ `IMEI` - Device identifier
- ✅ `Timestamp` - When the data was received
- ✅ `Ignition` - ON/OFF status
- ✅ `Charger` - CONNECTED/DISCONNECTED
- ✅ `GPSTracking` - ENABLED/DISABLED
- ✅ `OilElectricity` - CONNECTED/DISCONNECTED
- ✅ `DeviceStatus` - ACTIVATED/DEACTIVATED
- ✅ `VoltageLevel` & `VoltageStatus` - Battery information
- ✅ `GSMSignal` & `GSMStatus` - Network signal information
- ✅ `Satellites` - GPS satellite count
- ✅ `AlarmActive`, `AlarmType`, `AlarmCode` - Security alerts
- ✅ `MCC`, `MNC`, `LAC`, `CellID` - Cell tower information

## Implementation Details

### Files Modified
1. **`internal/tcp/server.go`**
   - Modified `handleGPSPacket()` method
   - Modified `handleStatusPacket()` method  
   - Added new `buildFilteredGPSData()` method

### New Method: `buildFilteredGPSData()`
This method creates a GPS data structure without location information, preserving only status and device information.

### WebSocket Behavior
- When location data is filtered, only status updates are broadcast via WebSocket
- No location updates are sent to prevent showing false movement on maps
- Device status information is still broadcast for monitoring connectivity

## Benefits

1. **Reduced Database Storage**: Eliminates unnecessary location records when vehicles are stationary
2. **Improved Performance**: Fewer database writes and WebSocket broadcasts
3. **Better Data Quality**: Prevents noise in tracking data from vehicles that aren't actually moving
4. **Preserved Monitoring**: Device status, connectivity, and security information is still tracked
5. **Route Accuracy**: Only meaningful movement is recorded for route analysis

## Testing

### Run the Test Program
```bash
cd luna_iot_server
go run cmd/test-gps-filtering/main.go
```

This will test various scenarios and show how the filtering logic works.

### Test Scenarios Covered
1. Vehicle stopped, ignition OFF → **FILTERED**
2. Vehicle parked, ignition ON, speed 0 → **FILTERED** 
3. Vehicle slow, ignition ON, speed 3 → **FILTERED**
4. Vehicle starting, ignition ON, speed 5 → **ACCEPTED**
5. Vehicle moving, ignition ON, speed 25 → **ACCEPTED**
6. Vehicle fast, ignition ON, speed 80 → **ACCEPTED**
7. Vehicle moving, ignition OFF, speed 15 → **FILTERED** (ignition takes precedence)

## Monitoring

The server logs will show when filtering is applied:

```
🚫 Filtering location data: Ignition is OFF
🚫 Filtering location data: Speed (3 km/h) is less than 5
📍 Saving status data only (no GPS coordinates) for device 1234567890123456
✅ Filtered GPS data (status only) saved for device 1234567890123456
```

## Configuration

The filtering is enabled by default and cannot be disabled. This ensures consistent behavior across all GPS tracking operations.

## Database Impact

- Status records are still created for every update
- Location fields (`latitude`, `longitude`, `speed`, `course`, `altitude`) will be `NULL` when filtered
- Other tracking and analysis queries remain unaffected
- Historical data before this implementation is not affected 