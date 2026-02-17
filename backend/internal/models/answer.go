package models

import "time"

type Answer struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	SessionID     uint      `gorm:"not null;uniqueIndex:idx_answer_unique" json:"session_id"`
	ParticipantID uint      `gorm:"not null;uniqueIndex:idx_answer_unique" json:"participant_id"`
	QuestionID    uint      `gorm:"not null;uniqueIndex:idx_answer_unique;index:idx_answer_order" json:"question_id"`
	OptionID      uint      `gorm:"not null" json:"option_id"`
	IsCorrect     bool      `gorm:"not null" json:"is_correct"`
	Score         int       `gorm:"not null;default:0" json:"score"`
	AnsweredAt    time.Time `gorm:"index:idx_answer_order" json:"answered_at"`
}
