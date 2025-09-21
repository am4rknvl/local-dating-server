package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"ethiopia-dating-app/internal/config"
	"ethiopia-dating-app/internal/models"
	"ethiopia-dating-app/internal/redis"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MatchHandler struct {
	db    *gorm.DB
	redis *redis.Client
	cfg   *config.Config
}

type MatchResponse struct {
	ID        uint        `json:"id"`
	User      models.User `json:"user"`
	CreatedAt time.Time   `json:"created_at"`
}

func NewMatchHandler(db *gorm.DB, redis *redis.Client, cfg *config.Config) *MatchHandler {
	return &MatchHandler{
		db:    db,
		redis: redis,
		cfg:   cfg,
	}
}

func (h *MatchHandler) LikeUser(c *gin.Context) {
	userID, _ := c.Get("user_id")
	likedID, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if user exists and is active
	var likedUser models.User
	if err := h.db.Where("id = ? AND is_active = ?", likedID, true).First(&likedUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check if already liked
	var existingLike models.Like
	if err := h.db.Where("liker_id = ? AND liked_id = ?", userID, likedID).First(&existingLike).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User already liked"})
		return
	}

	// Check if user is blocked
	var blocked models.BlockedUser
	if err := h.db.Where("blocker_id = ? AND blocked_id = ?", userID, likedID).First(&blocked).Error; err == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot like blocked user"})
		return
	}

	// Create like
	like := models.Like{
		LikerID: userID.(uint),
		LikedID: uint(likedID),
	}

	if err := h.db.Create(&like).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create like"})
		return
	}

	// Check for mutual like (match)
	var mutualLike models.Like
	if err := h.db.Where("liker_id = ? AND liked_id = ?", likedID, userID).First(&mutualLike).Error; err == nil {
		// Create match
		match := models.Match{
			User1ID:  userID.(uint),
			User2ID:  uint(likedID),
			IsActive: true,
		}

		if err := h.db.Create(&match).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create match"})
			return
		}

		// Create conversation
		conversation := models.Conversation{
			MatchID:  match.ID,
			IsActive: true,
		}

		if err := h.db.Create(&conversation).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation"})
			return
		}

		// Create notifications for both users
		h.createMatchNotification(userID.(uint), uint(likedID), match.ID)
		h.createMatchNotification(uint(likedID), userID.(uint), match.ID)

		// Cache match data in Redis
		h.cacheMatchData(match.ID, userID.(uint), uint(likedID))

		c.JSON(http.StatusCreated, gin.H{
			"message": "It's a match!",
			"match": gin.H{
				"id":         match.ID,
				"user":       likedUser,
				"created_at": match.CreatedAt,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User liked successfully"})
}

func (h *MatchHandler) DislikeUser(c *gin.Context) {
	userID, _ := c.Get("user_id")
	dislikedID, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if already disliked
	var existingDislike models.Dislike
	if err := h.db.Where("disliker_id = ? AND disliked_id = ?", userID, dislikedID).First(&existingDislike).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User already disliked"})
		return
	}

	// Create dislike
	dislike := models.Dislike{
		DislikerID: userID.(uint),
		DislikedID: uint(dislikedID),
	}

	if err := h.db.Create(&dislike).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create dislike"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User disliked successfully"})
}

func (h *MatchHandler) GetMatches(c *gin.Context) {
	userID, _ := c.Get("user_id")

	// Get matches where user is either user1 or user2
	var matches []models.Match
	if err := h.db.Where("(user1_id = ? OR user2_id = ?) AND is_active = ?", userID, userID, true).
		Preload("User1.ProfilePhotos").Preload("User1.Interests").
		Preload("User2.ProfilePhotos").Preload("User2.Interests").
		Order("created_at DESC").Find(&matches).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch matches"})
		return
	}

	var matchResponses []MatchResponse
	for _, match := range matches {
		var otherUser models.User
		if match.User1ID == userID.(uint) {
			otherUser = match.User2
		} else {
			otherUser = match.User1
		}

		matchResponses = append(matchResponses, MatchResponse{
			ID:        match.ID,
			User:      otherUser,
			CreatedAt: match.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"matches": matchResponses})
}

func (h *MatchHandler) Unmatch(c *gin.Context) {
	userID, _ := c.Get("user_id")
	matchID, err := strconv.ParseUint(c.Param("match_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid match ID"})
		return
	}

	// Find match
	var match models.Match
	if err := h.db.Where("id = ? AND (user1_id = ? OR user2_id = ?) AND is_active = ?",
		matchID, userID, userID, true).First(&match).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Match not found"})
		return
	}

	// Deactivate match
	match.IsActive = false
	if err := h.db.Save(&match).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmatch"})
		return
	}

	// Deactivate conversation
	var conversation models.Conversation
	if err := h.db.Where("match_id = ?", matchID).First(&conversation).Error; err == nil {
		conversation.IsActive = false
		h.db.Save(&conversation)
	}

	// Remove from Redis cache
	h.redis.Del(c.Request.Context(), "match:"+strconv.FormatUint(matchID, 10))

	c.JSON(http.StatusOK, gin.H{"message": "Unmatched successfully"})
}

// Helper methods
func (h *MatchHandler) createMatchNotification(userID, otherUserID, matchID uint) {
	notification := models.Notification{
		UserID: userID,
		Type:   "match",
		Title:  "New Match!",
		Body:   "You have a new match! Start chatting now.",
		Data:   `{"match_id": ` + strconv.FormatUint(uint64(matchID), 10) + `}`,
	}

	h.db.Create(&notification)

	// TODO: Send push notification
	// h.sendPushNotification(userID, notification.Title, notification.Body, notification.Data)
}

func (h *MatchHandler) cacheMatchData(matchID, user1ID, user2ID uint) {
	// Cache match data in Redis for quick access
	matchKey := "match:" + strconv.FormatUint(uint64(matchID), 10)
	matchData := map[string]interface{}{
		"id":         matchID,
		"user1_id":   user1ID,
		"user2_id":   user2ID,
		"created_at": time.Now().Unix(),
	}

	ctx := context.Background()
	h.redis.HSet(ctx, matchKey, matchData)
	h.redis.Expire(ctx, matchKey, 24*time.Hour)
}
