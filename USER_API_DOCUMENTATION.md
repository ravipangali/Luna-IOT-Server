# Luna IoT User-Based API Documentation

## Overview

This documentation covers the **user-based APIs** designed specifically for client applications. All endpoints are secured with user authentication and permission-based access control, ensuring users can only access their authorized vehicles.

## Authentication

All user-based APIs require authentication using Bearer tokens obtained from the login endpoint.

### Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "phone": "1234567890",
  "password": "your_password"
}
```

**Response:**
```json
{
  "success": true,
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": 1,
    "name": "John Doe",
    "email": "john@example.com",
    "phone": "1234567890",
    "role": 1
  }
}
```

### Using Authentication Token
Include the token in the Authorization header for all authenticated requests:
```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

## User-Based Vehicle APIs

### Get My Vehicles
Get all vehicles accessible to the authenticated user with their permissions.

```http
GET /api/v1/my-vehicles
Authorization: Bearer <token>
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "imei": "1234567890123456",
      "reg_no": "KA01AB1234",
      "name": "My Car",
      "vehicle_type": "car",
      "user_role": "main",
      "permissions": ["all_access"],
      "is_main_user": true,
      "device": {
        "model": "GT06",
        "status": "active"
      }
    }
  ]
}
```

### Get Specific Vehicle
```http
GET /api/v1/my-vehicles/{imei}
Authorization: Bearer <token>
```

## User-Based Tracking APIs

### Get All Vehicles Tracking
Get real-time tracking data for all user's accessible vehicles.

```http
GET /api/v1/my-tracking
Authorization: Bearer <token>
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "imei": "1234567890123456",
      "reg_no": "KA01AB1234",
      "name": "My Car",
      "vehicle_type": "car",
      "status": "moving",
      "has_status_data": true,
      "has_location_data": true,
      "latest_status": {
        "speed": 45,
        "ignition": "ON",
        "timestamp": "2024-01-15T10:30:00Z"
      },
      "latest_location": {
        "latitude": 12.9716,
        "longitude": 77.5946,
        "timestamp": "2024-01-15T10:30:00Z"
      }
    }
  ]
}
```

### Get Specific Vehicle Tracking
```http
GET /api/v1/my-tracking/{imei}
Authorization: Bearer <token>
```

### Get Vehicle Location Only
```http
GET /api/v1/my-tracking/{imei}/location
Authorization: Bearer <token>
```

### Get Vehicle Status Only
```http
GET /api/v1/my-tracking/{imei}/status
Authorization: Bearer <token>
```

### Get Vehicle History
```http
GET /api/v1/my-tracking/{imei}/history?page=1&limit=50&date=2024-01-15
Authorization: Bearer <token>
```

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `limit` (optional): Records per page (default: 50, max: 100)
- `date` (optional): Specific date (YYYY-MM-DD)
- `start_time` (optional): Start time (ISO 8601)
- `end_time` (optional): End time (ISO 8601)

### Get Vehicle Route
```http
GET /api/v1/my-tracking/{imei}/route?date=2024-01-15
Authorization: Bearer <token>
```

### Get Vehicle Reports
```http
GET /api/v1/my-tracking/{imei}/reports?period=today
Authorization: Bearer <token>
```

**Query Parameters:**
- `period`: today | yesterday | this_week | this_month | custom
- `start_date` (for custom): Start date (YYYY-MM-DD)
- `end_date` (for custom): End date (YYYY-MM-DD)

## User-Based Control APIs

### Cut Oil & Electricity
```http
POST /api/v1/my-control/{imei}/cut-oil
Authorization: Bearer <token>
```

