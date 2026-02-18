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
		{Text: "🔙 К комнате", CallbackData: fmt.Sprintf("host:backroom:%d", sessionID)},
	})

	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

type RoomPickItem struct {
	RoomID uint
	Label  string
}

func HostRoomPickKeyboard(rooms []RoomPickItem) *InlineKeyboardMarkup {
	var rows [][]InlineKeyboardButton
	for _, r := range rooms {
		rows = append(rows, []InlineKeyboardButton{
			{Text: r.Label, CallbackData: fmt.Sprintf("host:room:%d", r.RoomID)},
		})
	}
	rows = append(rows, []InlineKeyboardButton{
		{Text: "➕ Новая комната", CallbackData: "host:newroom:0"},
	})
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

func HostRoomControlKeyboard(roomID uint, hasSession bool) *InlineKeyboardMarkup {
	var rows [][]InlineKeyboardButton
	if !hasSession {
		rows = append(rows, []InlineKeyboardButton{
			{Text: "📋 Выбрать квиз", CallbackData: fmt.Sprintf("host:pickquiz:%d:0", roomID)},
		})
	}
	rows = append(rows, []InlineKeyboardButton{
		{Text: "🔄 Обновить", CallbackData: fmt.Sprintf("host:roomrefresh:%d", roomID)},
	})
	rows = append(rows, []InlineKeyboardButton{
		{Text: "❌ Закрыть комнату", CallbackData: fmt.Sprintf("host:closeroom:%d", roomID)},
		{Text: "🔙 К комнатам", CallbackData: "host:rooms:0"},
	})
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

type QuizPickItem struct {
	QuizID uint
	Label  string
}

func HostQuizPickKeyboard(roomID uint, quizzes []QuizPickItem, page, totalPages int) *InlineKeyboardMarkup {
	var rows [][]InlineKeyboardButton
	for _, q := range quizzes {
		rows = append(rows, []InlineKeyboardButton{
			{Text: q.Label, CallbackData: fmt.Sprintf("host:startquiz:%d:%d", roomID, q.QuizID)},
		})
	}
	if totalPages > 1 {
		var navRow []InlineKeyboardButton
		if page > 0 {
			navRow = append(navRow, InlineKeyboardButton{
				Text: "◀️", CallbackData: fmt.Sprintf("host:pickquiz:%d:%d", roomID, page-1),
			})
		}
		navRow = append(navRow, InlineKeyboardButton{
			Text: fmt.Sprintf("%d/%d", page+1, totalPages),
			CallbackData: "host:noop:0",
		})
		if page < totalPages-1 {
			navRow = append(navRow, InlineKeyboardButton{
				Text: "▶️", CallbackData: fmt.Sprintf("host:pickquiz:%d:%d", roomID, page+1),
			})
		}
		rows = append(rows, navRow)
	}
	rows = append(rows, []InlineKeyboardButton{
		{Text: "🔙 К комнате", CallbackData: fmt.Sprintf("host:room:%d", roomID)},
	})
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

type SessionPickItem struct {
	SessionID uint
	Label     string
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
