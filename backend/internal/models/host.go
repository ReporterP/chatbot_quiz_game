package models

import "time"

type Host struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Username       string    `gorm:"size:100;uniqueIndex;not null" json:"username"`
	PasswordHash   string    `gorm:"size:255;not null" json:"-"`
	BotToken       string    `gorm:"size:255" json:"bot_token,omitempty"`
	BotLink        string    `gorm:"size:255" json:"bot_link,omitempty"`
	RemotePassword string    `gorm:"size:255" json:"-"`
	CreatedAt      time.Time `json:"created_at"`
}
