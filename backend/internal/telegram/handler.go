package telegram

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"quiz-game-backend/internal/models"
	"quiz-game-backend/internal/services"
	"quiz-game-backend/internal/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UpdateHandler struct {
	client     *Client
	state      *StateManager
	tracker    *SessionTracker
	sessionSvc *services.SessionService
	tgUserSvc  *services.TelegramUserService
	hub        *ws.Hub
	db         *gorm.DB
	hostID     uint
}

func NewUpdateHandler(
	client *Client,
	state *StateManager,
	tracker *SessionTracker,
	sessionSvc *services.SessionService,
	tgUserSvc *services.TelegramUserService,
	hub *ws.Hub,
	db *gorm.DB,
	hostID uint,
) *UpdateHandler {
	return &UpdateHandler{
		client:     client,
		state:      state,
		tracker:    tracker,
		sessionSvc: sessionSvc,
		tgUserSvc:  tgUserSvc,
		hub:        hub,
		db:         db,
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

	if strings.HasPrefix(text, "/rejoin") {
		h.cmdRejoin(userID, chatID)
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
	case "🔄 Переподключиться":
		h.cmdRejoin(userID, chatID)
		return
	case "🎯 Пульт ведущего":
		h.startHostAuth(userID, chatID)
		return
	}

	us := h.state.Get(userID)
	switch us.State {
	case StateEnterCode:
		h.onCode(userID, chatID, text, msg.From.FirstName)
	case StateEnterNickname:
		h.onNickname(userID, chatID, text)
	case StateInSession:
		h.tryRecoverSession(userID, chatID, us)
	case StateHostPassword:
		h.onHostPassword(userID, chatID, text)
	case StateHostRemote:
		h.client.SendMessage(chatID, "🎯 Вы в режиме пульта. Используйте кнопки в сообщении выше.\n\nДля выхода нажмите /start", "HTML", nil)
	default:
		h.client.SendMessage(chatID, "Используйте /start или кнопки меню.", "", MainMenuKeyboard())
	}
}

// ─── /start ───

func (h *UpdateHandler) cmdStart(msg *Message, userID, chatID int64, text string) {
	firstName := "Player"
	if msg.From != nil && msg.From.FirstName != "" {
		firstName = msg.From.FirstName
	}

	args := extractStartArgs(text)

	us := h.state.Get(userID)
	if us.State == StateInSession && us.SessionID > 0 && args == "" {
		sessState, err := h.sessionSvc.GetSession(us.SessionID)
		if err == nil && sessState.Status != "finished" {
			h.client.SendMessage(chatID,
				"🎮 Вы сейчас в активной сессии.\n\nНажмите <b>🔄 Переподключиться</b> чтобы вернуться в игру, или введите новый код.",
				"HTML", SessionMenuKeyboard())
			return
		}
	}

	h.state.Clear(userID)

	user, created, err := h.tgUserSvc.GetOrCreate(userID, h.hostID, firstName)
	var nickname string
	if err == nil {
		nickname = user.Nickname
	}

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

// ─── Participant join flow ───

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
		Code:      code,
		Nickname:  nickname,
	})

	var statusText string
	if result.IsRejoin {
		statusText = fmt.Sprintf("🔄 Вы переподключились к квизу!\n\nНикнейм: <b>%s</b>", nickname)
	} else {
		statusText = fmt.Sprintf("🎮 Вы подключились к квизу!\n\nНикнейм: <b>%s</b>\nОжидайте начала игры...", nickname)
	}

	msgID, _ := h.client.SendMessage(chatID, statusText, "HTML", nil)

	h.tracker.AddParticipant(result.SessionID, userID, chatID, msgID)

	sessState, err := h.sessionSvc.GetSession(result.SessionID)
	if err == nil && sessState.Status != "waiting" {
		go h.tracker.SyncParticipant(result.SessionID, userID)
	}

	if h.hub != nil && !result.IsRejoin {
		h.hub.Broadcast(result.SessionID, ws.WSMessage{
			Type: "participant_joined",
			Data: result.Participant,
		})
	}
}

func (h *UpdateHandler) tryRecoverSession(userID, chatID int64, us *UserState) {
	if us.SessionID == 0 {
		h.state.Clear(userID)
		h.client.SendMessage(chatID, "Сессия не найдена. Используйте /start", "", MainMenuKeyboard())
		return
	}

	sessState, err := h.sessionSvc.GetSession(us.SessionID)
	if err != nil || sessState.Status == "finished" {
		h.state.Clear(userID)
		h.client.SendMessage(chatID, "Сессия завершена. Нажмите /start для новой игры.", "", MainMenuKeyboard())
		return
	}

	msgID, _ := h.client.SendMessage(chatID,
		"🔄 Переподключение к квизу...", "HTML", nil)

	h.tracker.AddParticipant(us.SessionID, userID, chatID, msgID)
	go h.tracker.SyncParticipant(us.SessionID, userID)
}

