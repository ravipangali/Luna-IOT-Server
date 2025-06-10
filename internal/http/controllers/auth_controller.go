package controllers

import (
	"net/http"
	"strings"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuthController handles authentication related HTTP requests
type AuthController struct{}

// NewAuthController creates a new auth controller
func NewAuthController() *AuthController {
	return &AuthController{}
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest represents the registration request body
type RegisterRequest struct {
	Name     string          `json:"name" binding:"required,min=2,max=100"`
	Phone    string          `json:"phone" binding:"required,min=10,max=15"`
	Email    string          `json:"email" binding:"required,email"`
	Password string          `json:"password" binding:"required,min=6"`
	Role     models.UserRole `json:"role" binding:"required,oneof=0 1"`
	Image    string          `json:"image,omitempty"`
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Token   string                 `json:"token,omitempty"`
	User    map[string]interface{} `json:"user,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// Login authenticates a user and returns a token
func (ac *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		colors.PrintError("Invalid login request: %v", err)
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	colors.PrintInfo("Login attempt for email: %s", req.Email)

	// Find user by email
	var user models.User
	if err := db.GetDB().Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			colors.PrintWarning("Login failed: User not found for email %s", req.Email)
			c.JSON(http.StatusUnauthorized, AuthResponse{
				Success: false,
				Error:   "Invalid credentials",
				Message: "Email or password is incorrect",
			})
			return
		}
		colors.PrintError("Database error during login: %v", err)
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Internal server error",
			Message: "Please try again later",
		})
		return
	}

	// Check password
	if !user.CheckPassword(req.Password) {
		colors.PrintWarning("Login failed: Invalid password for email %s", req.Email)
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Invalid credentials",
			Message: "Email or password is incorrect",
		})
		return
	}

	// Generate new token
	if err := user.GenerateToken(); err != nil {
		colors.PrintError("Failed to generate token for user %s: %v", req.Email, err)
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to generate authentication token",
			Message: "Please try again later",
		})
		return
	}

	// Save token to database
	if err := db.GetDB().Save(&user).Error; err != nil {
		colors.PrintError("Failed to save token for user %s: %v", req.Email, err)
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to save authentication token",
			Message: "Please try again later",
		})
		return
	}

	colors.PrintSuccess("User %s logged in successfully", req.Email)
	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Login successful",
		Token:   user.Token,
		User:    user.ToSafeUser(),
	})
}

// Register creates a new user account
func (ac *AuthController) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		colors.PrintError("Invalid registration request: %v", err)
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	colors.PrintInfo("Registration attempt for email: %s", req.Email)

	// Check if email already exists
	var existingUser models.User
	if err := db.GetDB().Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		colors.PrintWarning("Registration failed: Email %s already exists", req.Email)
		c.JSON(http.StatusConflict, AuthResponse{
			Success: false,
			Error:   "Email already exists",
			Message: "A user with this email address already exists",
		})
		return
	}

	// Check if phone already exists
	if err := db.GetDB().Where("phone = ?", req.Phone).First(&existingUser).Error; err == nil {
		colors.PrintWarning("Registration failed: Phone %s already exists", req.Phone)
		c.JSON(http.StatusConflict, AuthResponse{
			Success: false,
			Error:   "Phone number already exists",
			Message: "A user with this phone number already exists",
		})
		return
	}

	// Validate image if provided
	if req.Image != "" {
		if !strings.HasPrefix(req.Image, "data:image/") {
			c.JSON(http.StatusBadRequest, AuthResponse{
				Success: false,
				Error:   "Invalid image format",
				Message: "Image must be a valid base64 data URL",
			})
			return
		}
		if len(req.Image) > 7*1024*1024 {
			c.JSON(http.StatusBadRequest, AuthResponse{
				Success: false,
				Error:   "Image too large",
				Message: "Image size must be less than 5MB",
			})
			return
		}
	}

	// Create new user
	user := models.User{
		Name:     req.Name,
		Phone:    req.Phone,
		Email:    req.Email,
		Password: req.Password, // Will be hashed by BeforeCreate hook
		Role:     req.Role,
		Image:    req.Image,
	}

	// Generate initial token
	if err := user.GenerateToken(); err != nil {
		colors.PrintError("Failed to generate token for new user %s: %v", req.Email, err)
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to generate authentication token",
			Message: "Please try again later",
		})
		return
	}

	// Save user to database
	if err := db.GetDB().Create(&user).Error; err != nil {
		colors.PrintError("Failed to create user %s: %v", req.Email, err)
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to create user account",
			Message: "Please try again later",
		})
		return
	}

	colors.PrintSuccess("User %s registered successfully", req.Email)
	c.JSON(http.StatusCreated, AuthResponse{
		Success: true,
		Message: "Registration successful",
		Token:   user.Token,
		User:    user.ToSafeUser(),
	})
}

// Logout invalidates the user's token
func (ac *AuthController) Logout(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	user := userInterface.(*models.User)

	// Clear token
	user.ClearToken()
	if err := db.GetDB().Save(user).Error; err != nil {
		colors.PrintError("Failed to clear token for user %s: %v", user.Email, err)
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to logout",
			Message: "Please try again later",
		})
		return
	}

	colors.PrintInfo("User %s logged out successfully", user.Email)
	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Logout successful",
	})
}

// Me returns the current authenticated user's information
func (ac *AuthController) Me(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	user := userInterface.(*models.User)
	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "User information retrieved successfully",
		User:    user.ToSafeUser(),
	})
}

// RefreshToken generates a new token for the authenticated user
func (ac *AuthController) RefreshToken(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	user := userInterface.(*models.User)

	// Generate new token
	if err := user.GenerateToken(); err != nil {
		colors.PrintError("Failed to refresh token for user %s: %v", user.Email, err)
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to refresh token",
			Message: "Please try again later",
		})
		return
	}

	// Save new token
	if err := db.GetDB().Save(user).Error; err != nil {
		colors.PrintError("Failed to save refreshed token for user %s: %v", user.Email, err)
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to save refreshed token",
			Message: "Please try again later",
		})
		return
	}

	colors.PrintInfo("Token refreshed for user %s", user.Email)
	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Token refreshed successfully",
		Token:   user.Token,
		User:    user.ToSafeUser(),
	})
}
