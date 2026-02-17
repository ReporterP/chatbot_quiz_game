package services

import (
	"time"

	"quiz-game-backend/internal/models"

	"gorm.io/gorm"
)

type TelegramUserService struct {
	db *gorm.DB
}

func NewTelegramUserService(db *gorm.DB) *TelegramUserService {
	return &TelegramUserService{db: db}
}

func (s *TelegramUserService) GetOrCreate(telegramID int64, nickname string) (*models.TelegramUser, bool, error) {
	var user models.TelegramUser
	if err := s.db.Where("telegram_id = ?", telegramID).First(&user).Error; err == nil {
		return &user, false, nil
	}

	user = models.TelegramUser{
		TelegramID: telegramID,
		Nickname:   nickname,
	}
	if err := s.db.Create(&user).Error; err != nil {
		return nil, false, err
	}
	return &user, true, nil
}

func (s *TelegramUserService) Get(telegramID int64) (*models.TelegramUser, error) {
	var user models.TelegramUser
	if err := s.db.Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *TelegramUserService) UpdateNickname(telegramID int64, nickname string) (*models.TelegramUser, error) {
	var user models.TelegramUser
	if err := s.db.Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
		return nil, err
	}

	user.Nickname = nickname
	user.UpdatedAt = time.Now()
	if err := s.db.Save(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

type GameHistoryEntry struct {
	SessionID    uint      `json:"session_id"`
	QuizTitle    string    `json:"quiz_title"`
	TotalScore   int       `json:"total_score"`
	Position     int       `json:"position"`
	TotalPlayers int       `json:"total_players"`
	PlayedAt     time.Time `json:"played_at"`
}

func (s *TelegramUserService) GetHistory(telegramID int64) ([]GameHistoryEntry, error) {
	var participants []models.Participant
	s.db.Where("telegram_id = ?", telegramID).
		Order("joined_at DESC").
		Find(&participants)

	var entries []GameHistoryEntry
	for _, p := range participants {
		var session models.Session
		if err := s.db.Preload("Quiz").First(&session, p.SessionID).Error; err != nil {
			continue
		}
		if session.Status != models.SessionStatusFinished {
			continue
		}

		var allParticipants []models.Participant
		s.db.Where("session_id = ?", p.SessionID).Order("total_score DESC").Find(&allParticipants)

		position := 0
		for i, ap := range allParticipants {
			if ap.ID == p.ID {
				position = i + 1
				break
			}
		}

		entries = append(entries, GameHistoryEntry{
			SessionID:    p.SessionID,
			QuizTitle:    session.Quiz.Title,
			TotalScore:   p.TotalScore,
			Position:     position,
			TotalPlayers: len(allParticipants),
			PlayedAt:     p.JoinedAt,
		})
	}

	return entries, nil
}
