package handlers

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ethiopia-dating-app/internal/config"
	"ethiopia-dating-app/internal/models"
	"ethiopia-dating-app/internal/redis"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserHandler struct {
	db    *gorm.DB
	redis *redis.Client
	cfg   *config.Config
}

type UpdateProfileRequest struct {
	FirstName string   `json:"first_name,omitempty"`
	LastName  string   `json:"last_name,omitempty"`
	Bio       *string  `json:"bio,omitempty"`
	Location  *string  `json:"location,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	Interests []uint   `json:"interests,omitempty"`
}

type DiscoverUsersRequest struct {
	AgeMin      *int     `json:"age_min,omitempty"`
	AgeMax      *int     `json:"age_max,omitempty"`
	Gender      *string  `json:"gender,omitempty"`
	Location    *string  `json:"location,omitempty"`
	Latitude    *float64 `json:"latitude,omitempty"`
	Longitude   *float64 `json:"longitude,omitempty"`
	MaxDistance *int     `json:"max_distance,omitempty"` // in kilometers
	Interests   []uint   `json:"interests,omitempty"`
	Page        int      `json:"page" binding:"min=1"`
	Limit       int      `json:"limit" binding:"min=1,max=50"`
}

type ReportUserRequest struct {
	ReportedID  uint   `json:"reported_id" binding:"required"`
	Reason      string `json:"reason" binding:"required"`
	Description string `json:"description,omitempty"`
}

func NewUserHandler(db *gorm.DB, redis *redis.Client, cfg *config.Config) *UserHandler {
	return &UserHandler{
		db:    db,
		redis: redis,
		cfg:   cfg,
	}
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var user models.User
	if err := h.db.Preload("ProfilePhotos").Preload("Interests").Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.db.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update fields
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.Bio != nil {
		user.Bio = req.Bio
	}
	if req.Location != nil {
		user.Location = req.Location
	}
	if req.Latitude != nil {
		user.Latitude = req.Latitude
	}
	if req.Longitude != nil {
		user.Longitude = req.Longitude
	}

	// Update interests if provided
	if len(req.Interests) > 0 {
		// Remove existing interests
		h.db.Where("user_id = ?", userID).Delete(&models.UserInterest{})

		// Add new interests
		for _, interestID := range req.Interests {
			userInterest := models.UserInterest{
				UserID:     userID.(uint),
				InterestID: interestID,
			}
			h.db.Create(&userInterest)
		}
	}

	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	// Reload user with relations
	h.db.Preload("ProfilePhotos").Preload("Interests").Where("id = ?", userID).First(&user)

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully", "user": user})
}

func (h *UserHandler) UploadPhoto(c *gin.Context) {
	userID, _ := c.Get("user_id")

	file, header, err := c.Request.FormFile("photo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No photo provided"})
		return
	}
	defer file.Close()

	// Validate file
	if err := h.validateImageFile(header); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("profile_photos/%d_%s%s", userID, uuid.New().String(), ext)

	// Upload to S3/MinIO
	url, err := h.uploadToStorage(file, filename, header.Header.Get("Content-Type"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload photo"})
		return
	}

	// Check if this is the first photo (make it primary)
	var photoCount int64
	h.db.Model(&models.ProfilePhoto{}).Where("user_id = ?", userID).Count(&photoCount)

	// Create photo record
	photo := models.ProfilePhoto{
		UserID:    userID.(uint),
		URL:       url,
		IsPrimary: photoCount == 0,
		Order:     int(photoCount),
	}

	if err := h.db.Create(&photo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save photo record"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Photo uploaded successfully", "photo": photo})
}

func (h *UserHandler) DeletePhoto(c *gin.Context) {
	userID, _ := c.Get("user_id")
	photoID := c.Param("id")

	var photo models.ProfilePhoto
	if err := h.db.Where("id = ? AND user_id = ?", photoID, userID).First(&photo).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	// Delete from storage
	if err := h.deleteFromStorage(photo.URL); err != nil {
		// Log error but continue with database deletion
		fmt.Printf("Failed to delete photo from storage: %v\n", err)
	}

	// Delete from database
	if err := h.db.Delete(&photo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete photo"})
		return
	}

	// If this was the primary photo, make another one primary
	if photo.IsPrimary {
		var nextPhoto models.ProfilePhoto
		if err := h.db.Where("user_id = ? AND id != ?", userID, photoID).First(&nextPhoto).Error; err == nil {
			nextPhoto.IsPrimary = true
			h.db.Save(&nextPhoto)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Photo deleted successfully"})
}

func (h *UserHandler) DiscoverUsers(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req DiscoverUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 20
	}

	// Get current user
	var currentUser models.User
	if err := h.db.Where("id = ?", userID).First(&currentUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Build query
	query := h.db.Model(&models.User{}).Where("id != ? AND is_active = ? AND is_verified = ?", userID, true, true)

	// Age filter
	if req.AgeMin != nil || req.AgeMax != nil {
		now := time.Now()
		if req.AgeMin != nil {
			maxBirthDate := now.AddDate(-*req.AgeMin, 0, 0)
			query = query.Where("date_of_birth <= ?", maxBirthDate)
		}
		if req.AgeMax != nil {
			minBirthDate := now.AddDate(-*req.AgeMax-1, 0, 0)
			query = query.Where("date_of_birth >= ?", minBirthDate)
		}
	}

	// Gender filter
	if req.Gender != nil {
		query = query.Where("gender = ?", *req.Gender)
	}

	// Location filter
	if req.Location != nil {
		query = query.Where("location ILIKE ?", "%"+*req.Location+"%")
	}

	// Distance filter (if coordinates provided)
	if req.Latitude != nil && req.Longitude != nil && req.MaxDistance != nil {
		// Simple distance calculation (not accurate for large distances)
		query = query.Where(
			"latitude IS NOT NULL AND longitude IS NOT NULL AND "+
				"SQRT(POW(latitude - ?, 2) + POW(longitude - ?, 2)) * 111 <= ?",
			*req.Latitude, *req.Longitude, *req.MaxDistance,
		)
	}

	// Exclude blocked users
	query = query.Where("id NOT IN (SELECT blocked_id FROM blocked_users WHERE blocker_id = ?)", userID)

	// Exclude already liked/disliked users
	query = query.Where("id NOT IN (SELECT liked_id FROM likes WHERE liker_id = ?)", userID)
	query = query.Where("id NOT IN (SELECT disliked_id FROM dislikes WHERE disliker_id = ?)", userID)

	// Get total count
	var total int64
	query.Count(&total)

	// Apply pagination
	offset := (req.Page - 1) * req.Limit
	var users []models.User
	if err := query.Preload("ProfilePhotos").Preload("Interests").
		Offset(offset).Limit(req.Limit).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	// Filter by interests if provided
	if len(req.Interests) > 0 {
		var filteredUsers []models.User
		for _, user := range users {
			userInterests := make(map[uint]bool)
			for _, interest := range user.Interests {
				userInterests[interest.ID] = true
			}

			hasMatchingInterest := false
			for _, interestID := range req.Interests {
				if userInterests[interestID] {
					hasMatchingInterest = true
					break
				}
			}

			if hasMatchingInterest {
				filteredUsers = append(filteredUsers, user)
			}
		}
		users = filteredUsers
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"pagination": gin.H{
			"page":        req.Page,
			"limit":       req.Limit,
			"total":       total,
			"total_pages": (total + int64(req.Limit) - 1) / int64(req.Limit),
		},
	})
}

func (h *UserHandler) GetFavorites(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var favorites []models.Favorite
	if err := h.db.Preload("Favorite.ProfilePhotos").Preload("Favorite.Interests").
		Where("user_id = ?", userID).Find(&favorites).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch favorites"})
		return
	}

	var users []models.User
	for _, fav := range favorites {
		users = append(users, fav.Favorite)
	}

	c.JSON(http.StatusOK, gin.H{"favorites": users})
}

func (h *UserHandler) AddToFavorites(c *gin.Context) {
	userID, _ := c.Get("user_id")
	favoriteID, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if user exists
	var user models.User
	if err := h.db.Where("id = ?", favoriteID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check if already in favorites
	var existing models.Favorite
	if err := h.db.Where("user_id = ? AND favorite_id = ?", userID, favoriteID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User already in favorites"})
		return
	}

	// Add to favorites
	favorite := models.Favorite{
		UserID:     userID.(uint),
		FavoriteID: uint(favoriteID),
	}

	if err := h.db.Create(&favorite).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add to favorites"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Added to favorites successfully"})
}

func (h *UserHandler) RemoveFromFavorites(c *gin.Context) {
	userID, _ := c.Get("user_id")
	favoriteID, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := h.db.Where("user_id = ? AND favorite_id = ?", userID, favoriteID).Delete(&models.Favorite{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove from favorites"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Removed from favorites successfully"})
}

func (h *UserHandler) BlockUser(c *gin.Context) {
	userID, _ := c.Get("user_id")
	blockedID, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if user exists
	var user models.User
	if err := h.db.Where("id = ?", blockedID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check if already blocked
	var existing models.BlockedUser
	if err := h.db.Where("blocker_id = ? AND blocked_id = ?", userID, blockedID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User already blocked"})
		return
	}

	// Block user
	blocked := models.BlockedUser{
		BlockerID: userID.(uint),
		BlockedID: uint(blockedID),
	}

	if err := h.db.Create(&blocked).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to block user"})
		return
	}

	// Remove from favorites if exists
	h.db.Where("user_id = ? AND favorite_id = ?", userID, blockedID).Delete(&models.Favorite{})

	c.JSON(http.StatusCreated, gin.H{"message": "User blocked successfully"})
}

func (h *UserHandler) UnblockUser(c *gin.Context) {
	userID, _ := c.Get("user_id")
	blockedID, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := h.db.Where("blocker_id = ? AND blocked_id = ?", userID, blockedID).Delete(&models.BlockedUser{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unblock user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User unblocked successfully"})
}

func (h *UserHandler) ReportUser(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req ReportUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if reported user exists
	var user models.User
	if err := h.db.Where("id = ?", req.ReportedID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check if already reported
	var existing models.Report
	if err := h.db.Where("reporter_id = ? AND reported_id = ?", userID, req.ReportedID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User already reported"})
		return
	}

	// Create report
	report := models.Report{
		ReporterID:  userID.(uint),
		ReportedID:  req.ReportedID,
		Reason:      req.Reason,
		Description: &req.Description,
		Status:      "pending",
	}

	if err := h.db.Create(&report).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create report"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User reported successfully"})
}

// Helper methods for file handling
func (h *UserHandler) validateImageFile(header *multipart.FileHeader) error {
	// Check file size
	if header.Size > h.cfg.MaxFileSize {
		return fmt.Errorf("file too large, maximum size is %d bytes", h.cfg.MaxFileSize)
	}

	// Check file type
	contentType := header.Header.Get("Content-Type")
	allowed := false
	for _, allowedType := range h.cfg.AllowedImageTypes {
		if contentType == allowedType {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("invalid file type, allowed types are: %s", strings.Join(h.cfg.AllowedImageTypes, ", "))
	}

	return nil
}

func (h *UserHandler) uploadToStorage(file multipart.File, filename, contentType string) (string, error) {
	// TODO: Implement actual S3/MinIO upload
	// For now, return a placeholder URL
	return fmt.Sprintf("https://storage.example.com/%s", filename), nil
}

func (h *UserHandler) deleteFromStorage(url string) error {
	// TODO: Implement actual S3/MinIO deletion
	return nil
}
