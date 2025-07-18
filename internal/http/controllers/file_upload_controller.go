package controllers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type FileUploadController struct{}

func NewFileUploadController() *FileUploadController {
	return &FileUploadController{}
}

type FileUploadResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	FileName string `json:"file_name,omitempty"`
	FilePath string `json:"file_path,omitempty"`
	FileURL  string `json:"file_url,omitempty"`
	Error    string `json:"error,omitempty"`
}

// UploadNotificationImage handles image upload for notifications
func (fuc *FileUploadController) UploadNotificationImage(c *gin.Context) {
	// Log the request details
	fmt.Printf("Upload request received - Method: %s, Content-Type: %s\n",
		c.Request.Method, c.Request.Header.Get("Content-Type"))

	// Get the uploaded file
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		fmt.Printf("Error getting form file: %v\n", err)
		c.JSON(http.StatusBadRequest, FileUploadResponse{
			Success: false,
			Message: "No image file provided. Error: " + err.Error(),
		})
		return
	}
	defer file.Close()

	fmt.Printf("File received - Name: %s, Size: %d, Type: %s\n",
		header.Filename, header.Size, header.Header.Get("Content-Type"))

	// Validate file type
	contentType := header.Header.Get("Content-Type")
	fmt.Printf("Validating content type: %s\n", contentType)
	if !isValidImageType(contentType) {
		fmt.Printf("Invalid content type: %s\n", contentType)
		c.JSON(http.StatusBadRequest, FileUploadResponse{
			Success: false,
			Message: "Invalid file type. Only JPEG, PNG, and GIF images are allowed",
			Error:   "Invalid file type",
		})
		return
	}
	fmt.Printf("Content type validation passed\n")

	// Validate file size (max 5MB)
	fmt.Printf("Validating file size: %d bytes (max: 5242880)\n", header.Size)
	if header.Size > 5242880 { // 5MB in bytes
		fmt.Printf("File size too large: %d bytes\n", header.Size)
		c.JSON(http.StatusBadRequest, FileUploadResponse{
			Success: false,
			Message: "File size too large. Maximum size is 5MB",
			Error:   "File size too large",
		})
		return
	}
	fmt.Printf("File size validation passed\n")

	// Create uploads directory if it doesn't exist
	uploadDir := "uploads/notifications"
	fmt.Printf("Creating upload directory: %s\n", uploadDir)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		fmt.Printf("Error creating upload directory: %v\n", err)
		c.JSON(http.StatusInternalServerError, FileUploadResponse{
			Success: false,
			Message: "Failed to create upload directory. Error: " + err.Error(),
		})
		return
	}
	fmt.Printf("Upload directory created/verified successfully\n")

	// Generate unique filename with timestamp
	timestamp := time.Now().Format("20060102150405")
	uniqueID := uuid.New().String()[:8]
	fileExt := filepath.Ext(header.Filename)
	if fileExt == "" {
		// Determine extension from content type
		switch contentType {
		case "image/jpeg":
			fileExt = ".jpg"
		case "image/png":
			fileExt = ".png"
		case "image/gif":
			fileExt = ".gif"
		default:
			fileExt = ".jpg"
		}
	}

	fileName := fmt.Sprintf("notification_%s_%s%s", timestamp, uniqueID, fileExt)
	filePath := filepath.Join(uploadDir, fileName)

	fmt.Printf("Generated filename: %s\n", fileName)
	fmt.Printf("Full file path: %s\n", filePath)

	// Create the file
	dst, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		c.JSON(http.StatusInternalServerError, FileUploadResponse{
			Success: false,
			Message: "Failed to create file. Error: " + err.Error(),
		})
		return
	}
	defer dst.Close()
	fmt.Printf("File created successfully\n")

	// Copy the uploaded file to the destination file
	if _, err := io.Copy(dst, file); err != nil {
		fmt.Printf("Error copying file: %v\n", err)
		c.JSON(http.StatusInternalServerError, FileUploadResponse{
			Success: false,
			Message: "Failed to save file. Error: " + err.Error(),
		})
		return
	}
	fmt.Printf("File copied successfully\n")

	// Generate file access URL for API access (using public endpoint)
	fileAccessURL := fmt.Sprintf("/api/v1/public/files/notifications/%s", fileName)

	fmt.Printf("Generated file access URL: %s\n", fileAccessURL)
	fmt.Printf("Upload completed successfully\n")

	c.JSON(http.StatusOK, FileUploadResponse{
		Success:  true,
		Message:  "Image uploaded successfully",
		FileName: fileName,
		FilePath: filePath,
		FileURL:  fileAccessURL, // This is the access path to be stored in notification
	})
}

// ServeNotificationImage serves uploaded notification images
func (fuc *FileUploadController) ServeNotificationImage(c *gin.Context) {
	fileName := c.Param("filename")
	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Filename is required"})
		return
	}

	// Validate filename to prevent directory traversal
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid filename"})
		return
	}

	filePath := filepath.Join("uploads/notifications", fileName)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Serve the file
	c.File(filePath)
}

// DeleteNotificationImage deletes an uploaded notification image
func (fuc *FileUploadController) DeleteNotificationImage(c *gin.Context) {
	fileName := c.Param("filename")
	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Filename is required"})
		return
	}

	// Validate filename to prevent directory traversal
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid filename"})
		return
	}

	filePath := filepath.Join("uploads/notifications", fileName)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Delete the file
	if err := os.Remove(filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}

// isValidImageType checks if the content type is a valid image type
func isValidImageType(contentType string) bool {
	validTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
	}

	for _, validType := range validTypes {
		if contentType == validType {
			return true
		}
	}
	return false
}
