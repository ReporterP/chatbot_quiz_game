package models

type Category struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	QuizID    uint       `gorm:"not null;index" json:"quiz_id"`
	Title     string     `gorm:"size:255;not null" json:"title"`
	OrderNum  int        `gorm:"not null;default:0" json:"order_num"`
	Questions []Question `gorm:"foreignKey:CategoryID" json:"questions,omitempty"`
}
