package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL            string
	RedisURL               string
	JWTSecret              string
	JWTExpiry              time.Duration
	Port                   string
	GinMode                string
	AWSAccessKeyID         string
	AWSSecretAccessKey     string
	AWSRegion              string
	S3Bucket               string
	MinIOEndpoint          string
	MinIOAccessKey         string
	MinIOSecretKey         string
	MinIOUseSSL            bool
	FirebaseProjectID      string
	FirebasePrivateKeyPath string
	OTPEnabled             bool
	OTPExpiry              time.Duration
	MaxFileSize            int64
	AllowedImageTypes      []string
}

func Load() *Config {
	return &Config{
		DatabaseURL:            getEnv("DATABASE_URL", "postgres://username:password@localhost:5432/ethiopia_dating_app?sslmode=disable"),
		RedisURL:               getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:              getEnv("JWT_SECRET", "your-super-secret-jwt-key-here"),
		JWTExpiry:              getDurationEnv("JWT_EXPIRY", 24*time.Hour),
		Port:                   getEnv("PORT", "8080"),
		GinMode:                getEnv("GIN_MODE", "debug"),
		AWSAccessKeyID:         getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey:     getEnv("AWS_SECRET_ACCESS_KEY", ""),
		AWSRegion:              getEnv("AWS_REGION", "us-east-1"),
		S3Bucket:               getEnv("S3_BUCKET", "ethiopia-dating-photos"),
		MinIOEndpoint:          getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:         getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey:         getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOUseSSL:            getBoolEnv("MINIO_USE_SSL", false),
		FirebaseProjectID:      getEnv("FIREBASE_PROJECT_ID", ""),
		FirebasePrivateKeyPath: getEnv("FIREBASE_PRIVATE_KEY_PATH", "./firebase-private-key.json"),
		OTPEnabled:             getBoolEnv("OTP_ENABLED", true),
		OTPExpiry:              getDurationEnv("OTP_EXPIRY", 5*time.Minute),
		MaxFileSize:            getInt64Env("MAX_FILE_SIZE", 10*1024*1024), // 10MB
		AllowedImageTypes:      []string{"image/jpeg", "image/png", "image/webp"},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
