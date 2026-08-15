package database

import (
	"fmt"
	"log"
	"time"

	"smart-live-chats/internal/config"
	"smart-live-chats/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Connect establishes a connection to the database
func Connect(cfg *config.Config) error {
	var err error

	DB, err = gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL DB
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("Database connected successfully")
	return nil
}

// Migrate runs database migrations
func Migrate() error {
	log.Println("Running database migrations...")

	DB.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm")

	err := DB.AutoMigrate(
		&models.User{},
		&models.ChatSession{},
		&models.ChatMessage{},
		&models.WhatsAppDevice{},
		&models.KnowledgeBase{},
		&models.QAItem{},
		&models.Subscription{},
		&models.Tag{},
		&models.SessionTag{},
		&models.PlatformSetting{},
		&models.Product{},
		&models.ProductImage{},
		&models.Discount{},
		&models.CustomerProfile{},
		&models.Lead{},
	)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Drop the deprecated free-text product catalog blob. Structured products
	// (products / product_images / discounts) are now the single source of truth,
	// surfaced to the AI via tools. AutoMigrate never drops columns, so do it
	// explicitly and idempotently.
	if err := DB.Exec("ALTER TABLE knowledge_bases DROP COLUMN IF EXISTS product_info").Error; err != nil {
		return fmt.Errorf("failed to drop knowledge_bases.product_info: %w", err)
	}

	log.Println("Database migrations completed")
	return nil
}

// Seed adds initial data to the database
func Seed() error {
	log.Println("Seeding database with sample data...")

	// Check if we already have data
	var count int64
	DB.Model(&models.ChatSession{}).Count(&count)
	if count > 0 {
		log.Println("Database already has data, skipping seed")
		return nil
	}

	// Create sample chat sessions
	now := time.Now()
	sessions := []models.ChatSession{
		{
			CustomerName:   "LingoACE 刘杰",
			CustomerAvatar: "",
			CustomerPhone:  "1234567890",
			Platform:       "whatsapp",
			UnreadCount:    1,
			LastMessage:    "哈喽",
			LastMessageAt:  &now,
			Status:         "active",
		},
		{
			CustomerName:   "YouTube HR2",
			CustomerAvatar: "",
			CustomerPhone:  "7434243299529",
			Platform:       "whatsapp",
			UnreadCount:    8,
			LastMessage:    "Hello, I have a question about your service",
			LastMessageAt:  &now,
			Status:         "active",
		},
		{
			CustomerName:   "1E - Mrs Estrada",
			CustomerAvatar: "",
			CustomerPhone:  "2075843290490",
			Platform:       "whatsapp",
			UnreadCount:    0,
			LastMessage:    "Cool! Thank you so much for the info ❤️",
			LastMessageAt:  &now,
			Status:         "active",
		},
	}

	for _, session := range sessions {
		if err := DB.Create(&session).Error; err != nil {
			return fmt.Errorf("failed to create sample session: %w", err)
		}
	}

	// Create sample messages for the third session
	messages := []models.ChatMessage{
		{
			SessionID:   3,
			SenderType:  "customer",
			Content:     "the mini fete has games like maybe sporty things so i think by wearing sports it just more comfy for the kiddos",
			MessageType: "text",
			IsRead:      true,
			CreatedAt:   now.Add(-2 * time.Hour),
		},
		{
			SessionID:   3,
			SenderType:  "customer",
			Content:     "",
			MessageType: "image",
			MediaURL:    "https://via.placeholder.com/200x150",
			IsRead:      true,
			CreatedAt:   now.Add(-1 * time.Hour),
		},
		{
			SessionID:   3,
			SenderType:  "customer",
			Content:     "Found on school bytes 🙂",
			MessageType: "text",
			IsRead:      true,
			CreatedAt:   now.Add(-50 * time.Minute),
		},
		{
			SessionID:   3,
			SenderType:  "agent",
			Content:     "Cool! Thank you so much for the info ❤️",
			MessageType: "text",
			IsRead:      true,
			CreatedAt:   now.Add(-30 * time.Minute),
		},
	}

	for _, msg := range messages {
		if err := DB.Create(&msg).Error; err != nil {
			return fmt.Errorf("failed to create sample message: %w", err)
		}
	}

	log.Println("Database seeding completed")
	return nil
}

// Close closes the database connection
func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

