package main

import (
	"log"
	"os"

	"ethiopia-dating-app/internal/config"
	"ethiopia-dating-app/internal/database"
	"ethiopia-dating-app/internal/handlers"
	"ethiopia-dating-app/internal/middleware"
	"ethiopia-dating-app/internal/redis"
	"ethiopia-dating-app/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Initialize(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Initialize Redis
	redisClient, err := redis.Initialize(cfg.RedisURL)
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, redisClient, cfg)
	userHandler := handlers.NewUserHandler(db, redisClient, cfg)
	matchHandler := handlers.NewMatchHandler(db, redisClient, cfg)
	messageHandler := handlers.NewMessageHandler(db, redisClient, cfg, hub)
	adminHandler := handlers.NewAdminHandler(db, redisClient, cfg)

	// Setup routes
	router := setupRoutes(authHandler, userHandler, matchHandler, messageHandler, adminHandler, hub)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func setupRoutes(authHandler *handlers.AuthHandler, userHandler *handlers.UserHandler, 
	matchHandler *handlers.MatchHandler, messageHandler *handlers.MessageHandler, 
	adminHandler *handlers.AdminHandler, hub *websocket.Hub) *gin.Engine {
	
	router := gin.Default()

	// CORS middleware
	router.Use(middleware.CORS())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Authentication routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/verify-otp", authHandler.VerifyOTP)
			auth.POST("/resend-otp", authHandler.ResendOTP)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", middleware.AuthRequired(), authHandler.Logout)
		}

		// User routes
		users := v1.Group("/users")
		users.Use(middleware.AuthRequired())
		{
			users.GET("/profile", userHandler.GetProfile)
			users.PUT("/profile", userHandler.UpdateProfile)
			users.POST("/profile/photo", userHandler.UploadPhoto)
			users.DELETE("/profile/photo/:id", userHandler.DeletePhoto)
			users.GET("/discover", userHandler.DiscoverUsers)
			users.GET("/favorites", userHandler.GetFavorites)
			users.POST("/favorites/:user_id", userHandler.AddToFavorites)
			users.DELETE("/favorites/:user_id", userHandler.RemoveFromFavorites)
			users.POST("/block/:user_id", userHandler.BlockUser)
			users.DELETE("/block/:user_id", userHandler.UnblockUser)
			users.POST("/report", userHandler.ReportUser)
		}

		// Matching routes
		matches := v1.Group("/matches")
		matches.Use(middleware.AuthRequired())
		{
			matches.POST("/like/:user_id", matchHandler.LikeUser)
			matches.POST("/dislike/:user_id", matchHandler.DislikeUser)
			matches.GET("/", matchHandler.GetMatches)
			matches.DELETE("/:match_id", matchHandler.Unmatch)
		}

		// Messaging routes
		messages := v1.Group("/messages")
		messages.Use(middleware.AuthRequired())
		{
			messages.GET("/conversations", messageHandler.GetConversations)
			messages.GET("/conversations/:conversation_id", messageHandler.GetMessages)
			messages.POST("/conversations/:conversation_id", messageHandler.SendMessage)
			messages.PUT("/conversations/:conversation_id/read", messageHandler.MarkAsRead)
		}

		// WebSocket endpoint
		v1.GET("/ws", middleware.AuthRequired(), func(c *gin.Context) {
			websocket.HandleWebSocket(hub, c)
		})

		// Admin routes
		admin := v1.Group("/admin")
		admin.Use(middleware.AuthRequired(), middleware.AdminRequired())
		{
			admin.GET("/users", adminHandler.GetUsers)
			admin.GET("/users/:id", adminHandler.GetUser)
			admin.PUT("/users/:id/status", adminHandler.UpdateUserStatus)
			admin.GET("/reports", adminHandler.GetReports)
			admin.PUT("/reports/:id/status", adminHandler.UpdateReportStatus)
			admin.GET("/analytics", adminHandler.GetAnalytics)
		}
	}

	return router
}
