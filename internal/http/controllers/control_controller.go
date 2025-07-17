package controllers

import (
	"fmt"
	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/internal/protocol"
	"luna_iot_server/pkg/colors"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ControlController handles oil and electricity control operations
type ControlController struct {
	activeConnections map[string]net.Conn // Maps IMEI to active TCP connections
}

// NewControlController creates a new control controller instance
func NewControlController() *ControlController {
	return &ControlController{
		activeConnections: make(map[string]net.Conn),
	}
}

// RegisterConnection registers an active TCP connection for a device
func (cc *ControlController) RegisterConnection(imei string, conn net.Conn) {
	cc.activeConnections[imei] = conn
	colors.PrintConnection("🔗", "Registered connection for device %s", imei)

}

// UnregisterConnection removes a TCP connection for a device
func (cc *ControlController) UnregisterConnection(imei string) {
	delete(cc.activeConnections, imei)
	colors.PrintConnection("🔌", "Unregistered connection for device %s", imei)
}

// GetActiveConnection retrieves the active TCP connection for a device
func (cc *ControlController) GetActiveConnection(imei string) (net.Conn, bool) {
	colors.PrintDebug("Looking for active connection for IMEI: %s", imei)
	colors.PrintDebug("Currently registered IMEIs: %v", cc.getRegisteredIMEIs())
	conn, exists := cc.activeConnections[imei]
	if exists {
		colors.PrintDebug("Found active connection for IMEI: %s", imei)
	} else {
		colors.PrintWarning("No active connection found for IMEI: %s", imei)
	}
	return conn, exists
}

// getRegisteredIMEIs returns a list of currently registered IMEIs for debugging
func (cc *ControlController) getRegisteredIMEIs() []string {
	var imeis []string
	for imei := range cc.activeConnections {
		imeis = append(imeis, imei)
	}
	return imeis
}

// ControlRequest represents the request body for control operations
type ControlRequest struct {
	DeviceID *uint  `json:"device_id,omitempty"`
	IMEI     string `json:"imei,omitempty"`
}

// ControlResponse represents the response for control operations
type ControlResponse struct {
	Success    bool                      `json:"success"`
	Message    string                    `json:"message"`
	DeviceInfo *models.Device            `json:"device_info,omitempty"`
	Response   *protocol.ControlResponse `json:"control_response,omitempty"`
	Error      string                    `json:"error,omitempty"`
}

// validateControlRequest validates and processes the control request
func (cc *ControlController) validateControlRequest(c *gin.Context) (*models.Device, *ControlResponse, error) {
	var req ControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, &ControlResponse{
			Success: false,
			Error:   "Invalid request format",
			Message: err.Error(),
		}, err
	}

	var device models.Device
	var err error

	// Find device by IMEI or ID
	if req.IMEI != "" {
		err = db.GetDB().Where("imei = ?", req.IMEI).First(&device).Error
	} else if req.DeviceID != nil {
		err = db.GetDB().Where("id = ?", *req.DeviceID).First(&device).Error
	} else {
		return nil, &ControlResponse{
			Success: false,
			Error:   "Either device_id or imei must be provided",
			Message: "Missing device identifier",
		}, fmt.Errorf("missing device identifier")
	}

	if err != nil {
		return nil, &ControlResponse{
			Success: false,
			Error:   "Device not found",
			Message: fmt.Sprintf("Device not found in database: %v", err),
		}, err
	}

	return &device, nil, nil
}

// CutOilAndElectricity handles cutting oil and electricity for a device
// @Summary Cut oil and electricity
// @Description Send command to cut oil and electricity for a GPS tracking device
// @Tags control
// @Accept json
// @Produce json
// @Param request body ControlRequest true "Control request"
// @Success 200 {object} ControlResponse
// @Failure 400 {object} ControlResponse
// @Failure 404 {object} ControlResponse
// @Failure 500 {object} ControlResponse
// @Router /control/cut-oil [post]
func (cc *ControlController) CutOilAndElectricity(c *gin.Context) {
	device, errorResponse, err := cc.validateControlRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse)
		return
	}

	// Check if device has an active connection
	conn, exists := cc.GetActiveConnection(device.IMEI)
	if !exists {
		c.JSON(http.StatusServiceUnavailable, ControlResponse{
			Success:    false,
			Error:      "Device not connected",
			Message:    fmt.Sprintf("Device %s is not currently connected to the server", device.IMEI),
			DeviceInfo: device,
		})
		return
	}

	// Create GPS tracker controller
	controller := protocol.NewGPSTrackerController(conn, device.IMEI)

	// Send cut oil command
	controlResponse, err := controller.CutOilAndElectricity()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ControlResponse{
			Success:    false,
			Error:      "Command failed",
			Message:    fmt.Sprintf("Failed to cut oil and electricity: %v", err),
			DeviceInfo: device,
		})
		return
	}

	// Save control action to database (optional - you can create a control_logs table)
	colors.PrintControl("Oil cut command sent to device %s - Success: %v, Message: %s",
		device.IMEI, controlResponse.Success, controlResponse.Message)

	c.JSON(http.StatusOK, ControlResponse{
		Success:    controlResponse.Success,
		Message:    controlResponse.Message,
		DeviceInfo: device,
		Response:   controlResponse,
	})
}

