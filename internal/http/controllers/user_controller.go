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

	// Check if vehicle relationships should be loaded
	includeVehicles := c.Query("include_vehicles") == "true"

	query := db.GetDB()
	if includeVehicles {
		query = query.Preload("VehicleAccess").Preload("VehicleAccess.Vehicle").Preload("Vehicles")
	}

	if err := query.Find(&users).Error; err != nil {
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

	// Check if vehicle relationships should be loaded
	includeVehicles := c.Query("include_vehicles") == "true"

	var user models.User
	query := db.GetDB()
	if includeVehicles {
		query = query.Preload("VehicleAccess").Preload("VehicleAccess.Vehicle").Preload("Vehicles")
	}

	if err := query.First(&user, uint(id)).Error; err != nil {
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

	// Image validation
	if user.Image != "" {
		// Check if image is a valid base64 string
		if !strings.HasPrefix(user.Image, "data:image/") {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid image format",
			})
			return
		}

		// Check image size (roughly estimate base64 size)
		// Base64 encoding increases size by ~33%, so 5MB file would be ~6.7MB in base64
		// We'll use 7MB as a safe limit
		if len(user.Image) > 7*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Image size too large, max 5MB allowed",
			})
			return
		}

		colors.PrintInfo("User image included in request (size: %d bytes)", len(user.Image))
	}

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

	// Read raw body for debugging
	body, _ := c.GetRawData()
	colors.PrintDebug("📋 Update user raw request body: %s", string(body))

	// Reset body for binding
	c.Request.Body = io.NopCloser(strings.NewReader(string(body)))

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
		})
		return
	}

	colors.PrintInfo("Updating user: ID=%d, Data: %v", user.ID, updateData)

	// Image validation
	if image, ok := updateData["image"].(string); ok && image != "" && image != user.Image {
		if !strings.HasPrefix(image, "data:image/") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image format"})
			return
		}
		if len(image) > 7*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Image size too large, max 5MB allowed"})
			return
		}
		colors.PrintInfo("User image updated (size: %d bytes)", len(image))
	}

	// Handle password update
	if password, ok := updateData["password"].(string); ok {
		if strings.TrimSpace(password) == "" {
			delete(updateData, "password") // Remove empty password from update data
			colors.PrintInfo("Password field is empty, ignoring password update")
		} else if len(password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 6 characters"})
			return
		} else {
			colors.PrintInfo("Password update detected for user ID=%d", user.ID)
			// The BeforeUpdate hook will handle hashing
		}
	}

	// Validate email and phone uniqueness if they changed
	if email, ok := updateData["email"].(string); ok && email != user.Email {
		var existingUser models.User
		if err := db.GetDB().Where("email = ? AND id != ?", email, user.ID).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already in use by another user"})
			return
		}
	}

	if phone, ok := updateData["phone"].(string); ok && phone != user.Phone {
		var existingUser models.User
		if err := db.GetDB().Where("phone = ? AND id != ?", phone, user.ID).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Phone number already in use by another user"})
			return
		}
	}

	if err := db.GetDB().Model(&user).Updates(updateData).Error; err != nil {
		colors.PrintError("Failed to update user in database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update user",
		})
		return
	}

	// Important: We need to reload the user from the database to get the updated fields
	// because `Updates` with a map doesn't update the original `user` struct in-place.
	db.GetDB().First(&user, uint(id))

	// Clear password before returning response
	user.Password = ""
	colors.PrintSuccess("User updated successfully: ID=%d", user.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
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

// GetUserImage returns a user's profile image
func (uc *UserController) GetUserImage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	var user models.User
	if err := db.GetDB().Select("id, image").First(&user, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	if user.Image == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User has no profile image",
		})
		return
	}

	// Return the base64 image data
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"image":   user.Image,
	})
}

// DeleteUserImage removes a user's profile image
func (uc *UserController) DeleteUserImage(c *gin.Context) {
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

	// Check if user has an image
	if user.Image == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User has no profile image to delete",
		})
		return
	}

	// Update user to remove image
	if err := db.GetDB().Model(&user).Update("image", "").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete user image",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User image deleted successfully",
	})
}

// ForceDeleteUsersBackupData permanently deletes all soft-deleted users
func (uc *UserController) ForceDeleteUsersBackupData(c *gin.Context) {
	gormDB := db.GetDB()

	// Count records to be deleted for confirmation
	var deletedUsers int64
	gormDB.Unscoped().Model(&models.User{}).Where("deleted_at IS NOT NULL").Count(&deletedUsers)

	if deletedUsers == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success":       true,
			"message":       "No deleted user backup data found to force delete",
			"deleted_count": 0,
		})
		return
	}

	// Perform the permanent deletion
	result := gormDB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.User{})
	if result.Error != nil {
		colors.PrintError("Failed to force delete users: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to force delete user backup data"})
		return
	}

	colors.PrintSuccess("Force deleted %d users permanently", deletedUsers)

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "User backup data has been permanently removed",
		"deleted_count": deletedUsers,
	})
}
