package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"ethiopia-dating-app/internal/config"
	"ethiopia-dating-app/internal/models"
	"ethiopia-dating-app/internal/redis"
	"ethiopia-dating-app/internal/websocket"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MessageHandler struct {
	db    *gorm.DB
	redis *redis.Client
	cfg   *config.Config
	hub   *websocket.Hub
}

type SendMessageRequest struct {
	Content     string `json:"content" binding:"required"`
	MessageType string `json:"message_type" binding:"omitempty,oneof=text image emoji"`
}

type ConversationResponse struct {
	ID          uint            `json:"id"`
	MatchID     uint            `json:"match_id"`
	OtherUser   models.User     `json:"other_user"`
	LastMessage *models.Message `json:"last_message,omitempty"`
	UnreadCount int64           `json:"unread_count"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type MessageResponse struct {
	ID          uint        `json:"id"`
	SenderID    uint        `json:"sender_id"`
	Content     string      `json:"content"`
	MessageType string      `json:"message_type"`
	IsRead      bool        `json:"is_read"`
	ReadAt      *time.Time  `json:"read_at,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	Sender      models.User `json:"sender,omitempty"`
}

func NewMessageHandler(db *gorm.DB, redis *redis.Client, cfg *config.Config, hub *websocket.Hub) *MessageHandler {
	return &MessageHandler{
		db:    db,
		redis: redis,
		cfg:   cfg,
		hub:   hub,
	}
}