// ConnectOilAndElectricity handles connecting oil and electricity for a device
// @Summary Connect oil and electricity
// @Description Send command to connect oil and electricity for a GPS tracking device
// @Tags control
// @Accept json
// @Produce json
// @Param request body ControlRequest true "Control request"
// @Success 200 {object} ControlResponse
// @Failure 400 {object} ControlResponse
// @Failure 404 {object} ControlResponse
// @Failure 500 {object} ControlResponse
// @Router /control/connect-oil [post]
func (cc *ControlController) ConnectOilAndElectricity(c *gin.Context) {
	device, errorResponse, err := cc.validateControlRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse)
		return
	}

	// Check if device has an active connection
	conn, exists := cc.GetActiveConnection(device.IMEI)
	if !exists {
		c.JSON(http.StatusServiceUnavailable, ControlResponse{
			Success:    false,
			Error:      "Device not connected",
			Message:    fmt.Sprintf("Device %s is not currently connected to the server", device.IMEI),
			DeviceInfo: device,
		})
		return
	}

	// Create GPS tracker controller
	controller := protocol.NewGPSTrackerController(conn, device.IMEI)

	// Send connect oil command
	controlResponse, err := controller.ConnectOilAndElectricity()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ControlResponse{
			Success:    false,
			Error:      "Command failed",
			Message:    fmt.Sprintf("Failed to connect oil and electricity: %v", err),
			DeviceInfo: device,
		})
		return
	}

	// Save control action to database (optional)
	colors.PrintControl("Oil connect command sent to device %s - Success: %v, Message: %s",
		device.IMEI, controlResponse.Success, controlResponse.Message)

	c.JSON(http.StatusOK, ControlResponse{
		Success:    controlResponse.Success,
		Message:    controlResponse.Message,
		DeviceInfo: device,
		Response:   controlResponse,
	})
}

// GetLocation handles getting current location for a device
// @Summary Get device location
// @Description Send command to get current location from a GPS tracking device
// @Tags control
// @Accept json
// @Produce json
// @Param request body ControlRequest true "Control request"
// @Success 200 {object} ControlResponse
// @Failure 400 {object} ControlResponse
// @Failure 404 {object} ControlResponse
// @Failure 500 {object} ControlResponse
// @Router /control/get-location [post]
func (cc *ControlController) GetLocation(c *gin.Context) {
	device, errorResponse, err := cc.validateControlRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse)
		return
	}

	// Check if device has an active connection
	conn, exists := cc.GetActiveConnection(device.IMEI)
	if !exists {
		c.JSON(http.StatusServiceUnavailable, ControlResponse{
			Success:    false,
			Error:      "Device not connected",
			Message:    fmt.Sprintf("Device %s is not currently connected to the server", device.IMEI),
			DeviceInfo: device,
		})
		return
	}

	// Create GPS tracker controller
	controller := protocol.NewGPSTrackerController(conn, device.IMEI)

	// Send get location command
	controlResponse, err := controller.GetLocation()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ControlResponse{
			Success:    false,
			Error:      "Command failed",
			Message:    fmt.Sprintf("Failed to get location: %v", err),
			DeviceInfo: device,
		})
		return
	}

	// Save control action to database (optional)
	colors.PrintControl("Location request sent to device %s - Success: %v, Response: %s",
		device.IMEI, controlResponse.Success, controlResponse.Response)

	c.JSON(http.StatusOK, ControlResponse{
		Success:    controlResponse.Success,
		Message:    controlResponse.Message,
		DeviceInfo: device,
		Response:   controlResponse,
	})
}

