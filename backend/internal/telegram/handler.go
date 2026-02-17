package telegram

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"quiz-game-backend/internal/services"
	"quiz-game-backend/internal/ws"

	"github.com/gin-gonic/gin"
)

type UpdateHandler struct {
	client     *Client
	state      *StateManager
	tracker    *SessionTracker
	sessionSvc *services.SessionService
	tgUserSvc  *services.TelegramUserService
	hub        *ws.Hub
	hostID     uint
}

func NewUpdateHandler(
	client *Client,
	state *StateManager,
	tracker *SessionTracker,
	sessionSvc *services.SessionService,
	tgUserSvc *services.TelegramUserService,
	hub *ws.Hub,
	hostID uint,
) *UpdateHandler {
	return &UpdateHandler{
		client:     client,
		state:      state,
		tracker:    tracker,
		sessionSvc: sessionSvc,
		tgUserSvc:  tgUserSvc,
		hub:        hub,
		hostID:     hostID,
	}
}

func (h *UpdateHandler) Handle(upd Update) {
	if upd.CallbackQuery != nil {
		h.handleCallback(upd.CallbackQuery)
		return
	}
	if upd.Message != nil {
		h.handleMessage(upd.Message)
	}
}

func (h *UpdateHandler) handleMessage(msg *Message) {
	if msg.From == nil {
		return
	}
	userID := msg.From.ID
	chatID := msg.Chat.ID
	text := strings.TrimSpace(msg.Text)

	if isCommand(msg, "start") {
		h.cmdStart(msg, userID, chatID, text)
		return
	}

	if strings.HasPrefix(text, "/nickname") {
		h.cmdNickname(userID, chatID, text)
		return
	}

	switch text {
	case "🎮 Войти в квиз":
		h.state.Set(userID, &UserState{State: StateEnterCode})
		h.client.SendMessage(chatID, "Введите 6-значный код сессии:", "", nil)
		return
	case "👤 Мой профиль":
		h.cmdProfile(userID, chatID)
		return
	case "📊 История игр":
		h.cmdHistory(userID, chatID)
		return
	}

	us := h.state.Get(userID)
	switch us.State {
	case StateEnterCode:
		h.onCode(userID, chatID, text, msg.From.FirstName)
	case StateEnterNickname:
		h.onNickname(userID, chatID, text)
	default:
		h.client.SendMessage(chatID, "Используйте /start или кнопки меню.", "", MainMenuKeyboard())
	}
}

func (h *UpdateHandler) cmdStart(msg *Message, userID, chatID int64, text string) {
	h.state.Clear(userID)

	firstName := "Player"
	if msg.From != nil && msg.From.FirstName != "" {
		firstName = msg.From.FirstName
	}

	user, created, err := h.tgUserSvc.GetOrCreate(userID, h.hostID, firstName)
	var nickname string
	if err == nil {
		nickname = user.Nickname
	}

	args := extractStartArgs(text)

	if args != "" {
		code := strings.TrimSpace(args)
		if nickname != "" && !created {
			h.doJoin(userID, chatID, code, nickname)
		} else {
			h.state.Set(userID, &UserState{State: StateEnterNickname, Code: code})
			h.client.SendMessage(chatID,
				fmt.Sprintf("👋 Добро пожаловать в Quiz Game!\n\nКод сессии: <b>%s</b>\nВведите ваш никнейм:", code),
				"HTML", nil)
		}
		return
	}

	if nickname != "" && !created {
		h.client.SendMessage(chatID,
			fmt.Sprintf("👋 Привет, <b>%s</b>!\n\nВыберите действие:", nickname),
			"HTML", MainMenuKeyboard())
	} else {
		h.state.Set(userID, &UserState{State: StateEnterNickname})
		h.client.SendMessage(chatID,
			"👋 Добро пожаловать в Quiz Game!\n\nВведите ваш никнейм:", "", nil)
	}
}

