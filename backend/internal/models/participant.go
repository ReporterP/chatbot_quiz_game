package models

import "time"

type Participant struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SessionID  uint      `gorm:"not null;uniqueIndex:idx_session_telegram" json:"session_id"`
	TelegramID int64     `gorm:"not null;uniqueIndex:idx_session_telegram" json:"telegram_id"`
	Nickname   string    `gorm:"size:100;not null" json:"nickname"`
	TotalScore int       `gorm:"not null;default:0" json:"total_score"`
	JoinedAt   time.Time `json:"joined_at"`
}