func (h *UpdateHandler) cmdRejoin(userID, chatID int64) {
	us := h.state.Get(userID)
	if us.State != StateInSession || us.SessionID == 0 {
		h.client.SendMessage(chatID, "Вы не подключены к сессии. Нажмите /start", "", MainMenuKeyboard())
		return
	}

	sessState, err := h.sessionSvc.GetSession(us.SessionID)
	if err != nil || sessState.Status == "finished" {
		h.state.Clear(userID)
		h.client.SendMessage(chatID, "Сессия завершена. Нажмите /start для новой игры.", "", MainMenuKeyboard())
		return
	}

	msgID, _ := h.client.SendMessage(chatID,
		"🔄 Переподключение к квизу...", "HTML", nil)

	h.tracker.AddParticipant(us.SessionID, userID, chatID, msgID)
	go h.tracker.SyncParticipant(us.SessionID, userID)
}

// ─── Profile / History / Nickname ───

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

// ─── Host Remote Control ───

func (h *UpdateHandler) startHostAuth(userID, chatID int64) {
	var host models.Host
	if err := h.db.First(&host, h.hostID).Error; err != nil || host.RemotePassword == "" {
		h.client.SendMessage(chatID,
			"❌ Пульт ведущего не настроен.\n\nВладелец бота должен задать пароль для пульта в настройках на сайте.",
			"HTML", MainMenuKeyboard())
		return
	}

	h.state.Set(userID, &UserState{State: StateHostPassword})
	h.client.SendMessage(chatID, "🔐 Введите пароль пульта ведущего:", "", nil)
}

func (h *UpdateHandler) onHostPassword(userID, chatID int64, password string) {
	var host models.Host
	if err := h.db.First(&host, h.hostID).Error; err != nil {
		h.client.SendMessage(chatID, "Ошибка. Попробуйте /start", "", MainMenuKeyboard())
		h.state.Clear(userID)
		return
	}

	if strings.TrimSpace(password) != host.RemotePassword {
		h.client.SendMessage(chatID, "❌ Неверный пароль. Попробуйте ещё раз:", "", nil)
		return
	}

	h.state.Set(userID, &UserState{State: StateHostRemote})

	sessions, err := h.sessionSvc.GetActiveSessions(h.hostID)
	if err != nil || len(sessions) == 0 {
		h.client.SendMessage(chatID,
			"🎯 <b>Пульт ведущего</b>\n\n📋 Нет активных сессий.\nСоздайте сессию на сайте, затем нажмите /start → 🎯 Пульт ведущего",
			"HTML", MainMenuKeyboard())
		h.state.Clear(userID)
		return
	}

	var items []SessionPickItem
	statusLabels := map[string]string{
		"waiting":  "⏳ ожидание",
		"question": "❓ вопрос",
		"revealed": "👁 ответ показан",
	}
	for _, s := range sessions {
		sl := statusLabels[s.Status]
		if sl == "" {
			sl = s.Status
		}
		items = append(items, SessionPickItem{
			SessionID: s.ID,
			Label:     fmt.Sprintf("%s [%s] %s 👥%d", s.QuizTitle, s.Code, sl, s.ParticipantCount),
		})
	}

	h.client.SendMessage(chatID,
		"🎯 <b>Пульт ведущего</b>\n\n✅ Авторизация успешна!\nВыберите сессию:",
		"HTML", HostSessionPickKeyboard(items))
}

func (h *UpdateHandler) handleHostPick(cb *CallbackQuery, sessionID uint) {
	userID := cb.From.ID
	chatID := cb.Message.Chat.ID

	sessState, err := h.sessionSvc.GetSession(sessionID)
	if err != nil {
		h.client.AnswerCallbackQuery(cb.ID, "Сессия не найдена", true)
		return
	}

	if sessState.HostID != h.hostID {
		h.client.AnswerCallbackQuery(cb.ID, "Нет доступа к этой сессии", true)
		return
	}

	h.state.UpdateField(userID, func(s *UserState) {
		s.State = StateHostRemote
		s.SessionID = sessionID
	})

	text := h.tracker.buildHostControlText(sessState)
	kb := HostControlKeyboard(sessionID, sessState.Status, sessState.CurrentQuestion, sessState.TotalQuestions)

	msgID, _ := h.client.SendMessage(chatID, text, "HTML", kb)

	h.tracker.SetHostRemote(sessionID, chatID, msgID)

	h.client.AnswerCallbackQuery(cb.ID, "", false)
}

