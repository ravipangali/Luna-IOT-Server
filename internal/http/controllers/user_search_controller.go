package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"

	"github.com/gin-gonic/gin"
)

type UserSearchController struct{}

func NewUserSearchController() *UserSearchController {
	return &UserSearchController{}
}

// SearchUsersRequest represents the request body for searching users
type SearchUsersRequest struct {
	Query string `json:"query" binding:"required"` // Phone number or name
}

// SearchUsers searches for users by phone number or name
func (usc *UserSearchController) SearchUsers(c *gin.Context) {
	var req SearchUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request body",
			"error":   err.Error(),
		})
		return
	}

	query := strings.TrimSpace(req.Query)
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Query cannot be empty",
		})
		return
	}

	database := db.GetDB()
	var users []models.User

	// Search by phone number (exact match) or name (partial match)
	if err := database.
		Where("phone LIKE ? OR name ILIKE ?", query, "%"+query+"%").
		Limit(20).
		Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to search users",
			"error":   err.Error(),
		})
		return
	}

	// Convert to safe user data (without sensitive information)
	var safeUsers []map[string]interface{}
	for _, user := range users {
		safeUsers = append(safeUsers, user.ToSafeUser())
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Users found successfully",
		"data":    safeUsers,
		"count":   len(safeUsers),
	})
}

// GetAllUsers gets all users for notification selection
func (usc *UserSearchController) GetAllUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset := (page - 1) * limit

	var users []models.User
	var total int64

	database := db.GetDB()

	// Count total
	if err := database.Model(&models.User{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to count users",
			"error":   err.Error(),
		})
		return
	}

	// Get users with pagination
	if err := database.
		Offset(offset).
		Limit(limit).
		Order("name ASC").
		Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to fetch users",
			"error":   err.Error(),
		})
		return
	}

	// Convert to safe user data
	var safeUsers []map[string]interface{}
	for _, user := range users {
		safeUsers = append(safeUsers, user.ToSafeUser())
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Users fetched successfully",
		"data":    safeUsers,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (int(total) + limit - 1) / limit,
		},
	})
}