**Response:**
```json
{
  "success": true,
  "message": "Oil and electricity cut command sent successfully",
  "vehicle_info": {
    "imei": "1234567890123456",
    "reg_no": "KA01AB1234",
    "name": "My Car"
  },
  "control_response": {
    "command": "cut_oil",
    "status": "sent",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

### Connect Oil & Electricity
```http
POST /api/v1/my-control/{imei}/connect-oil
Authorization: Bearer <token>
```

### Request Vehicle Location
```http
POST /api/v1/my-control/{imei}/get-location
Authorization: Bearer <token>
```

### Get Active Devices
```http
GET /api/v1/my-control/active-devices
Authorization: Bearer <token>
```

## User-Based GPS APIs

### Get All Vehicles GPS Data
```http
GET /api/v1/my-gps
Authorization: Bearer <token>
```

### Get Vehicle GPS Location
```http
GET /api/v1/my-gps/{imei}/location
Authorization: Bearer <token>
```

### Get Vehicle GPS Status
```http
GET /api/v1/my-gps/{imei}/status
Authorization: Bearer <token>
```

### Get Vehicle GPS History
```http
GET /api/v1/my-gps/{imei}/history?page=1&limit=50
Authorization: Bearer <token>
```

### Get Vehicle GPS Route
```http
GET /api/v1/my-gps/{imei}/route?date=2024-01-15
Authorization: Bearer <token>
```

### Get Vehicle GPS Report
```http
GET /api/v1/my-gps/{imei}/report?period=today
Authorization: Bearer <token>
```

## WebSocket Real-Time Updates

### Connection
Connect to WebSocket for real-time updates using authentication token:

```javascript
const ws = new WebSocket('ws://localhost:8080/ws?token=eyJhbGciOiJIUzI1NiIs...');
```

### Message Types

#### Welcome Message
Received immediately after successful connection:
```json
{
  "type": "welcome",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "user_id": 1,
    "accessible_imeis": ["1234567890123456"],
    "message": "WebSocket connection established"
  }
}
```

#### Location Update
Real-time GPS location updates:
```json
{
  "type": "location_update",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "imei": "1234567890123456",
    "vehicle_name": "My Car",
    "reg_no": "KA01AB1234",
    "latitude": 12.9716,
    "longitude": 77.5946,
    "speed": 45,
    "course": 180,
    "location_valid": true
  }
}
```

#### Status Update
Real-time device status updates:
```json
{
  "type": "status_update",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "imei": "1234567890123456",
    "vehicle_name": "My Car",
    "speed": 45,
    "ignition": "ON",
    "is_moving": true,
    "connection_status": "connected",
    "battery": {
      "level": 85,
      "status": "Normal",
      "charging": true
    },
    "signal": {
      "bars": 4,
      "status": "Good",
      "percentage": 80
    }
  }
}
```

## Permission System

Users have different permission levels for each vehicle:

- **all_access**: Full access to all features
- **live_tracking**: Real-time location and status
- **history**: Historical GPS data access
- **report**: Analytics and reports
- **vehicle_edit**: Modify vehicle settings
- **notification**: Receive alerts
- **share_tracking**: Share vehicle with others

## Error Handling

All APIs return consistent error responses:

```json
{
  "success": false,
  "error": "Vehicle not found or access denied",
  "code": "VEHICLE_ACCESS_DENIED"
}
```

### Common Error Codes
- `AUTHENTICATION_REQUIRED`: Missing or invalid token
- `VEHICLE_ACCESS_DENIED`: No permission for vehicle
- `PERMISSION_INSUFFICIENT`: Specific permission required
- `VEHICLE_NOT_FOUND`: Vehicle doesn't exist
- `ACCESS_EXPIRED`: Vehicle access has expired
- `DEVICE_OFFLINE`: Device not connected

## Rate Limiting

API endpoints have the following rate limits:
- Authentication: 10 requests/minute
- Tracking APIs: 60 requests/minute
- Control APIs: 20 requests/minute
- GPS APIs: 100 requests/minute

## SDKs and Libraries

### JavaScript/TypeScript
```javascript
// Example usage with fetch
const response = await fetch('/api/v1/my-tracking', {
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  }
});
const data = await response.json();
```

### Flutter/Dart
```dart
// Example service method
Future<List<Vehicle>> getMyVehicles() async {
  final response = await http.get(
    Uri.parse('$baseUrl/api/v1/my-vehicles'),
    headers: {
      'Authorization': 'Bearer $token',
      'Content-Type': 'application/json',
    },
  );
  // Handle response...
}
```

## Best Practices

1. **Cache Management**: Cache vehicle data but refresh tracking data frequently
2. **WebSocket Reconnection**: Implement automatic reconnection with exponential backoff
3. **Permission Checks**: Check user permissions before showing UI elements
4. **Error Handling**: Provide user-friendly error messages
5. **Offline Support**: Cache essential data for offline viewing
6. **Token Refresh**: Implement automatic token refresh before expiration

## Support

For API support and questions:
- Email: support@lunaiot.com
- Documentation: https://docs.lunaiot.com
- Issue Tracker: https://github.com/lunaiot/issues 