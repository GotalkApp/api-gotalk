package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/quocanhngo/gotalk/internal/config"
	"github.com/quocanhngo/gotalk/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Load config
	cfg := config.Load()
	
	// Force DB logging off to avoid noise
	db, err := gorm.Open(postgres.Open(cfg.DB.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	log.Println("✅ Connected to Database")

	// Common password for all users
	password := "password123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("❌ Failed to hash password: %v", err)
	}

	// Create 10 users
	log.Println("🌱 Seeding 10 users...")
	
	for i := 1; i <= 10; i++ {
		username := fmt.Sprintf("user%d", i)
		email := fmt.Sprintf("user%d@gotalk.local", i)
		
		// Check if exists
		var existing model.User
		if err := db.Where("email = ?", email).First(&existing).Error; err == nil {
			if existing.Name == "" {
				db.Model(&existing).Update("name", fmt.Sprintf("User Number %d", i))
				log.Printf("🔄 Updated user name: %s", username)
			}
			continue
		}

		now := time.Now()
		user := model.User{
			ID:              uuid.New(),
			Name:            fmt.Sprintf("User Number %d", i),
			Email:           email,
			Password:        string(hashedPassword),
			AuthProvider:    model.AuthProviderEmail,
			EmailVerifiedAt: &now, // Verified immediately
			IsOnline:        i%3 == 0, // Randomly online (user3, user6, user9)
			Avatar:          fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s", username), // Random avatar
		}

		if err := db.Create(&user).Error; err != nil {
			log.Printf("❌ Failed to create user %s: %v", username, err)
		} else {
			log.Printf("✅ Created user: %s | Email: %s | Pass: %s", username, email, password)
		}
	}

	// Create a demo group conversation
	seedGroupChat(db)

	log.Println("🎉 Seeding completed!")
}

func seedGroupChat(db *gorm.DB) {
	// Find first 3 users
	var users []model.User
	if err := db.Limit(3).Find(&users).Error; err != nil || len(users) < 3 {
		return
	}

	admin := users[0]
	members := users[1:] // user2, user3

	// Check if group exists
	var count int64
	db.Model(&model.Conversation{}).Where("name = ?", "General Chat").Count(&count)
	if count > 0 {
		return
	}

	// Create Group
	group := model.Conversation{
		ID:        uuid.New(),
		Name:      "General Chat",
		Type:      model.ConversationTypeGroup,
		Avatar:    "https://api.dicebear.com/7.x/initials/svg?seed=GC",
		CreatorID: &admin.ID,
	}

	if err := db.Create(&group).Error; err != nil {
		log.Printf("❌ Failed to create group: %v", err)
		return
	}

	// Add Admin
	db.Create(&model.ConversationMember{
		ConversationID: group.ID,
		UserID:         admin.ID,
		Role:           model.MemberRoleAdmin,
	})

	// Add Members
	for _, m := range members {
		db.Create(&model.ConversationMember{
			ConversationID: group.ID,
			UserID:         m.ID,
			Role:           model.MemberRoleMember,
		})
	}
	
	// Add a welcome message
	msg := model.Message{
		ID:             uuid.New(),
		ConversationID: group.ID,
		SenderID:       admin.ID,
		Content:        "Welcome everybody to GoTalk! 🚀",
		Type:           model.MessageTypeText,
		Status:         model.MessageStatusSent,
	}
	db.Create(&msg)

	log.Println("✅ Created demo group: 'General Chat' with 3 members")
}
