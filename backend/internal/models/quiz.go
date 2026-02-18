package models

import "time"

type Quiz struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	HostID     uint       `gorm:"not null;index" json:"host_id"`
	Host       Host       `gorm:"foreignKey:HostID;constraint:OnDelete:CASCADE" json:"-"`
	Title      string     `gorm:"size:255;not null" json:"title"`
	Mode       string     `gorm:"size:10;not null;default:'web'" json:"mode"`
	Categories []Category `gorm:"foreignKey:QuizID" json:"categories,omitempty"`
	Questions  []Question `gorm:"foreignKey:QuizID" json:"questions,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
