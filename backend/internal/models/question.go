package models

type Question struct {
	ID            uint            `gorm:"primaryKey" json:"id"`
	QuizID        uint            `gorm:"not null;index" json:"quiz_id"`
	CategoryID    *uint           `gorm:"index" json:"category_id,omitempty"`
	Type          string          `gorm:"size:20;not null;default:'single_choice'" json:"type"`
	Text          string          `gorm:"type:text;not null" json:"text"`
	OrderNum      int             `gorm:"not null" json:"order_num"`
	CorrectNumber *float64        `json:"correct_number,omitempty"`
	Tolerance     *float64        `json:"tolerance,omitempty"`
	Options       []Option        `gorm:"foreignKey:QuestionID" json:"options,omitempty"`
	Images        []QuestionImage `gorm:"foreignKey:QuestionID" json:"images,omitempty"`
}

const (
	QuestionTypeSingleChoice   = "single_choice"
	QuestionTypeMultipleChoice = "multiple_choice"
	QuestionTypeOrdering       = "ordering"
	QuestionTypeMatching       = "matching"
	QuestionTypeNumeric        = "numeric"
)
