package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	Email         string         `json:"email" gorm:"uniqueIndex;not null"`
	Phone         *string        `json:"phone,omitempty" gorm:"uniqueIndex"`
	PasswordHash  string         `json:"-" gorm:"not null"`
	FirstName     string         `json:"first_name" gorm:"not null"`
	LastName      string         `json:"last_name" gorm:"not null"`
	DateOfBirth   time.Time      `json:"date_of_birth" gorm:"not null"`
	Gender        string         `json:"gender" gorm:"not null"` // male, female, other
	Bio           *string        `json:"bio,omitempty"`
	Location      *string        `json:"location,omitempty"`
	Latitude      *float64       `json:"latitude,omitempty"`
	Longitude     *float64       `json:"longitude,omitempty"`
	IsVerified    bool           `json:"is_verified" gorm:"default:false"`
	IsActive      bool           `json:"is_active" gorm:"default:true"`
	IsOnline      bool           `json:"is_online" gorm:"default:false"`
	LastSeen      *time.Time     `json:"last_seen,omitempty"`
	ProfilePhotos []ProfilePhoto `json:"profile_photos,omitempty"`
	Interests     []Interest     `json:"interests,omitempty" gorm:"many2many:user_interests;"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

type ProfilePhoto struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    uint           `json:"user_id" gorm:"not null"`
	URL       string         `json:"url" gorm:"not null"`
	IsPrimary bool           `json:"is_primary" gorm:"default:false"`
	Order     int            `json:"order" gorm:"default:0"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	User      User           `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

type Interest struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Name      string         `json:"name" gorm:"uniqueIndex;not null"`
	Category  string         `json:"category" gorm:"not null"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type UserInterest struct {
	UserID     uint      `json:"user_id" gorm:"primaryKey"`
	InterestID uint      `json:"interest_id" gorm:"primaryKey"`
	CreatedAt  time.Time `json:"created_at"`
}

type OTP struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email" gorm:"not null"`
	Phone     *string   `json:"phone,omitempty"`
	Code      string    `json:"code" gorm:"not null"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	IsUsed    bool      `json:"is_used" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
}

type UserSession struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null"`
	Token     string    `json:"token" gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	User      User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

type BlockedUser struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	BlockerID uint      `json:"blocker_id" gorm:"not null"`
	BlockedID uint      `json:"blocked_id" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	Blocker   User      `json:"blocker,omitempty" gorm:"foreignKey:BlockerID"`
	Blocked   User      `json:"blocked,omitempty" gorm:"foreignKey:BlockedID"`
}

type Report struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	ReporterID  uint      `json:"reporter_id" gorm:"not null"`
	ReportedID  uint      `json:"reported_id" gorm:"not null"`
	Reason      string    `json:"reason" gorm:"not null"`
	Description *string   `json:"description,omitempty"`
	Status      string    `json:"status" gorm:"default:pending"` // pending, reviewed, resolved, dismissed
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Reporter    User      `json:"reporter,omitempty" gorm:"foreignKey:ReporterID"`
	Reported    User      `json:"reported,omitempty" gorm:"foreignKey:ReportedID"`
}

type Favorite struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"user_id" gorm:"not null"`
	FavoriteID uint      `json:"favorite_id" gorm:"not null"`
	CreatedAt  time.Time `json:"created_at"`
	User       User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Favorite   User      `json:"favorite,omitempty" gorm:"foreignKey:FavoriteID"`
}
