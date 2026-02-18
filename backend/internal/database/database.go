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
	// Drop old unique index on telegram_id (was global, now per-host)
	db.Exec("DROP INDEX IF EXISTS idx_telegram_users_telegram_id")

	// Migrate host_id for existing telegram_users: add column with default, backfill, then let AutoMigrate finalize
	db.Exec(`DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'telegram_users')
		   AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'telegram_users' AND column_name = 'host_id')
		THEN
			ALTER TABLE telegram_users ADD COLUMN host_id bigint NOT NULL DEFAULT 0;
			-- Try to assign existing users to the first host
			UPDATE telegram_users SET host_id = COALESCE((SELECT id FROM hosts ORDER BY id LIMIT 1), 0) WHERE host_id = 0;
		END IF;
	END $$;`)

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
