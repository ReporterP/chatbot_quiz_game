package telegram

import "fmt"

func MainMenuKeyboard() *ReplyKeyboardMarkup {
	return &ReplyKeyboardMarkup{
		Keyboard: [][]KeyboardButton{
			{{Text: "🎮 Войти в квиз"}},
			{{Text: "👤 Мой профиль"}, {Text: "📊 История игр"}},
		},
		ResizeKeyboard: true,
	}
}

func AnswerKeyboard(sessionID uint, options []QuestionOption, selectedID uint) *InlineKeyboardMarkup {
	var rows [][]InlineKeyboardButton
	for _, opt := range options {
		text := opt.Text
		if selectedID > 0 && opt.ID == selectedID {
			text = "✅ " + text
		}
		rows = append(rows, []InlineKeyboardButton{
			{Text: text, CallbackData: fmt.Sprintf("ans:%d:%d", sessionID, opt.ID)},
		})
	}
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}
