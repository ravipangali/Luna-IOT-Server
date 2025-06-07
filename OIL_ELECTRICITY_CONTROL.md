# Oil and Electricity Control System

This document describes the oil and electricity control functionality implemented in the Luna IoT Server, based on the GT06 GPS tracking protocol.

## Overview

The oil and electricity control system allows remote control of vehicle fuel and electrical systems through GPS tracking devices. This is commonly used for:

- **Vehicle Security**: Remotely disable vehicles in case of theft
- **Fleet Management**: Control vehicle access and operation
- **Emergency Response**: Quickly disable vehicles in emergency situations
- **Parental Control**: Control teen driver vehicle access

## Features

- **Cut Oil and Electricity**: Remotely disable vehicle fuel and electrical systems
- **Connect Oil and Electricity**: Remotely enable vehicle fuel and electrical systems
- **Location Requests**: Get current vehicle location on demand
- **Active Device Monitoring**: Track which devices are currently connected
- **Real-time Communication**: Direct TCP communication with GPS devices

## Architecture

### Components

1. **GPS Tracker Control Protocol** (`internal/protocol/gps_tracker_control.go`)
   - Implements GT06 protocol for control commands
   - Handles packet construction and response parsing
   - Manages command serialization and checksums

2. **Control Controller** (`internal/http/controllers/control_controller.go`)
   - HTTP API endpoints for control operations
   - Connection management for active devices
   - Request validation and response handling

3. **TCP Server Integration** (`cmd/tcp-server/main.go`)
   - Tracks active device connections
   - Registers/unregisters devices for control operations
   - Maintains connection state for real-time commands

## Protocol Details

### Command Structure

The system uses the GT06 protocol with the following packet structure:

```
Start Bit (2 bytes) | Packet Length (1 byte) | Protocol Number (1 byte) | 
Command Length (1 byte) | Server Flag (4 bytes) | Command Content (variable) |
Language (2 bytes) | Serial Number (2 bytes) | Error Check (2 bytes) | Stop Bit (2 bytes)
```

### Control Commands

| Command | Code | Description |
|---------|------|-------------|
| Cut Oil | `DYD#` | Disable fuel and electrical systems |
| Connect Oil | `HFYD#` | Enable fuel and electrical systems |
| Get Location | `DWXX#` | Request current GPS location |

### Response Codes

#### Cut Oil Responses
- `DYD=Success!` - Oil and electricity successfully cut
- `DYD=Speed Limit, Speed XXkm/h` - Cannot cut due to high speed (>20km/h)
- `DYD=Unvalued Fix` - Cannot cut due to GPS tracking being off

#### Connect Oil Responses
- `HFYD=Success!` - Oil and electricity successfully connected
- `HFYD=Fail!` - Failed to connect oil and electricity

## API Endpoints

### Base URL: `http://localhost:8080/api/v1/control`

### 1. Cut Oil and Electricity
```http
POST /control/cut-oil
Content-Type: application/json

{
  "imei": "123456789012345"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Oil and electricity successfully cut",
  "device_info": {
    "id": 1,
    "imei": "123456789012345",
    "sim_no": "9841234567",
    "sim_operator": "Ncell",
    "protocol": "GT06"
  },
  "control_response": {
    "command": "DYD#",
    "response": "DYD=Success!",
    "success": true,
    "message": "Oil and electricity successfully cut",
    "timestamp": "2024-01-15T10:30:00Z",
    "device_imei": "123456789012345"
  }
}
```

### 2. Connect Oil and Electricity
```http
POST /control/connect-oil
Content-Type: application/json

{
  "imei": "123456789012345"
}
```

### 3. Get Location
```http
POST /control/get-location
Content-Type: application/json

{
  "imei": "123456789012345"
}
```

### 4. Get Active Devices
```http
GET /control/active-devices
```

**Response:**
```json
{
  "success": true,
  "message": "Active devices retrieved successfully",
  "active_devices": [
    {
      "imei": "123456789012345",
      "id": 1,
      "sim_no": "9841234567",
      "sim_operator": "Ncell",
      "protocol": "GT06",
      "connected_at": "2024-01-15T10:00:00Z"
    }
  ],
  "total_count": 1
}
```

### 5. Quick Control Endpoints

#### Quick Cut Oil (by Device ID)
```http
POST /control/quick-cut/{device_id}
```

#### Quick Cut Oil (by IMEI)
```http
POST /control/quick-cut-imei/{imei}
```

#### Quick Connect Oil (by Device ID)
```http
POST /control/quick-connect/{device_id}
```

#### Quick Connect Oil (by IMEI)
```http
POST /control/quick-connect-imei/{imei}
```

## Usage Examples

### Using cURL

#### Cut Oil and Electricity
```bash
curl -X POST http://localhost:8080/api/v1/control/cut-oil \
  -H "Content-Type: application/json" \
  -d '{"imei": "123456789012345"}'
```