func (h *UpdateHandler) onCode(userID, chatID int64, code, firstName string) {
	if len(code) != 6 || !isDigits(code) {
		h.client.SendMessage(chatID, "❌ Код должен состоять из 6 цифр. Попробуйте ещё раз:", "", nil)
		return
	}

	user, created, err := h.tgUserSvc.GetOrCreate(userID, h.hostID, firstName)
	var nickname string
	if err == nil {
		nickname = user.Nickname
	}

	if nickname != "" && !created {
		h.doJoin(userID, chatID, code, nickname)
	} else {
		h.state.Set(userID, &UserState{State: StateEnterNickname, Code: code})
		h.client.SendMessage(chatID,
			fmt.Sprintf("✅ Код принят: <b>%s</b>\n\nВведите ваш никнейм:", code),
			"HTML", nil)
	}
}

func (h *UpdateHandler) onNickname(userID, chatID int64, nickname string) {
	if len(nickname) < 1 || len(nickname) > 100 {
		h.client.SendMessage(chatID, "❌ Никнейм должен быть от 1 до 100 символов. Попробуйте ещё раз:", "", nil)
		return
	}

	h.tgUserSvc.UpdateNickname(userID, h.hostID, nickname)

	us := h.state.Get(userID)
	code := us.Code

	if code == "" {
		h.state.Clear(userID)
		h.client.SendMessage(chatID,
			fmt.Sprintf("✅ Никнейм установлен: <b>%s</b>\n\nВыберите действие:", nickname),
			"HTML", MainMenuKeyboard())
		return
	}

	h.doJoin(userID, chatID, code, nickname)
}

func (h *UpdateHandler) doJoin(userID, chatID int64, code, nickname string) {
	result, err := h.sessionSvc.JoinSession(code, userID, nickname)
	if err != nil {
		h.client.SendMessage(chatID,
			fmt.Sprintf("❌ Ошибка: %s\n\nПопробуйте /start заново.", err.Error()),
			"", MainMenuKeyboard())
		h.state.Clear(userID)
		return
	}

	h.state.Set(userID, &UserState{
		State:     StateInSession,
		SessionID: result.SessionID,
		Nickname:  nickname,
	})

	msgID, _ := h.client.SendMessage(chatID,
		fmt.Sprintf("🎮 Вы подключились к квизу!\n\nНикнейм: <b>%s</b>\nОжидайте начала игры...", nickname),
		"HTML", nil)

	h.tracker.AddParticipant(result.SessionID, userID, chatID, msgID)

	if h.hub != nil {
		h.hub.Broadcast(result.SessionID, ws.WSMessage{
			Type: "participant_joined",
			Data: result.Participant,
		})
	}
}

func (h *UpdateHandler) cmdProfile(userID, chatID int64) {
	user, _, err := h.tgUserSvc.GetOrCreate(userID, h.hostID, "Player")
	if err != nil {
		h.client.SendMessage(chatID, "Ошибка загрузки профиля", "", nil)
		return
	}
	h.client.SendMessage(chatID,
		fmt.Sprintf("👤 <b>Ваш профиль</b>\n\nНикнейм: <b>%s</b>\n\nЧтобы изменить ник, отправьте:\n/nickname Новый_ник", user.Nickname),
		"HTML", nil)
}

func (h *UpdateHandler) cmdHistory(userID, chatID int64) {
	entries, err := h.tgUserSvc.GetHistory(userID, h.hostID)
	if err != nil || len(entries) == 0 {
		h.client.SendMessage(chatID, "📊 У вас пока нет завершённых игр.", "", nil)
		return
	}

	medals := map[int]string{1: "🥇", 2: "🥈", 3: "🥉"}
	lines := []string{"📊 <b>Ваша история игр:</b>\n"}
	limit := 20
	if len(entries) < limit {
		limit = len(entries)
	}
	for _, e := range entries[:limit] {
		medal, ok := medals[e.Position]
		if !ok {
			medal = fmt.Sprintf("%d.", e.Position)
		}
		lines = append(lines, fmt.Sprintf("%s <b>%s</b>\n   Очки: %d | Место: %d/%d",
			medal, e.QuizTitle, e.TotalScore, e.Position, e.TotalPlayers))
	}

	h.client.SendMessage(chatID, strings.Join(lines, "\n"), "HTML", nil)
}

