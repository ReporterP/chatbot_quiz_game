package models

type Question struct {
	ID         uint            `gorm:"primaryKey" json:"id"`
	QuizID     uint            `gorm:"not null;index" json:"quiz_id"`
	CategoryID *uint           `gorm:"index" json:"category_id,omitempty"`
	Text       string          `gorm:"type:text;not null" json:"text"`
	OrderNum   int             `gorm:"not null" json:"order_num"`
	Options    []Option        `gorm:"foreignKey:QuestionID" json:"options,omitempty"`
	Images     []QuestionImage `gorm:"foreignKey:QuestionID" json:"images,omitempty"`
}
