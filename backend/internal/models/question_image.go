package models

type QuestionImage struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	QuestionID uint   `gorm:"not null;index" json:"question_id"`
	URL        string `gorm:"size:500;not null" json:"url"`
	Type       string `gorm:"size:10;not null;default:'image'" json:"type"`
	OrderNum   int    `gorm:"not null;default:0" json:"order_num"`
}
