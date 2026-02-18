package telegram

import "fmt"

func MainMenuKeyboard() *ReplyKeyboardMarkup {
	return &ReplyKeyboardMarkup{
		Keyboard: [][]KeyboardButton{
			{{Text: "🎮 Войти в квиз"}},
			{{Text: "👤 Мой профиль"}, {Text: "📊 История игр"}},
			{{Text: "🎯 Пульт ведущего"}},
		},
		ResizeKeyboard: true,
	}
}

func SessionMenuKeyboard() *ReplyKeyboardMarkup {
	return &ReplyKeyboardMarkup{
		Keyboard: [][]KeyboardButton{
			{{Text: "🔄 Переподключиться"}},
			{{Text: "🎮 Войти в квиз"}},
		},
		ResizeKeyboard: true,
	}
}

func HostControlKeyboard(sessionID uint, status string, current, total int) *InlineKeyboardMarkup {
	var rows [][]InlineKeyboardButton

	switch status {
	case "waiting":
		rows = append(rows, []InlineKeyboardButton{
			{Text: "▶️ Начать квиз", CallbackData: fmt.Sprintf("host:next:%d", sessionID)},
		})
	case "question":
		rows = append(rows, []InlineKeyboardButton{
			{Text: "👁 Показать ответ", CallbackData: fmt.Sprintf("host:reveal:%d", sessionID)},
		})
		rows = append(rows, []InlineKeyboardButton{
			{Text: "⏭ Завершить квиз", CallbackData: fmt.Sprintf("host:finish:%d", sessionID)},
		})
	case "revealed":
		if current < total {
			rows = append(rows, []InlineKeyboardButton{
				{Text: "➡️ Следующий вопрос", CallbackData: fmt.Sprintf("host:next:%d", sessionID)},
			})
		}
		rows = append(rows, []InlineKeyboardButton{
			{Text: "🏆 Завершить квиз", CallbackData: fmt.Sprintf("host:finish:%d", sessionID)},
		})
	}

	rows = append(rows, []InlineKeyboardButton{
		{Text: "🔄 Обновить", CallbackData: fmt.Sprintf("host:refresh:%d", sessionID)},
	})

	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

func HostSessionPickKeyboard(sessions []SessionPickItem) *InlineKeyboardMarkup {
	var rows [][]InlineKeyboardButton
	for _, s := range sessions {
		rows = append(rows, []InlineKeyboardButton{
			{Text: s.Label, CallbackData: fmt.Sprintf("host:pick:%d", s.SessionID)},
		})
	}
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

type SessionPickItem struct {
	SessionID uint
	Label     string
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
