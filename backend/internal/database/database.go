package database

import (
	"fmt"
	"log"

	"quiz-game-backend/internal/config"
	"quiz-game-backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(cfg *config.Config) *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	log.Println("database connected")
	return db
}

func AutoMigrate(db *gorm.DB) {
	err := db.AutoMigrate(
		&models.Host{},
		&models.TelegramUser{},
		&models.Quiz{},
		&models.Category{},
		&models.Question{},
		&models.QuestionImage{},
		&models.Option{},
		&models.Session{},
		&models.Participant{},
		&models.Answer{},
	)
	if err != nil {
		log.Fatalf("failed to auto-migrate: %v", err)
	}
	log.Println("database migrated")
}