func (h *UpdateHandler) handleHostAction(cb *CallbackQuery, action string, sessionID uint) {
	chatID := cb.Message.Chat.ID

	broadcastToAll := func(msgType string, data interface{}, roomID uint) {
		if h.hub != nil {
			h.hub.Broadcast(sessionID, ws.WSMessage{Type: msgType, Data: data})
			if roomID > 0 {
				h.hub.BroadcastToRoom(roomID, ws.WSMessage{Type: msgType, Data: data})
			}
		}
	}

	sessForRoom, _ := h.sessionSvc.GetSession(sessionID)
	var roomID uint
	if sessForRoom != nil {
		roomID = sessForRoom.RoomID
	}

	switch action {
	case "reveal":
		state, err := h.sessionSvc.RevealAnswer(sessionID, h.hostID)
		if err != nil {
			h.client.AnswerCallbackQuery(cb.ID, "Ошибка: "+err.Error(), true)
			return
		}
		broadcastToAll("revealed", state, roomID)
		h.client.AnswerCallbackQuery(cb.ID, "👁 Ответ показан", false)

	case "next":
		state, err := h.sessionSvc.NextQuestion(sessionID, h.hostID)
		if err != nil {
			h.client.AnswerCallbackQuery(cb.ID, "Ошибка: "+err.Error(), true)
			return
		}
		msgType := "question"
		if state.Status == "finished" {
			msgType = "finished"
		}
		broadcastToAll(msgType, state, roomID)
		h.client.AnswerCallbackQuery(cb.ID, "➡️ Далее", false)

	case "finish":
		state, err := h.sessionSvc.ForceFinish(sessionID, h.hostID)
		if err != nil {
			h.client.AnswerCallbackQuery(cb.ID, "Ошибка: "+err.Error(), true)
			return
		}
		broadcastToAll("finished", state, roomID)
		h.client.AnswerCallbackQuery(cb.ID, "🏆 Квиз завершён", false)

	case "refresh":
		h.client.AnswerCallbackQuery(cb.ID, "🔄 Обновлено", false)
	}

	sessState, err := h.sessionSvc.GetSession(sessionID)
	if err != nil {
		return
	}

	text := h.tracker.buildHostControlText(sessState)
	kb := HostControlKeyboard(sessionID, sessState.Status, sessState.CurrentQuestion, sessState.TotalQuestions)

	if cb.Message != nil && cb.Message.MessageID > 0 {
		if err := h.client.EditMessageText(chatID, cb.Message.MessageID, text, "HTML", kb); err != nil {
			msgID, _ := h.client.SendMessage(chatID, text, "HTML", kb)
			if msgID > 0 {
				h.tracker.SetHostRemote(sessionID, chatID, msgID)
			}
		}
	}

	if sessState.Status == "finished" {
		h.state.Clear(cb.From.ID)
	}
}

// ─── Callback router ───

func (h *UpdateHandler) handleCallback(cb *CallbackQuery) {
	if strings.HasPrefix(cb.Data, "host:") {
		h.routeHostCallback(cb)
		return
	}

	if !strings.HasPrefix(cb.Data, "ans:") {
		h.client.AnswerCallbackQuery(cb.ID, "Неверные данные", true)
		return
	}

	h.handleAnswerCallback(cb)
}

func (h *UpdateHandler) routeHostCallback(cb *CallbackQuery) {
	// format: host:<action>:<sessionID>
	parts := strings.Split(cb.Data, ":")
	if len(parts) != 3 {
		h.client.AnswerCallbackQuery(cb.ID, "Неверные данные", true)
		return
	}

	action := parts[1]
	sessionID, _ := strconv.ParseUint(parts[2], 10, 64)
	if sessionID == 0 {
		h.client.AnswerCallbackQuery(cb.ID, "Неверные данные", true)
		return
	}

	us := h.state.Get(cb.From.ID)
	if us.State != StateHostRemote {
		h.client.AnswerCallbackQuery(cb.ID, "Авторизуйтесь заново: /start → 🎯 Пульт ведущего", true)
		return
	}

	if action == "pick" {
		h.handleHostPick(cb, uint(sessionID))
		return
	}

	h.handleHostAction(cb, action, uint(sessionID))
}

func (h *UpdateHandler) handleAnswerCallback(cb *CallbackQuery) {
	userID := cb.From.ID
	us := h.state.Get(userID)
	if us.State != StateInSession {
		h.client.AnswerCallbackQuery(cb.ID, "Вы не в активной сессии. Нажмите /rejoin", true)
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
		} else if strings.Contains(errText, "participant not found") {
			h.client.AnswerCallbackQuery(cb.ID, "Переподключение...", false)
			if us.Code != "" && us.Nickname != "" {
				go h.doJoin(userID, cb.Message.Chat.ID, us.Code, us.Nickname)
			}
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

// ─── Helpers ───

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
