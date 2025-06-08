package controllers

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"

	"github.com/gin-gonic/gin"
)

// UserController handles user-related HTTP requests
type UserController struct{}

// NewUserController creates a new user controller
func NewUserController() *UserController {
	return &UserController{}
}

// GetUsers returns all users
func (uc *UserController) GetUsers(c *gin.Context) {
	var users []models.User

	if err := db.GetDB().Find(&users).Error; err != nil {
		colors.PrintError("Failed to fetch users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch users",
		})
		return
	}

	// Clear passwords before returning response
	for i := range users {
		users[i].Password = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    users,
		"count":   len(users),
		"message": "Users retrieved successfully",
	})
}

// GetUser returns a single user by ID
func (uc *UserController) GetUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	var user models.User
	if err := db.GetDB().First(&user, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	// Clear password before returning response
	user.Password = ""

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
		"message": "User retrieved successfully",
	})
}

// CreateUser creates a new user
func (uc *UserController) CreateUser(c *gin.Context) {
	var user models.User

	// Log the raw request body for debugging
	body, _ := c.GetRawData()
	colors.PrintDebug("Raw request body: %s", string(body))

	// Reset the request body for binding
	c.Request.Body = io.NopCloser(strings.NewReader(string(body)))

	if err := c.ShouldBindJSON(&user); err != nil {
		colors.PrintError("Invalid JSON in user creation request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	colors.PrintInfo("Creating user: Name=%s, Email=%s, Role=%d", user.Name, user.Email, user.Role)
	colors.PrintDebug("Password received: %t (length: %d)", user.Password != "", len(user.Password))

	// Validate required fields
	if strings.TrimSpace(user.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Name is required",
		})
		return
	}

	if strings.TrimSpace(user.Email) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email is required",
		})
		return
	}

	if strings.TrimSpace(user.Phone) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Phone is required",
		})
		return
	}

	if strings.TrimSpace(user.Password) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Password is required",
		})
		return
	}

	if len(user.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Password must be at least 6 characters",
		})
		return
	}

	// Validate role
	if user.Role != models.UserRoleAdmin && user.Role != models.UserRoleClient {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       "Invalid role",
			"valid_roles": []int{int(models.UserRoleAdmin), int(models.UserRoleClient)},
		})
		return
	}

	// Check if email already exists
	var existingUser models.User
	if err := db.GetDB().Where("email = ?", user.Email).First(&existingUser).Error; err == nil {
		colors.PrintWarning("User with email %s already exists", user.Email)
		c.JSON(http.StatusConflict, gin.H{
			"error": "User with this email already exists",
		})
		return
	}

	// Check if phone already exists
	if err := db.GetDB().Where("phone = ?", user.Phone).First(&existingUser).Error; err == nil {
		colors.PrintWarning("User with phone %s already exists", user.Phone)
		c.JSON(http.StatusConflict, gin.H{
			"error": "User with this phone number already exists",
		})
		return
	}

	if err := db.GetDB().Create(&user).Error; err != nil {
		colors.PrintError("Failed to create user in database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create user",
			"details": err.Error(),
		})
		return
	}

	colors.PrintSuccess("User created successfully: ID=%d, Email=%s", user.ID, user.Email)
	// Clear password before returning response
	user.Password = ""
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    user,
		"message": "User created successfully",
	})
}

// UpdateUser updates an existing user
func (uc *UserController) UpdateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	var user models.User
	if err := db.GetDB().First(&user, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	var updateData models.User
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
		})
		return
	}

	if err := db.GetDB().Model(&user).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update user",
		})
		return
	}

	// Clear password before returning response
	user.Password = ""

	c.JSON(http.StatusOK, gin.H{
		"data":    user,
		"message": "User updated successfully",
	})
}

// DeleteUser deletes a user
func (uc *UserController) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	var user models.User
	if err := db.GetDB().First(&user, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	if err := db.GetDB().Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User deleted successfully",
	})
}