func (h *UpdateHandler) cmdNickname(userID, chatID int64, text string) {
	parts := strings.SplitN(text, " ", 2)
	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		h.client.SendMessage(chatID, "Использование: /nickname Ваш_новый_ник", "", nil)
		return
	}

	newNick := strings.TrimSpace(parts[1])
	if len(newNick) > 100 {
		h.client.SendMessage(chatID, "Никнейм слишком длинный (макс 100 символов)", "", nil)
		return
	}

	user, err := h.tgUserSvc.UpdateNickname(userID, h.hostID, newNick)
	if err != nil {
		h.client.SendMessage(chatID, fmt.Sprintf("Ошибка: %s", err.Error()), "", nil)
		return
	}

	h.client.SendMessage(chatID,
		fmt.Sprintf("✅ Никнейм изменён на: <b>%s</b>", user.Nickname),
		"HTML", MainMenuKeyboard())
}

func (h *UpdateHandler) handleCallback(cb *CallbackQuery) {
	if !strings.HasPrefix(cb.Data, "ans:") {
		h.client.AnswerCallbackQuery(cb.ID, "Неверные данные", true)
		return
	}

	userID := cb.From.ID
	us := h.state.Get(userID)
	if us.State != StateInSession {
		h.client.AnswerCallbackQuery(cb.ID, "Вы не в активной сессии", true)
		return
	}

	parts := strings.Split(cb.Data, ":")
	if len(parts) != 3 {
		h.client.AnswerCallbackQuery(cb.ID, "Неверные данные", true)
		return
	}

	sessionID, _ := strconv.ParseUint(parts[1], 10, 64)
	optionID, _ := strconv.ParseUint(parts[2], 10, 64)

	err := h.sessionSvc.SubmitAnswer(uint(sessionID), userID, uint(optionID))
	if err != nil {
		errText := err.Error()
		if strings.Contains(errText, "not accepting") {
			h.client.AnswerCallbackQuery(cb.ID, "Время для ответа вышло", true)
		} else {
			h.client.AnswerCallbackQuery(cb.ID, "Ошибка: "+errText, true)
		}
		return
	}

	if us.QuestionData != nil && cb.Message != nil {
		kb := AnswerKeyboard(uint(sessionID), us.QuestionData.Options, uint(optionID))
		text := fmt.Sprintf("❓ <b>Вопрос %d из %d</b>\n\n%s\n\n✅ <b>Ваш ответ принят</b>",
			us.CurrentQNum, us.TotalQuestions, us.QuestionData.Text)

		if err := h.client.EditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, text, "HTML", kb); err != nil {
			log.Printf("edit answer msg: %v", err)
		}
	}

	h.state.UpdateField(userID, func(s *UserState) {
		s.SelectedOptionID = uint(optionID)
	})

	if h.hub != nil {
		h.hub.Broadcast(uint(sessionID), ws.WSMessage{
			Type: "answer_received",
			Data: gin.H{"session_id": sessionID},
		})
	}

	h.client.AnswerCallbackQuery(cb.ID, "✅ Ответ принят!", false)
}

func isCommand(msg *Message, cmd string) bool {
	if msg.Entities == nil {
		return false
	}
	for _, e := range msg.Entities {
		if e.Type == "bot_command" && e.Offset == 0 {
			cmdText := msg.Text[e.Offset:e.Offset+e.Length]
			cmdText = strings.Split(cmdText, "@")[0]
			return cmdText == "/"+cmd
		}
	}
	return false
}

func extractStartArgs(text string) string {
	parts := strings.SplitN(text, " ", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
