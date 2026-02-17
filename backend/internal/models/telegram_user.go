package models

import "time"

type TelegramUser struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TelegramID int64     `gorm:"not null;uniqueIndex:idx_tg_host" json:"telegram_id"`
	HostID     uint      `gorm:"not null;uniqueIndex:idx_tg_host" json:"host_id"`
	Nickname   string    `gorm:"size:100;not null" json:"nickname"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
