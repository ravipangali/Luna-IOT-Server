package middleware

import (
	"net/http"
	"strings"

	"luna_iot_server/internal/db"
	"luna_iot_server/internal/models"
	"luna_iot_server/pkg/colors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuthMiddleware validates the authentication token
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			colors.PrintWarning("Authentication failed: No Authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Unauthorized",
				"message": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Extract token from Bearer token format
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			colors.PrintWarning("Authentication failed: Invalid Authorization header format")
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Unauthorized",
				"message": "Invalid authorization header format. Use: Bearer <token>",
			})
			c.Abort()
			return
		}

		token := tokenParts[1]
		if token == "" {
			colors.PrintWarning("Authentication failed: Empty token")
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Unauthorized",
				"message": "Token is required",
			})
			c.Abort()
			return
		}

		// Find user by token
		var user models.User
		if err := db.GetDB().Where("token = ?", token).First(&user).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				colors.PrintWarning("Authentication failed: Invalid token")
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error":   "Unauthorized",
					"message": "Invalid or expired token",
				})
			} else {
				colors.PrintError("Database error during authentication: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   "Internal server error",
					"message": "Authentication service unavailable",
				})
			}
			c.Abort()
			return
		}

		// Check if token is valid (not expired)
		if !user.IsTokenValid() {
			colors.PrintWarning("Authentication failed: Token expired for user %s", user.Email)
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Unauthorized",
				"message": "Token has expired",
			})
			c.Abort()
			return
		}

		// Set user in context for use in handlers
		c.Set("user", &user)
		c.Set("user_id", user.ID)
		c.Set("user_role", user.Role)

		colors.PrintDebug("Authentication successful for user %s (ID: %d)", user.Email, user.ID)
		c.Next()
	}
}

// OptionalAuthMiddleware validates token if present but doesn't require it
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No auth header, continue without authentication
			c.Next()
			return
		}

		// Extract token from Bearer token format
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			// Invalid format, continue without authentication
			c.Next()
			return
		}

		token := tokenParts[1]
		if token == "" {
			// Empty token, continue without authentication
			c.Next()
			return
		}

		// Find user by token
		var user models.User
		if err := db.GetDB().Where("token = ?", token).First(&user).Error; err != nil {
			// Invalid token, continue without authentication
			c.Next()
			return
		}

		// Check if token is valid (not expired)
		if !user.IsTokenValid() {
			// Expired token, continue without authentication
			c.Next()
			return
		}

		// Set user in context for use in handlers
		c.Set("user", &user)
		c.Set("user_id", user.ID)
		c.Set("user_role", user.Role)

		colors.PrintDebug("Optional authentication successful for user %s (ID: %d)", user.Email, user.ID)
		c.Next()
	}
}

// AdminOnlyMiddleware ensures the authenticated user is an admin
func AdminOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userInterface, exists := c.Get("user")
		if !exists {
			colors.PrintWarning("Admin access denied: No authenticated user")
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Unauthorized",
				"message": "Authentication required",
			})
			c.Abort()
			return
		}

		user := userInterface.(*models.User)
		if user.Role != models.UserRoleAdmin {
			colors.PrintWarning("Admin access denied: User %s is not an admin (role: %d)", user.Email, user.Role)
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Forbidden",
				"message": "Admin access required",
			})
			c.Abort()
			return
		}

		colors.PrintDebug("Admin access granted for user %s", user.Email)
		c.Next()
	}
}
