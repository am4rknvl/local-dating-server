package handlers

import (
	"net/http"
	"strconv"
	"time"

	"ethiopia-dating-app/internal/config"
	"ethiopia-dating-app/internal/models"
	"ethiopia-dating-app/internal/redis"
	"ethiopia-dating-app/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db    *gorm.DB
	redis *redis.Client
	cfg   *config.Config
}

type RegisterRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Phone       string `json:"phone,omitempty"`
	Password    string `json:"password" binding:"required,min=8"`
	FirstName   string `json:"first_name" binding:"required"`
	LastName    string `json:"last_name" binding:"required"`
	DateOfBirth string `json:"date_of_birth" binding:"required"`
	Gender      string `json:"gender" binding:"required,oneof=male female other"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type VerifyOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func NewAuthHandler(db *gorm.DB, redis *redis.Client, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		db:    db,
		redis: redis,
		cfg:   cfg,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse date of birth
	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
		return
	}

	// Check if user is 18+
	age := time.Since(dob).Hours() / 24 / 365
	if age < 18 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You must be 18 or older to use this app"})
		return
	}

	// Check if user already exists
	var existingUser models.User
	if err := h.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User already exists with this email"})
		return
	}

	// Format phone number if provided
	var phone *string
	if req.Phone != "" {
		formattedPhone := utils.FormatPhoneNumber(req.Phone)
		phone = &formattedPhone

		// Check if phone already exists
		if err := h.db.Where("phone = ?", formattedPhone).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists with this phone number"})
			return
		}
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	// Create user
	user := models.User{
		Email:        req.Email,
		Phone:        phone,
		PasswordHash: hashedPassword,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		DateOfBirth:  dob,
		Gender:       req.Gender,
		IsVerified:   !h.cfg.OTPEnabled, // Auto-verify if OTP is disabled
		IsActive:     true,
	}

	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Generate OTP if enabled
	if h.cfg.OTPEnabled {
		otp, err := utils.GenerateOTP()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
			return
		}

		otpRecord := models.OTP{
			Email:     req.Email,
			Phone:     phone,
			Code:      otp,
			ExpiresAt: time.Now().Add(h.cfg.OTPExpiry),
		}

		if err := h.db.Create(&otpRecord).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create OTP"})
			return
		}

		// TODO: Send OTP via SMS/Email
		// For now, return OTP in response for development
		c.JSON(http.StatusCreated, gin.H{
			"message": "User created successfully. Please verify your account.",
			"otp":     otp, // Remove this in production
		})
		return
	}

	// Generate tokens
	accessToken, err := utils.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// Store session in Redis
	sessionKey := "session:" + strconv.FormatUint(uint64(user.ID), 10)
	sessionData := map[string]interface{}{
		"user_id":       user.ID,
		"email":         user.Email,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_at":    time.Now().Add(h.cfg.JWTExpiry).Unix(),
	}

	if err := h.redis.HSet(c.Request.Context(), sessionKey, sessionData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store session"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "User created successfully",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          user,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user
	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check if user is active
	if !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is deactivated"})
		return
	}

	// Verify password
	valid, err := utils.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate tokens
	accessToken, err := utils.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// Store session in Redis
	sessionKey := "session:" + strconv.FormatUint(uint64(user.ID), 10)
	sessionData := map[string]interface{}{
		"user_id":       user.ID,
		"email":         user.Email,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_at":    time.Now().Add(h.cfg.JWTExpiry).Unix(),
	}

	if err := h.redis.HSet(c.Request.Context(), sessionKey, sessionData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store session"})
		return
	}

	// Update last seen
	user.LastSeen = &[]time.Time{time.Now()}[0]
	user.IsOnline = true
	h.db.Save(&user)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          user,
	})
}

func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find OTP record
	var otp models.OTP
	if err := h.db.Where("email = ? AND code = ? AND is_used = ?", req.Email, req.Code, false).First(&otp).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired OTP"})
		return
	}

	// Check if OTP is expired
	if utils.IsOTPExpired(otp.CreatedAt, h.cfg.OTPExpiry) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OTP has expired"})
		return
	}

	// Mark OTP as used
	otp.IsUsed = true
	h.db.Save(&otp)

	// Verify user
	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
		return
	}

	user.IsVerified = true
	h.db.Save(&user)

	// Generate tokens
	accessToken, err := utils.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Account verified successfully",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          user,
	})
}

func (h *AuthHandler) ResendOTP(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user exists
	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Generate new OTP
	otp, err := utils.GenerateOTP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
		return
	}

	// Create new OTP record
	otpRecord := models.OTP{
		Email:     req.Email,
		Phone:     user.Phone,
		Code:      otp,
		ExpiresAt: time.Now().Add(h.cfg.OTPExpiry),
	}

	if err := h.db.Create(&otpRecord).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create OTP"})
		return
	}

	// TODO: Send OTP via SMS/Email
	c.JSON(http.StatusOK, gin.H{
		"message": "OTP sent successfully",
		"otp":     otp, // Remove this in production
	})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate refresh token
	claims, err := utils.ValidateToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Find user
	var user models.User
	if err := h.db.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Generate new tokens
	accessToken, err := utils.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Remove session from Redis
	sessionKey := "session:" + strconv.FormatUint(uint64(userID.(uint)), 10)
	h.redis.Del(c.Request.Context(), sessionKey)

	// Update user online status
	var user models.User
	if err := h.db.Where("id = ?", userID).First(&user).Error; err == nil {
		user.IsOnline = false
		user.LastSeen = &[]time.Time{time.Now()}[0]
		h.db.Save(&user)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}
