package handlers

import (
	"net/http"
	"strconv"
	"time"

	"ethiopia-dating-app/internal/config"
	"ethiopia-dating-app/internal/models"
	"ethiopia-dating-app/internal/redis"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db    *gorm.DB
	redis *redis.Client
	cfg   *config.Config
}

type UpdateUserStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active inactive suspended"`
}

type UpdateReportStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=pending reviewed resolved dismissed"`
}

type UserListResponse struct {
	Users []models.User `json:"users"`
	Total int64         `json:"total"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
}

type ReportListResponse struct {
	Reports []models.Report `json:"reports"`
	Total   int64           `json:"total"`
	Page    int             `json:"page"`
	Limit   int             `json:"limit"`
}

func NewAdminHandler(db *gorm.DB, redis *redis.Client, cfg *config.Config) *AdminHandler {
	return &AdminHandler{
		db:    db,
		redis: redis,
		cfg:   cfg,
	}
}

func (h *AdminHandler) GetUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	status := c.Query("status")
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Build query
	query := h.db.Model(&models.User{})

	// Filter by status
	if status != "" {
		switch status {
		case "active":
			query = query.Where("is_active = ?", true)
		case "inactive":
			query = query.Where("is_active = ?", false)
		case "verified":
			query = query.Where("is_verified = ?", true)
		case "unverified":
			query = query.Where("is_verified = ?", false)
		}
	}

	// Search by name or email
	if search != "" {
		query = query.Where("(first_name ILIKE ? OR last_name ILIKE ? OR email ILIKE ?)",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get users
	var users []models.User
	if err := query.Preload("ProfilePhotos").
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, UserListResponse{
		Users: users,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

func (h *AdminHandler) GetUser(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	if err := h.db.Preload("ProfilePhotos").Preload("Interests").
		Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Get user activity
	var activities []models.UserActivity
	h.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(10).Find(&activities)

	// Get reports against this user
	var reports []models.Report
	h.db.Preload("Reporter").Where("reported_id = ?", userID).Find(&reports)

	c.JSON(http.StatusOK, gin.H{
		"user":       user,
		"activities": activities,
		"reports":    reports,
	})
}

func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req UpdateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.db.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update status
	switch req.Status {
	case "active":
		user.IsActive = true
	case "inactive":
		user.IsActive = false
	case "suspended":
		user.IsActive = false
		// You might want to add a separate suspended field
	}

	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user status"})
		return
	}

	// Log admin action
	adminID, _ := c.Get("user_id")
	activity := models.UserActivity{
		UserID:    uint(userID),
		Action:    "status_updated",
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}
	h.db.Create(&activity)

	c.JSON(http.StatusOK, gin.H{"message": "User status updated successfully"})
}

func (h *AdminHandler) GetReports(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Build query
	query := h.db.Model(&models.Report{})

	// Filter by status
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get reports
	var reports []models.Report
	if err := query.Preload("Reporter").Preload("Reported").
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reports"})
		return
	}

	c.JSON(http.StatusOK, ReportListResponse{
		Reports: reports,
		Total:   total,
		Page:    page,
		Limit:   limit,
	})
}

func (h *AdminHandler) UpdateReportStatus(c *gin.Context) {
	reportID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report ID"})
		return
	}

	var req UpdateReportStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var report models.Report
	if err := h.db.Where("id = ?", reportID).First(&report).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Report not found"})
		return
	}

	// Update status
	report.Status = req.Status
	if err := h.db.Save(&report).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update report status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Report status updated successfully"})
}

func (h *AdminHandler) GetAnalytics(c *gin.Context) {
	// Get analytics for the last 30 days
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	// Total users
	var totalUsers int64
	h.db.Model(&models.User{}).Count(&totalUsers)

	// Active users (logged in within last 7 days)
	var activeUsers int64
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	h.db.Model(&models.User{}).Where("last_seen > ?", sevenDaysAgo).Count(&activeUsers)

	// New users today
	var newUsersToday int64
	today := time.Now().Truncate(24 * time.Hour)
	h.db.Model(&models.User{}).Where("created_at >= ?", today).Count(&newUsersToday)

	// Total matches
	var totalMatches int64
	h.db.Model(&models.Match{}).Where("is_active = ?", true).Count(&totalMatches)

	// Matches today
	var matchesToday int64
	h.db.Model(&models.Match{}).Where("is_active = ? AND created_at >= ?", true, today).Count(&matchesToday)

	// Total messages
	var totalMessages int64
	h.db.Model(&models.Message{}).Count(&totalMessages)

	// Messages today
	var messagesToday int64
	h.db.Model(&models.Message{}).Where("created_at >= ?", today).Count(&messagesToday)

	// Pending reports
	var pendingReports int64
	h.db.Model(&models.Report{}).Where("status = ?", "pending").Count(&pendingReports)

	// User registrations by day (last 30 days)
	var dailyRegistrations []struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}
	h.db.Model(&models.User{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= ?", thirtyDaysAgo).
		Group("DATE(created_at)").
		Order("date").
		Scan(&dailyRegistrations)

	// Gender distribution
	var genderDistribution []struct {
		Gender string `json:"gender"`
		Count  int64  `json:"count"`
	}
	h.db.Model(&models.User{}).
		Select("gender, COUNT(*) as count").
		Group("gender").
		Scan(&genderDistribution)

	analytics := models.Analytics{
		TotalUsers:     totalUsers,
		ActiveUsers:    activeUsers,
		NewUsersToday:  newUsersToday,
		TotalMatches:   totalMatches,
		MatchesToday:   matchesToday,
		TotalMessages:  totalMessages,
		MessagesToday:  messagesToday,
		PendingReports: pendingReports,
		Date:           time.Now(),
	}

	c.JSON(http.StatusOK, gin.H{
		"analytics":           analytics,
		"daily_registrations": dailyRegistrations,
		"gender_distribution": genderDistribution,
	})
}