#### Connect Oil and Electricity
```bash
curl -X POST http://localhost:8080/api/v1/control/connect-oil \
  -H "Content-Type: application/json" \
  -d '{"imei": "123456789012345"}'
```

#### Get Active Devices
```bash
curl http://localhost:8080/api/v1/control/active-devices
```

#### Quick Cut Oil by Device ID
```bash
curl -X POST http://localhost:8080/api/v1/control/quick-cut/1
```

### Using Go Code

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type ControlRequest struct {
    IMEI string `json:"imei"`
}

func cutOilAndElectricity(imei string) error {
    request := ControlRequest{IMEI: imei}
    jsonData, _ := json.Marshal(request)
    
    resp, err := http.Post(
        "http://localhost:8080/api/v1/control/cut-oil",
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // Handle response...
    return nil
}
```

## Safety Considerations

### Speed Limitations
- Oil cutting is automatically disabled when vehicle speed exceeds 20 km/h
- This prevents dangerous situations where sudden power loss could cause accidents

### GPS Requirements
- Oil cutting requires active GPS tracking
- Ensures the vehicle location is known before disabling systems

### Connection Requirements
- Device must be actively connected to the TCP server
- Commands are sent in real-time over the existing TCP connection

## Error Handling

### Common Error Scenarios

1. **Device Not Connected**
   ```json
   {
     "success": false,
     "error": "Device not connected",
     "message": "Device 123456789012345 is not currently connected to the server"
   }
   ```

2. **Device Not Found**
   ```json
   {
     "success": false,
     "error": "Device not found",
     "message": "Device not found in database"
   }
   ```

3. **Speed Limit Exceeded**
   ```json
   {
     "success": false,
     "message": "Cannot cut oil - vehicle speed too high (>20km/h)",
     "control_response": {
       "command": "DYD#",
       "response": "DYD=Speed Limit, Speed 45km/h",
       "success": false
     }
   }
   ```

4. **GPS Tracking Off**
   ```json
   {
     "success": false,
     "message": "Cannot cut oil - GPS tracking is off",
     "control_response": {
       "command": "DYD#",
       "response": "DYD=Unvalued Fix",
       "success": false
     }
   }
   ```

## Testing

### Test File
Run the provided test file to verify functionality:

```bash
go run examples/oil_control_test.go
```

### Prerequisites for Testing
1. Have the TCP server running (`go run cmd/tcp-server/main.go`)
2. Have the HTTP server running (`go run cmd/http-server/main.go`)
3. Have at least one device registered in the database
4. Have a GPS device connected to the TCP server

### Test Scenarios
The test file covers:
- Getting active devices
- Cutting oil and electricity
- Connecting oil and electricity
- Getting location
- Quick control operations

## Integration with Existing System

### Database Requirements
- Uses existing `devices` table for device validation
- No additional database tables required
- Leverages existing GORM models

### TCP Server Integration
- Extends existing GT06 protocol handling
- Maintains backward compatibility
- Adds connection tracking for control operations

### HTTP Server Integration
- Adds new `/control` endpoint group
- Uses existing middleware and error handling
- Follows existing API patterns

## Security Considerations

### Authentication
- Currently uses device IMEI validation against database
- Devices must be pre-registered to receive commands
- Unauthorized devices are automatically rejected

### Authorization
- Commands are only sent to registered devices
- Connection state is verified before sending commands
- Real-time validation of device status

### Audit Trail
- All control commands are logged
- Response status is tracked
- Connection events are recorded

## Future Enhancements

### Planned Features
1. **Command History**: Store control command history in database
2. **User Authentication**: Add user-based access control
3. **Geofencing**: Location-based automatic control
4. **Scheduling**: Time-based control operations
5. **Notifications**: Real-time alerts for control events
6. **Batch Operations**: Control multiple devices simultaneously

### Protocol Extensions
1. **Additional Commands**: Support for more GT06 control commands
2. **Custom Commands**: Support for device-specific commands
3. **Status Monitoring**: Enhanced device status reporting
4. **Firmware Updates**: Remote firmware update capabilities

## Troubleshooting

### Common Issues

1. **Device Not Responding**
   - Check TCP connection status
   - Verify device is powered on
   - Check network connectivity

2. **Commands Timing Out**
   - Increase timeout values
   - Check device signal strength
   - Verify protocol compatibility

3. **Invalid Responses**
   - Check device firmware version
   - Verify command format
   - Review protocol documentation

### Debug Mode
Enable debug logging by setting environment variables:
```bash
export DEBUG=true
export LOG_LEVEL=debug
```

### Monitoring
- Monitor TCP server logs for connection events
- Check HTTP server logs for API requests
- Review device response patterns

## Support

For technical support or questions about the oil and electricity control system:

1. Check the server logs for error messages
2. Verify device connectivity and registration
3. Test with the provided example code
4. Review the GT06 protocol documentation
5. Contact the development team with specific error details

---

**Note**: This system is designed for legitimate fleet management and security purposes. Always ensure proper authorization before implementing remote vehicle control systems and comply with local laws and regulations regarding vehicle control systems. 