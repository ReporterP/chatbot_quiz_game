package models

import "time"

type Room struct {
	ID        uint         `gorm:"primaryKey" json:"id"`
	HostID    uint         `gorm:"not null;index" json:"host_id"`
	Host      Host         `gorm:"foreignKey:HostID;constraint:OnDelete:CASCADE" json:"-"`
	Code      string       `gorm:"size:6;index" json:"code"`
	Mode      string       `gorm:"size:10;not null;default:'web'" json:"mode"`
	Status    string       `gorm:"size:20;not null;default:'active'" json:"status"`
	Members   []RoomMember `gorm:"foreignKey:RoomID" json:"members,omitempty"`
	Sessions  []Session    `gorm:"foreignKey:RoomID" json:"sessions,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

const (
	RoomModeWeb = "web"
	RoomModeBot = "bot"

	RoomStatusActive = "active"
	RoomStatusClosed = "closed"
)

type RoomMember struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	RoomID     uint      `gorm:"not null;index" json:"room_id"`
	Nickname   string    `gorm:"size:100;not null" json:"nickname"`
	TelegramID int64     `gorm:"default:0" json:"telegram_id,omitempty"`
	WebToken   string    `gorm:"size:64" json:"web_token,omitempty"`
	JoinedAt   time.Time `json:"joined_at"`
}
