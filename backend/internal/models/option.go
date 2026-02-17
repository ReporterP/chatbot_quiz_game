package models

type Option struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	QuestionID uint   `gorm:"not null;index" json:"question_id"`
	Text       string `gorm:"size:500;not null" json:"text"`
	IsCorrect  bool   `gorm:"not null;default:false" json:"is_correct"`
	Color      string `gorm:"size:7;default:''" json:"color"`
}