func (h *MessageHandler) GetConversations(c *gin.Context) {
	userID, _ := c.Get("user_id")

	// Get all matches for the user
	var matches []models.Match
	if err := h.db.Where("(user1_id = ? OR user2_id = ?) AND is_active = ?", userID, userID, true).
		Preload("User1.ProfilePhotos").Preload("User2.ProfilePhotos").
		Find(&matches).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch matches"})
		return
	}

	var conversations []ConversationResponse
	for _, match := range matches {
		// Get conversation for this match
		var conversation models.Conversation
		if err := h.db.Where("match_id = ? AND is_active = ?", match.ID, true).First(&conversation).Error; err != nil {
			continue // Skip if no conversation exists
		}

		// Determine the other user
		var otherUser models.User
		if match.User1ID == userID.(uint) {
			otherUser = match.User2
		} else {
			otherUser = match.User1
		}

		// Get last message
		var lastMessage models.Message
		h.db.Where("conversation_id = ?", conversation.ID).
			Order("created_at DESC").First(&lastMessage)

		// Get unread count
		var unreadCount int64
		h.db.Model(&models.Message{}).
			Where("conversation_id = ? AND sender_id != ? AND is_read = ?",
				conversation.ID, userID, false).Count(&unreadCount)

		conversations = append(conversations, ConversationResponse{
			ID:          conversation.ID,
			MatchID:     match.ID,
			OtherUser:   otherUser,
			LastMessage: &lastMessage,
			UnreadCount: unreadCount,
			CreatedAt:   conversation.CreatedAt,
			UpdatedAt:   conversation.UpdatedAt,
		})
	}

	// Sort by last message time
	for i := 0; i < len(conversations)-1; i++ {
		for j := i + 1; j < len(conversations); j++ {
			if conversations[i].LastMessage != nil && conversations[j].LastMessage != nil {
				if conversations[i].LastMessage.CreatedAt.Before(conversations[j].LastMessage.CreatedAt) {
					conversations[i], conversations[j] = conversations[j], conversations[i]
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"conversations": conversations})
}

func (h *MessageHandler) GetMessages(c *gin.Context) {
	userID, _ := c.Get("user_id")
	conversationID, err := strconv.ParseUint(c.Param("conversation_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	// Verify user has access to this conversation
	if !h.userHasAccessToConversation(userID.(uint), uint(conversationID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this conversation"})
		return
	}

	// Get messages
	var messages []models.Message
	if err := h.db.Where("conversation_id = ?", conversationID).
		Preload("Sender").
		Order("created_at ASC").Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	// Mark messages as read
	h.db.Model(&models.Message{}).
		Where("conversation_id = ? AND sender_id != ? AND is_read = ?",
			conversationID, userID, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": time.Now(),
		})

	var messageResponses []MessageResponse
	for _, msg := range messages {
		messageResponses = append(messageResponses, MessageResponse{
			ID:          msg.ID,
			SenderID:    msg.SenderID,
			Content:     msg.Content,
			MessageType: msg.MessageType,
			IsRead:      msg.IsRead,
			ReadAt:      msg.ReadAt,
			CreatedAt:   msg.CreatedAt,
			Sender:      msg.Sender,
		})
	}

	c.JSON(http.StatusOK, gin.H{"messages": messageResponses})
}

func (h *MessageHandler) SendMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	conversationID, err := strconv.ParseUint(c.Param("conversation_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default message type
	if req.MessageType == "" {
		req.MessageType = "text"
	}

	// Verify user has access to this conversation
	if !h.userHasAccessToConversation(userID.(uint), uint(conversationID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this conversation"})
		return
	}

	// Create message
	message := models.Message{
		ConversationID: uint(conversationID),
		SenderID:       userID.(uint),
		Content:        req.Content,
		MessageType:    req.MessageType,
		IsRead:         false,
	}

	if err := h.db.Create(&message).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	// Load sender information
	h.db.Preload("Sender").First(&message, message.ID)

	// Update conversation timestamp
	h.db.Model(&models.Conversation{}).
		Where("id = ?", conversationID).
		Update("updated_at", time.Now())

	// Broadcast message via WebSocket
	messageData := websocket.Message{
		Type:           "message",
		ConversationID: uint(conversationID),
		SenderID:       userID.(uint),
		Content:        req.Content,
		MessageType:    req.MessageType,
		Timestamp:      message.CreatedAt.Format(time.RFC3339),
	}

	if messageBytes, err := json.Marshal(messageData); err == nil {
		h.hub.BroadcastToConversation(uint(conversationID), messageBytes)
	}

	// Create notification for the other user
	h.createMessageNotification(uint(conversationID), userID.(uint), req.Content)

	// Return the created message
	messageResponse := MessageResponse{
		ID:          message.ID,
		SenderID:    message.SenderID,
		Content:     message.Content,
		MessageType: message.MessageType,
		IsRead:      message.IsRead,
		ReadAt:      message.ReadAt,
		CreatedAt:   message.CreatedAt,
		Sender:      message.Sender,
	}

	c.JSON(http.StatusCreated, gin.H{"message": messageResponse})
}

func (h *MessageHandler) MarkAsRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	conversationID, err := strconv.ParseUint(c.Param("conversation_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	// Verify user has access to this conversation
	if !h.userHasAccessToConversation(userID.(uint), uint(conversationID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this conversation"})
		return
	}

	// Mark all messages in this conversation as read
	if err := h.db.Model(&models.Message{}).
		Where("conversation_id = ? AND sender_id != ? AND is_read = ?",
			conversationID, userID, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": time.Now(),
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark messages as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Messages marked as read"})
}

// Helper methods
func (h *MessageHandler) userHasAccessToConversation(userID, conversationID uint) bool {
	// Check if user is part of the match that owns this conversation
	var count int64
	h.db.Table("conversations").
		Joins("JOIN matches ON conversations.match_id = matches.id").
		Where("conversations.id = ? AND (matches.user1_id = ? OR matches.user2_id = ?) AND conversations.is_active = ?",
			conversationID, userID, userID, true).
		Count(&count)

	return count > 0
}

func (h *MessageHandler) createMessageNotification(conversationID, senderID uint, content string) {
	// Get the other user in the conversation
	var otherUserID uint
	h.db.Table("conversations").
		Joins("JOIN matches ON conversations.match_id = matches.id").
		Select("CASE WHEN matches.user1_id = ? THEN matches.user2_id ELSE matches.user1_id END", senderID).
		Where("conversations.id = ?", conversationID).
		Scan(&otherUserID)

	if otherUserID == 0 {
		return
	}

	// Create notification
	notification := models.Notification{
		UserID: otherUserID,
		Type:   "message",
		Title:  "New Message",
		Body:   content,
		Data:   `{"conversation_id": ` + strconv.FormatUint(uint64(conversationID), 10) + `}`,
	}

	h.db.Create(&notification)

	// TODO: Send push notification
	// h.sendPushNotification(otherUserID, notification.Title, notification.Body, notification.Data)
}
