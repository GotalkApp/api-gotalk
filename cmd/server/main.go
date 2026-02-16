package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/quocanhngo/gotalk/internal/config"
	"github.com/quocanhngo/gotalk/internal/handler"
	"github.com/quocanhngo/gotalk/internal/middleware"
	"github.com/quocanhngo/gotalk/internal/model"
	"github.com/quocanhngo/gotalk/internal/repository"
	"github.com/quocanhngo/gotalk/internal/service"
	"github.com/quocanhngo/gotalk/internal/ws"
	"github.com/quocanhngo/gotalk/migrations"
	"github.com/quocanhngo/gotalk/pkg/auth"
	"github.com/quocanhngo/gotalk/pkg/mailer"
	"github.com/quocanhngo/gotalk/pkg/storage"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// @title           GoTalk API
// @version         1.0
// @description     Real-time Chat & Video Call API with Go, Gin, WebSocket, Redis Pub/Sub.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@gotalk.local

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      api.localhost
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	// ==================== Load Config ====================
	cfg := config.Load()
	log.Printf("🚀 Starting GoTalk API Server [env=%s]", cfg.App.Env)

	// ==================== Database (PostgreSQL) ====================
	gormLogger := logger.Default.LogMode(logger.Info)
	if cfg.App.Env == "production" {
		gormLogger = logger.Default.LogMode(logger.Warn)
	}

	db, err := gorm.Open(postgres.Open(cfg.DB.DSN()), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	log.Println("✅ Connected to PostgreSQL")

	// ==================== Run Migrations ====================
	dbURL := cfg.DB.URL()
	if err := migrations.Run(dbURL); err != nil {
		log.Printf("⚠️  Migration warning: %v", err)
		log.Println("📦 Falling back to GORM AutoMigrate...")
		// Fallback to AutoMigrate if migration files fail
		if err := db.AutoMigrate(
			&model.User{},
			&model.OTPCode{},
			&model.Conversation{},
			&model.ConversationMember{},
			&model.Message{},
			&model.MessageAttachment{},
			&model.ReadReceipt{},
		); err != nil {
			log.Fatalf("❌ Failed to migrate database: %v", err)
		}
	}
	log.Println("✅ Database migrated successfully")

	// ==================== Redis ====================
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       0,
	})

	ctx := context.Background()
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatalf("❌ Failed to connect to Redis: %v", err)
	}
	log.Println("✅ Connected to Redis")

	// ==================== Email (SMTP / Mailpit) ====================
	mailClient := mailer.New(mailer.Config{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		Username: cfg.SMTP.Username,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
		FromName: cfg.SMTP.FromName,
	})
	log.Printf("📧 SMTP configured: %s:%s", cfg.SMTP.Host, cfg.SMTP.Port)

	// ==================== Initialize Layers ====================
	// JWT Manager
	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.Expiry)

	// Repositories
	userRepo := repository.NewUserRepository(db)
	otpRepo := repository.NewOTPRepository(db)
	convRepo := repository.NewConversationRepository(db)
	msgRepo := repository.NewMessageRepository(db)

	// Services
	// Services
	authService := service.NewAuthService(userRepo, otpRepo, jwtManager, mailClient, rdb, cfg.Google.ClientID)
	chatService := service.NewChatService(convRepo, msgRepo, userRepo)

	// WebSocket Hub (with Redis Pub/Sub for horizontal scaling)
	hub := ws.NewHub(rdb, func(userID uuid.UUID, online bool) {
		// Callback: update user online status in DB
		_ = userRepo.UpdateOnlineStatus(userID, online)
		log.Printf("👤 User %s is now %s", userID, map[bool]string{true: "ONLINE", false: "OFFLINE"}[online])
	})

	// Start Hub event loop
	hubCtx, hubCancel := context.WithCancel(context.Background())
	defer hubCancel()
	go hub.Run(hubCtx)

	// MinIO Storage
	minioStorage, err := storage.NewMinIO(storage.Config{
		Endpoint:  cfg.MinIO.Endpoint,
		PublicURL: cfg.MinIO.PublicURL,
		AccessKey: cfg.MinIO.AccessKey,
		SecretKey: cfg.MinIO.SecretKey,
		Bucket:    cfg.MinIO.Bucket,
		UseSSL:    cfg.MinIO.UseSSL,
	})
	if err != nil {
		log.Printf("⚠️  MinIO not available: %v (file upload disabled)", err)
	}
	if minioStorage != nil {
		log.Println("✅ Connected to MinIO")
	}

	// Handlers
	authHandler := handler.NewAuthHandler(authService)
	chatHandler := handler.NewChatHandler(chatService, hub)
	wsHandler := handler.NewWSHandler(hub, chatService, jwtManager)
	uploadHandler := handler.NewUploadHandler(minioStorage)

	// ==================== Gin Router ====================
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Swagger configuration
	// Serve swagger.json at /docs/swagger.json to avoid conflict with /swagger/* wildcard
	router.StaticFile("/docs/swagger.json", "./docs/swagger.json")

	// Swagger UI handling
	url := ginSwagger.URL("/docs/swagger.json") // Point to the relative path
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))

	// Global middleware
	router.Use(middleware.CORSMiddleware(cfg.CORS.Origins))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "gotalk-api",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// ==================== API Routes ====================
	api := router.Group("/api/v1")
	{
		// Auth routes (public)
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/verify-otp", authHandler.VerifyOTP)
			authGroup.POST("/resend-otp", authHandler.ResendOTP)
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/google", authHandler.GoogleLogin)
			authGroup.POST("/forgot-password", authHandler.ForgotPassword)
			authGroup.POST("/reset-password", authHandler.ResetPassword)
		}

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(jwtManager, rdb))
		{
			// Auth
			protected.POST("/auth/logout", authHandler.Logout)
			protected.GET("/auth/profile", authHandler.GetProfile)
			protected.GET("/users/search", authHandler.SearchUsers)

			// Conversations
			protected.GET("/conversations", chatHandler.GetConversations)
			protected.POST("/conversations", chatHandler.CreateConversation)
			protected.POST("/conversations/direct", chatHandler.GetOrCreateDirect)
			protected.GET("/conversations/:id", chatHandler.GetConversation)

			// Messages
			protected.GET("/conversations/:id/messages", chatHandler.GetMessages)
			protected.POST("/conversations/:id/messages", chatHandler.SendMessage)
			protected.POST("/conversations/:id/read", chatHandler.MarkAsRead)

			// Upload
			protected.POST("/upload", uploadHandler.UploadFile)
			protected.POST("/upload/multiple", uploadHandler.UploadMultiple)
		}
	}

	// WebSocket endpoint (auth via query parameter)
	router.GET("/ws", wsHandler.HandleWebSocket)

	// ==================== Start Server ====================
	srv := &http.Server{
		Addr:    ":" + cfg.App.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server failed: %v", err)
		}
	}()

	log.Printf("🌐 GoTalk API running on http://0.0.0.0:%s", cfg.App.Port)
	log.Printf("📋 API docs: http://0.0.0.0:%s/swagger/index.html", cfg.App.Port)
	log.Printf("📄 Swagger JSON: http://0.0.0.0:%s/docs/swagger.json", cfg.App.Port)
	log.Printf("🔌 WebSocket: ws://0.0.0.0:%s/ws?token=<jwt>", cfg.App.Port)
	log.Printf("📧 Mailpit UI: http://localhost:8025")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down server...")

	// Give ongoing requests 5 seconds to complete
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("❌ Server forced to shutdown: %v", err)
	}

	hubCancel()
	log.Println("✅ Server exited gracefully")
}
