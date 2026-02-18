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
	db.Exec("DROP INDEX IF EXISTS idx_telegram_users_telegram_id")

	db.Exec(`DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'telegram_users')
		   AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'telegram_users' AND column_name = 'host_id')
		THEN
			ALTER TABLE telegram_users ADD COLUMN host_id bigint NOT NULL DEFAULT 0;
			UPDATE telegram_users SET host_id = COALESCE((SELECT id FROM hosts ORDER BY id LIMIT 1), 0) WHERE host_id = 0;
		END IF;
	END $$;`)

	// Add room_id to sessions if missing (backward compat: set to 0)
	db.Exec(`DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'sessions')
		   AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sessions' AND column_name = 'room_id')
		THEN
			ALTER TABLE sessions ADD COLUMN room_id bigint DEFAULT 0;
		END IF;
	END $$;`)

	// Add member_id to participants if missing
	db.Exec(`DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'participants')
		   AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'participants' AND column_name = 'member_id')
		THEN
			ALTER TABLE participants ADD COLUMN member_id bigint DEFAULT 0;
		END IF;
	END $$;`)

	// Drop old unique index on participants that requires telegram_id
	db.Exec("DROP INDEX IF EXISTS idx_session_telegram")

	// Relax NOT NULL on room_id/member_id if it was set by a previous migration
	db.Exec("ALTER TABLE sessions ALTER COLUMN room_id DROP NOT NULL")
	db.Exec("ALTER TABLE participants ALTER COLUMN member_id DROP NOT NULL")

	// Add mode to quizzes if missing
	db.Exec(`DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'quizzes')
		   AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'quizzes' AND column_name = 'mode')
		THEN
			ALTER TABLE quizzes ADD COLUMN mode varchar(10) NOT NULL DEFAULT 'web';
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
		&models.Room{},
		&models.RoomMember{},
		&models.Session{},
		&models.Participant{},
		&models.Answer{},
	)
	if err != nil {
		log.Fatalf("failed to auto-migrate: %v", err)
	}
	log.Println("database migrated")
}