// GetActiveDevices returns a list of currently connected devices
// @Summary Get active devices
// @Description Get list of devices currently connected to the TCP server
// @Tags control
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /control/active-devices [get]
func (cc *ControlController) GetActiveDevices(c *gin.Context) {
	activeDevices := make([]map[string]interface{}, 0)

	for imei := range cc.activeConnections {
		var device models.Device
		err := db.GetDB().Where("imei = ?", imei).First(&device).Error
		if err == nil {
			activeDevices = append(activeDevices, map[string]interface{}{
				"imei":         device.IMEI,
				"id":           device.ID,
				"sim_no":       device.SimNo,
				"sim_operator": device.SimOperator,
				"protocol":     device.Protocol,
				"connected_at": time.Now(),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"message":        "Active devices retrieved successfully",
		"active_devices": activeDevices,
		"total_count":    len(activeDevices),
	})
}

// QuickCutOil handles cutting oil for a device by ID or IMEI via URL params
// @Summary Quick cut oil (URL params)
// @Description Quick endpoint to cut oil and electricity using URL parameters
// @Tags control
// @Produce json
// @Param id query int false "Device ID"
// @Param imei query string false "Device IMEI"
// @Success 200 {object} ControlResponse
// @Failure 400 {object} ControlResponse
// @Router /control/quick-cut/{id} [post]
// @Router /control/quick-cut-imei/{imei} [post]
func (cc *ControlController) QuickCutOil(c *gin.Context) {
	// Try to get device by ID from URL param
	idParam := c.Param("id")
	imeiParam := c.Param("imei")

	var device models.Device
	var err error

	if idParam != "" {
		id, parseErr := strconv.ParseUint(idParam, 10, 32)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, ControlResponse{
				Success: false,
				Error:   "Invalid device ID",
				Message: "Device ID must be a valid number",
			})
			return
		}
		err = db.GetDB().Where("id = ?", uint(id)).First(&device).Error
	} else if imeiParam != "" {
		err = db.GetDB().Where("imei = ?", imeiParam).First(&device).Error
	} else {
		c.JSON(http.StatusBadRequest, ControlResponse{
			Success: false,
			Error:   "Missing parameter",
			Message: "Either device ID or IMEI must be provided in URL",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusNotFound, ControlResponse{
			Success: false,
			Error:   "Device not found",
			Message: fmt.Sprintf("Device not found: %v", err),
		})
		return
	}

	// Check connection and send command
	conn, exists := cc.GetActiveConnection(device.IMEI)
	if !exists {
		c.JSON(http.StatusServiceUnavailable, ControlResponse{
			Success:    false,
			Error:      "Device not connected",
			Message:    fmt.Sprintf("Device %s is not currently connected", device.IMEI),
			DeviceInfo: &device,
		})
		return
	}

	controller := protocol.NewGPSTrackerController(conn, device.IMEI)
	controlResponse, err := controller.CutOilAndElectricity()

	if err != nil {
		c.JSON(http.StatusInternalServerError, ControlResponse{
			Success:    false,
			Error:      "Command failed",
			Message:    err.Error(),
			DeviceInfo: &device,
		})
		return
	}

	c.JSON(http.StatusOK, ControlResponse{
		Success:    controlResponse.Success,
		Message:    controlResponse.Message,
		DeviceInfo: &device,
		Response:   controlResponse,
	})
}

// QuickConnectOil handles connecting oil for a device by ID or IMEI via URL params
// @Summary Quick connect oil (URL params)
// @Description Quick endpoint to connect oil and electricity using URL parameters
// @Tags control
// @Produce json
// @Param id query int false "Device ID"
// @Param imei query string false "Device IMEI"
// @Success 200 {object} ControlResponse
// @Failure 400 {object} ControlResponse
// @Router /control/quick-connect/{id} [post]
// @Router /control/quick-connect-imei/{imei} [post]
func (cc *ControlController) QuickConnectOil(c *gin.Context) {
	// Try to get device by ID from URL param
	idParam := c.Param("id")
	imeiParam := c.Param("imei")

	var device models.Device
	var err error

	if idParam != "" {
		id, parseErr := strconv.ParseUint(idParam, 10, 32)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, ControlResponse{
				Success: false,
				Error:   "Invalid device ID",
				Message: "Device ID must be a valid number",
			})
			return
		}
		err = db.GetDB().Where("id = ?", uint(id)).First(&device).Error
	} else if imeiParam != "" {
		err = db.GetDB().Where("imei = ?", imeiParam).First(&device).Error
	} else {
		c.JSON(http.StatusBadRequest, ControlResponse{
			Success: false,
			Error:   "Missing parameter",
			Message: "Either device ID or IMEI must be provided in URL",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusNotFound, ControlResponse{
			Success: false,
			Error:   "Device not found",
			Message: fmt.Sprintf("Device not found: %v", err),
		})
		return
	}

	// Check connection and send command
	conn, exists := cc.GetActiveConnection(device.IMEI)
	if !exists {
		c.JSON(http.StatusServiceUnavailable, ControlResponse{
			Success:    false,
			Error:      "Device not connected",
			Message:    fmt.Sprintf("Device %s is not currently connected", device.IMEI),
			DeviceInfo: &device,
		})
		return
	}

	controller := protocol.NewGPSTrackerController(conn, device.IMEI)
	controlResponse, err := controller.ConnectOilAndElectricity()

	if err != nil {
		c.JSON(http.StatusInternalServerError, ControlResponse{
			Success:    false,
			Error:      "Command failed",
			Message:    err.Error(),
			DeviceInfo: &device,
		})
		return
	}

	c.JSON(http.StatusOK, ControlResponse{
		Success:    controlResponse.Success,
		Message:    controlResponse.Message,
		DeviceInfo: &device,
		Response:   controlResponse,
	})
}
