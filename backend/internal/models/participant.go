package models

import "time"

type Participant struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SessionID  uint      `gorm:"not null;index" json:"session_id"`
	MemberID   uint      `gorm:"default:0" json:"member_id"`
	TelegramID int64     `gorm:"default:0" json:"telegram_id,omitempty"`
	Nickname   string    `gorm:"size:100;not null" json:"nickname"`
	TotalScore int       `gorm:"not null;default:0" json:"total_score"`
	JoinedAt   time.Time `json:"joined_at"`
}
