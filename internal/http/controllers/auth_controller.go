package controllers

import (
	"fmt"
	"log"
	"luna_iot_server/config"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// OTPData stores the generated OTP and its expiration time
type OTPData struct {
	OTP       string
	ExpiresAt time.Time
}

// In-memory store for OTPs, mapping phone numbers to OTP data
var (
	otpStore = make(map[string]OTPData)
	otpMutex = &sync.Mutex{}
)

// AuthController handles authentication related HTTP requests
type AuthController struct{}

// NewAuthController creates a new auth controller
func NewAuthController() *AuthController {
	return &AuthController{}
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Phone    string `json:"phone" binding:"required,min=10,max=15"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest represents the registration request body
type RegisterRequest struct {
	Name     string          `json:"name" binding:"required,min=2,max=100"`
	Phone    string          `json:"phone" binding:"required,min=10,max=15"`
	Email    string          `json:"email" binding:"required,email"`
	Password string          `json:"password" binding:"required,min=6"`
	Role     models.UserRole `json:"role,omitempty"` // Optional, defaults to client (1)
	Image    string          `json:"image,omitempty"`
	OTP      string          `json:"otp,omitempty" binding:"required,len=6"`
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

	colors.PrintInfo("Login attempt for phone: %s", req.Phone)

	// Find user by phone number
	var user models.User
	if err := db.GetDB().Where("phone = ?", req.Phone).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			colors.PrintWarning("Login failed: User not found for phone %s", req.Phone)
			c.JSON(http.StatusUnauthorized, AuthResponse{
				Success: false,
				Error:   "Invalid credentials",
				Message: "Phone number or password is incorrect",
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

	// Check if user is active
	if !user.IsActive {
		colors.PrintWarning("Login failed: User account is not active for phone %s", req.Phone)
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Account not active",
			Message: "Your account is not active. Please contact an administrator.",
		})
		return
	}

	// Check password
	if !user.CheckPassword(req.Password) {
		colors.PrintWarning("Login failed: Invalid password for phone %s", req.Phone)
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Invalid credentials",
			Message: "Phone number or password is incorrect",
		})
		return
	}

	// Generate new token
	if err := user.GenerateToken(); err != nil {
		colors.PrintError("Failed to generate token for user %s: %v", req.Phone, err)
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to generate authentication token",
			Message: "Please try again later",
		})
		return
	}

	// Save token to database
	if err := db.GetDB().Save(&user).Error; err != nil {
		colors.PrintError("Failed to save token for user %s: %v", req.Phone, err)
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to save authentication token",
			Message: "Please try again later",
		})
		return
	}

	colors.PrintSuccess("User %s logged in successfully", req.Phone)
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

	// Verify OTP
	otpMutex.Lock()
	otpData, ok := otpStore[req.Phone]
	otpMutex.Unlock()

	if !ok || otpData.OTP != req.OTP || time.Now().After(otpData.ExpiresAt) {
		colors.PrintWarning("Registration failed: Invalid or expired OTP for phone %s", req.Phone)
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Invalid or expired OTP",
			Message: "The OTP you entered is incorrect or has expired. Please try again.",
		})
		return
	}

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

	// Default role to client (1) if not provided or is 0
	role := req.Role
	if role == models.UserRole(0) {
		// Frontend now always sends role=1, but if somehow 0 is sent, default to client
		role = models.UserRoleClient
	}

	// Create new user
	user := models.User{
		Name:     req.Name,
		Phone:    req.Phone,
		Email:    req.Email,
		Password: req.Password, // Will be hashed by BeforeCreate hook
		Role:     role,
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

	// Clean up OTP from store after successful registration
	otpMutex.Lock()
	delete(otpStore, req.Phone)
	otpMutex.Unlock()
}

// SendOTPRequest represents the request body for sending an OTP
type SendOTPRequest struct {
	Phone string `json:"phone" binding:"required,min=10,max=15"`
}

// SendOTP generates and sends an OTP to the user's phone
func (ac *AuthController) SendOTP(c *gin.Context) {
	var req SendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	// Check if phone number is already registered
	var existingUser models.User
	if err := db.GetDB().Where("phone = ?", req.Phone).First(&existingUser).Error; err == nil {
		colors.PrintWarning("OTP request for already registered phone: %s", req.Phone)
		c.JSON(http.StatusConflict, AuthResponse{
			Success: false,
			Error:   "Phone number already registered",
			Message: "A user with this phone number already exists. Please login.",
		})
		return
	}

	// Generate 6-digit OTP
	otp := fmt.Sprintf("%06d", rand.Intn(1000000))
	expiresAt := time.Now().Add(5 * time.Minute) // OTP valid for 5 minutes

	// Store OTP
	otpMutex.Lock()
	otpStore[req.Phone] = OTPData{OTP: otp, ExpiresAt: expiresAt}
	otpMutex.Unlock()

	colors.PrintInfo("Generated OTP %s for phone %s. Expires at %s", otp, req.Phone, expiresAt.Format(time.RFC3339))

	// Send SMS
	if err := sendSMS(req.Phone, fmt.Sprintf("Your Luna IOT verification code is: %s. It is valid for 5 minutes.", otp)); err != nil {
		colors.PrintError("Failed to send SMS to %s: %v", req.Phone, err)
		// Don't fail the request to the user, but log the error.
		// In a production environment, you might want to handle this differently.
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "OTP sent successfully to your phone number.",
	})
}

// sendSMS is a helper function to call the SMS provider API
func sendSMS(contact, message string) error {
	smsCfg := config.GetSMSConfig()
	if smsCfg.APIKey == "" {
		log.Println("SMS_API_KEY is not set. Skipping SMS.")
		return nil // Or return an error if SMS is critical
	}

	// URL encode the message
	encodedMsg := url.QueryEscape(message)

	// Construct the URL
	apiURL := fmt.Sprintf("%s?key=%s&campaign=%s&routeid=%s&type=text&contacts=%s&senderid=%s&msg=%s",
		smsCfg.APIURL,
		smsCfg.APIKey,
		smsCfg.CampaignID,
		smsCfg.RouteID,
		contact,
		smsCfg.SenderID,
		encodedMsg,
	)

	colors.PrintDebug("Sending SMS via URL: %s", apiURL)

	// Make the GET request
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to make HTTP request to SMS API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SMS API returned non-200 status code: %d", resp.StatusCode)
	}

	colors.PrintSuccess("Successfully sent SMS to %s", contact)
	return nil
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

// DeleteAccount handles account deactivation (soft delete)
// @Summary delete user account
// @Description deletes the currently authenticated user's account by setting is_active to false. Requires password confirmation.
// @Tags auth
// @Accept json
// @Produce json
// @Param password query string true "Password for confirmation"
// @Security BearerAuth
// @Success 200 {object} AuthResponse "Account deleted successfully"
// @Failure 400 {object} AuthResponse "Invalid request"
// @Failure 401 {object} AuthResponse "Unauthorized or invalid password"
// @Failure 500 {object} AuthResponse "Internal server error"
// @Router /auth/delete-account [get]
func (ac *AuthController) DeleteAccount(c *gin.Context) {
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}
	user := currentUser.(*models.User)

	password := c.Query("password")
	if password == "" {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Password is required",
			Message: "Password must be provided as a query parameter.",
		})
		return
	}

	// Verify password
	if !user.CheckPassword(password) {
		colors.PrintWarning("Account deleted failed: Invalid password for user %s", user.Email)
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Invalid password",
			Message: "The provided password is incorrect.",
		})
		return
	}

	// delete account
	if err := db.GetDB().Model(&user).Update("is_active", false).Error; err != nil {
		colors.PrintError("Failed to delete account for user %s: %v", user.Email, err)
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to delete account",
			Message: "An internal error occurred. Please try again later.",
		})
		return
	}

	colors.PrintSuccess("Account deleted for user %s (ID: %d)", user.Email, user.ID)

	c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		Message: "Your account has been successfully deleted.",
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
