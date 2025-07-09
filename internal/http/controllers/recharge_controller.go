package controllers

import (
	"bytes"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"strings"

	"luna_iot_server/config"
	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RechargeController struct{}

func NewRechargeController() *RechargeController {
	return &RechargeController{}
}

type RechargeRequest struct {
	IMEI   string `json:"imei" binding:"required"`
	Amount int    `json:"amount" binding:"required"`
}

type MyPayRequest struct {
	Token     string `json:"token"`
	Reference int    `json:"reference"`
	Amount    int    `json:"amount"`
	Number    string `json:"number"`
}

type MyPayResponse struct {
	Status  bool   `json:"Status"`
	Message string `json:"Message"`
	Data    struct {
		CreditsConsumed  float64 `json:"CreditsConsumed"`
		CreditsAvailable float64 `json:"CreditsAvailable"`
		ID               int     `json:"Id"`
	} `json:"Data"`
	StatusCode  int         `json:"StatusCode"`
	State       string      `json:"State"`
	Description interface{} `json:"Description"`
}

func (rc *RechargeController) RechargePhone(c *gin.Context) {
	var req RechargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	colors.PrintInfo("Recharge request for IMEI: %s, Amount: %d", req.IMEI, req.Amount)

	// Get vehicle and device details
	var vehicle models.Vehicle
	if err := db.GetDB().Where("imei = ?", req.IMEI).First(&vehicle).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Vehicle not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while fetching vehicle"})
		return
	}

	if err := vehicle.LoadDevice(db.GetDB()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load device details"})
		return
	}

	if vehicle.Device.SimNo == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device or SIM number not found for this vehicle"})
		return
	}

	colors.PrintInfo("Found device for recharge: SIM No: %s, Operator: %s", vehicle.Device.SimNo, vehicle.Device.SimOperator)

	// Prepare request for MyPay API
	myPayConfig := config.GetMyPayConfig()
	simOperator := strings.ToLower(string(vehicle.Device.SimOperator))
	if simOperator != "ntc" && simOperator != "ncell" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported SIM operator", "operator": simOperator})
		return
	}

	myPayReq := MyPayRequest{
		Token:     myPayConfig.Token,
		Reference: rand.Intn(1000000), // Random positive number
		Amount:    req.Amount,
		Number:    vehicle.Device.SimNo,
	}

	postData, _ := json.Marshal(myPayReq)
	url := myPayConfig.URL + simOperator

	colors.PrintDebug("Calling MyPay API: URL=%s, Body=%s", url, string(postData))

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(postData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to call recharge service", "details": err.Error()})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read recharge service response", "details": err.Error()})
		return
	}

	colors.PrintDebug("MyPay API Response: Status=%d, Body=%s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Recharge service returned non-OK status", "status": resp.StatusCode, "response": string(body)})
		return
	}

	var myPayResp MyPayResponse
	if err := json.Unmarshal(body, &myPayResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse recharge service response", "details": err.Error(), "raw_response": string(body)})
		return
	}

	if !myPayResp.Status {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Recharge failed", "response": myPayResp})
		return
	}

	colors.PrintSuccess("Recharge successful for %s. New available credit: %.2f", vehicle.Device.SimNo, myPayResp.Data.CreditsAvailable)

	// Update settings with new balance
	var setting models.Setting
	if err := db.GetDB().First(&setting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve settings to update"})
		return
	}

	if err := db.GetDB().Model(&setting).Update("my_pay_balance", myPayResp.Data.CreditsAvailable).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update balance in settings"})
		return
	}

	colors.PrintSuccess("Successfully updated MyPay balance in settings to %.2f", myPayResp.Data.CreditsAvailable)

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "Recharge successful",
		"response": myPayResp,
	})
}
