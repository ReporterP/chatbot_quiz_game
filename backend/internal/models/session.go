package models

import "time"

type Session struct {
	ID              uint          `gorm:"primaryKey" json:"id"`
	RoomID          uint          `gorm:"not null;index" json:"room_id"`
	QuizID          uint          `gorm:"not null" json:"quiz_id"`
	Quiz            Quiz          `gorm:"foreignKey:QuizID" json:"quiz,omitempty"`
	HostID          uint          `gorm:"not null;index" json:"host_id"`
	Code            string        `gorm:"size:6;index" json:"code"`
	Status          string        `gorm:"size:20;not null;default:'waiting'" json:"status"`
	CurrentQuestion int           `gorm:"not null;default:0" json:"current_question"`
	Participants    []Participant `gorm:"foreignKey:SessionID" json:"participants,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
}

const (
	SessionStatusWaiting  = "waiting"
	SessionStatusQuestion = "question"
	SessionStatusRevealed = "revealed"
	SessionStatusFinished = "finished"
)
