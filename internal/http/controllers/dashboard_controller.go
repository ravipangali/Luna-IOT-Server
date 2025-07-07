package controllers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/utils"

	"github.com/gin-gonic/gin"
)

type DashboardController struct{}

func NewDashboardController() *DashboardController {
	return &DashboardController{}
}

type DashboardStatsResponse struct {
	TotalUsers        int64   `json:"total_users"`
	TotalVehicles     int64   `json:"total_vehicles"`
	TotalHitsToday    int64   `json:"total_hits_today"`
	TotalKMToday      float64 `json:"total_km_today"`
	TotalSMSAvailable int     `json:"total_sms_available"`
}

type smsBalance struct {
	Balance int `json:"BALANCE"`
}

func (dc *DashboardController) GetDashboardStats(c *gin.Context) {
	var totalUsers, totalVehicles, totalHitsToday int64
	var totalKMToday float64
	var totalSMSAvailable int

	gormDB := db.GetDB()

	if err := gormDB.Model(&models.User{}).Count(&totalUsers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get total users"})
		return
	}

	if err := gormDB.Model(&models.Vehicle{}).Count(&totalVehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get total vehicles"})
		return
	}

	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	if err := gormDB.Model(&models.GPSData{}).Where("timestamp >= ? AND timestamp < ?", startOfDay, endOfDay).Count(&totalHitsToday).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get total hits today"})
		return
	}

	var gpsDataToday []models.GPSData
	if err := gormDB.Model(&models.GPSData{}).Where("timestamp >= ? AND timestamp < ?", startOfDay, endOfDay).Order("imei, timestamp").Find(&gpsDataToday).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get GPS data for today"})
		return
	}

	if len(gpsDataToday) > 1 {
		for i := 1; i < len(gpsDataToday); i++ {
			prev, curr := gpsDataToday[i-1], gpsDataToday[i]
			if prev.IMEI == curr.IMEI && prev.IsValidLocation() && curr.IsValidLocation() {
				totalKMToday += utils.CalculateDistance(*prev.Latitude, *prev.Longitude, *curr.Latitude, *curr.Longitude)
			}
		}
	}

	resp, err := http.Get("https://sms.kaichogroup.com/miscapi/568383D0C5AA82/getBalance/true/")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err == nil {
				var smsBalanceResponse []smsBalance
				if json.Unmarshal(bodyBytes, &smsBalanceResponse) == nil && len(smsBalanceResponse) > 0 {
					totalSMSAvailable = smsBalanceResponse[0].Balance
				}
			}
		}
	}

	stats := DashboardStatsResponse{
		TotalUsers:        totalUsers,
		TotalVehicles:     totalVehicles,
		TotalHitsToday:    totalHitsToday,
		TotalKMToday:      totalKMToday,
		TotalSMSAvailable: totalSMSAvailable,
	}

	c.JSON(http.StatusOK, stats)
}
