package database

import (
	"fmt"
	"log"

	"ethiopia-dating-app/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Initialize(databaseURL string) (*gorm.DB, error) {
	// Configure GORM
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(databaseURL), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Auto-migrate tables
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database connected and migrated successfully")
	return db, nil
}

func migrate(db *gorm.DB) error {
	// Enable UUID extension
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		log.Printf("Warning: Could not create uuid-ossp extension: %v", err)
	}

	// Auto-migrate all models
	return db.AutoMigrate(
		&models.User{},
		&models.ProfilePhoto{},
		&models.Interest{},
		&models.UserInterest{},
		&models.OTP{},
		&models.UserSession{},
		&models.BlockedUser{},
		&models.Report{},
		&models.Favorite{},
		&models.Match{},
		&models.Like{},
		&models.Dislike{},
		&models.Conversation{},
		&models.Message{},
		&models.Notification{},
		&models.Admin{},
		&models.UserActivity{},
	)
}

func SeedInterests(db *gorm.DB) error {
	interests := []models.Interest{
		{Name: "Music", Category: "Entertainment"},
		{Name: "Movies", Category: "Entertainment"},
		{Name: "Sports", Category: "Sports"},
		{Name: "Fitness", Category: "Sports"},
		{Name: "Travel", Category: "Lifestyle"},
		{Name: "Photography", Category: "Arts"},
		{Name: "Cooking", Category: "Lifestyle"},
		{Name: "Reading", Category: "Education"},
		{Name: "Gaming", Category: "Entertainment"},
		{Name: "Dancing", Category: "Arts"},
		{Name: "Art", Category: "Arts"},
		{Name: "Technology", Category: "Education"},
		{Name: "Nature", Category: "Lifestyle"},
		{Name: "Fashion", Category: "Lifestyle"},
		{Name: "Food", Category: "Lifestyle"},
		{Name: "Coffee", Category: "Lifestyle"},
		{Name: "Wine", Category: "Lifestyle"},
		{Name: "Adventure", Category: "Lifestyle"},
		{Name: "Yoga", Category: "Sports"},
		{Name: "Meditation", Category: "Lifestyle"},
		{Name: "Volunteering", Category: "Social"},
		{Name: "Politics", Category: "Social"},
		{Name: "Religion", Category: "Social"},
		{Name: "Family", Category: "Social"},
		{Name: "Career", Category: "Education"},
		{Name: "Business", Category: "Education"},
		{Name: "Science", Category: "Education"},
		{Name: "History", Category: "Education"},
		{Name: "Languages", Category: "Education"},
		{Name: "Culture", Category: "Social"},
	}

	for _, interest := range interests {
		if err := db.FirstOrCreate(&interest, models.Interest{Name: interest.Name}).Error; err != nil {
			return fmt.Errorf("failed to seed interest %s: %w", interest.Name, err)
		}
	}

	log.Println("Interests seeded successfully")
	return nil
}
