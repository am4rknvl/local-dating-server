package models

import (
	"time"

	"gorm.io/gorm"
)

type Match struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	User1ID   uint           `json:"user1_id" gorm:"not null"`
	User2ID   uint           `json:"user2_id" gorm:"not null"`
	IsActive  bool           `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	User1     User           `json:"user1,omitempty" gorm:"foreignKey:User1ID"`
	User2     User           `json:"user2,omitempty" gorm:"foreignKey:User2ID"`
}

type Like struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	LikerID   uint      `json:"liker_id" gorm:"not null"`
	LikedID   uint      `json:"liked_id" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	Liker     User      `json:"liker,omitempty" gorm:"foreignKey:LikerID"`
	Liked     User      `json:"liked,omitempty" gorm:"foreignKey:LikedID"`
}

type Dislike struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	DislikerID uint      `json:"disliker_id" gorm:"not null"`
	DislikedID uint      `json:"disliked_id" gorm:"not null"`
	CreatedAt  time.Time `json:"created_at"`
	Disliker   User      `json:"disliker,omitempty" gorm:"foreignKey:DislikerID"`
	Disliked   User      `json:"disliked,omitempty" gorm:"foreignKey:DislikedID"`
}

type Conversation struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	MatchID   uint           `json:"match_id" gorm:"not null"`
	IsActive  bool           `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	Match     Match          `json:"match,omitempty" gorm:"foreignKey:MatchID"`
	Messages  []Message      `json:"messages,omitempty"`
}

type Message struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	ConversationID uint           `json:"conversation_id" gorm:"not null"`
	SenderID       uint           `json:"sender_id" gorm:"not null"`
	Content        string         `json:"content" gorm:"not null"`
	MessageType    string         `json:"message_type" gorm:"default:text"` // text, image, emoji
	IsRead         bool           `json:"is_read" gorm:"default:false"`
	ReadAt         *time.Time     `json:"read_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
	Conversation   Conversation   `json:"conversation,omitempty" gorm:"foreignKey:ConversationID"`
	Sender         User           `json:"sender,omitempty" gorm:"foreignKey:SenderID"`
}

type Notification struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null"`
	Type      string    `json:"type" gorm:"not null"` // match, message, like, etc.
	Title     string    `json:"title" gorm:"not null"`
	Body      string    `json:"body" gorm:"not null"`
	Data      string    `json:"data" gorm:"type:jsonb"` // Additional data as JSON
	IsRead    bool      `json:"is_read" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
	User      User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
}
