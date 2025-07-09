package controllers

import (
	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SettingController struct{}

func NewSettingController() *SettingController {
	return &SettingController{}
}

func (sc *SettingController) GetSettings(c *gin.Context) {
	var setting models.Setting
	if err := db.GetDB().First(&setting).Error; err != nil {
		// If no settings exist, one should have been created on startup.
		// If it still fails, there's a problem.
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve settings"})
		return
	}
	c.JSON(http.StatusOK, setting)
}

type UpdateSettingRequest struct {
	MyPayBalance float64 `json:"my_pay_balance"`
}

func (sc *SettingController) UpdateSettings(c *gin.Context) {
	var req UpdateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var setting models.Setting
	if err := db.GetDB().First(&setting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve settings to update"})
		return
	}

	if err := db.GetDB().Model(&setting).Update("my_pay_balance", req.MyPayBalance).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settings"})
		return
	}

	// reload setting to return updated value
	db.GetDB().First(&setting)

	c.JSON(http.StatusOK, setting)
}
