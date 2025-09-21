package models

import (
	"time"

	"gorm.io/gorm"
)

type Admin struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	Email        string         `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash string         `json:"-" gorm:"not null"`
	FirstName    string         `json:"first_name" gorm:"not null"`
	LastName     string         `json:"last_name" gorm:"not null"`
	Role         string         `json:"role" gorm:"not null"` // super_admin, moderator, support
	IsActive     bool           `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

type Analytics struct {
	TotalUsers     int64     `json:"total_users"`
	ActiveUsers    int64     `json:"active_users"`
	NewUsersToday  int64     `json:"new_users_today"`
	TotalMatches   int64     `json:"total_matches"`
	MatchesToday   int64     `json:"matches_today"`
	TotalMessages  int64     `json:"total_messages"`
	MessagesToday  int64     `json:"messages_today"`
	PendingReports int64     `json:"pending_reports"`
	Date           time.Time `json:"date"`
}

type UserActivity struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null"`
	Action    string    `json:"action" gorm:"not null"` // login, logout, profile_update, etc.
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	User      User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
}
