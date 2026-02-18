package telegram

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"quiz-game-backend/internal/services"
)

type ParticipantInfo struct {
	ChatID     int64
	TelegramID int64
	MessageID  int64
}

type HostRemoteInfo struct {
	ChatID    int64
	MessageID int64
}

type SessionInfo struct {
	SessionID       uint
	LastStatus      string
	LastQuestion    int
	LastAnswerCount int
	Participants    map[int64]*ParticipantInfo
	HostRemote      *HostRemoteInfo
	mu              sync.Mutex
}

type SessionTracker struct {
	client       *Client
	state        *StateManager
	sessionSvc   *services.SessionService
	pollInterval time.Duration

	mu       sync.Mutex
	sessions map[uint]*SessionInfo
	stopChs  map[uint]chan struct{}
}

func NewSessionTracker(
	client *Client,
	state *StateManager,
	sessionSvc *services.SessionService,
	pollInterval time.Duration,
) *SessionTracker {
	return &SessionTracker{
		client:       client,
		state:        state,
		sessionSvc:   sessionSvc,
		pollInterval: pollInterval,
		sessions:     make(map[uint]*SessionInfo),
		stopChs:      make(map[uint]chan struct{}),
	}
}

func (t *SessionTracker) AddParticipant(sessionID uint, telegramID, chatID, messageID int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	info, exists := t.sessions[sessionID]
	if !exists {
		info = &SessionInfo{
			SessionID:    sessionID,
			Participants: make(map[int64]*ParticipantInfo),
		}
		t.sessions[sessionID] = info

		stopCh := make(chan struct{})
		t.stopChs[sessionID] = stopCh
		go t.pollLoop(sessionID, stopCh)
	}

	info.mu.Lock()
	info.Participants[telegramID] = &ParticipantInfo{
		ChatID:     chatID,
		TelegramID: telegramID,
		MessageID:  messageID,
	}
	info.mu.Unlock()
}

func (t *SessionTracker) SetHostRemote(sessionID uint, chatID, messageID int64) {
	t.mu.Lock()
	info, exists := t.sessions[sessionID]
	if !exists {
		info = &SessionInfo{
			SessionID:    sessionID,
			Participants: make(map[int64]*ParticipantInfo),
		}
		t.sessions[sessionID] = info

		stopCh := make(chan struct{})
		t.stopChs[sessionID] = stopCh
		go t.pollLoop(sessionID, stopCh)
	}
	t.mu.Unlock()

	info.mu.Lock()
	info.HostRemote = &HostRemoteInfo{ChatID: chatID, MessageID: messageID}
	info.mu.Unlock()
}

func (t *SessionTracker) SendHostControl(sessionID uint) {
	t.mu.Lock()
	info, ok := t.sessions[sessionID]
	t.mu.Unlock()
	if !ok {
		return
	}

	sessState, err := t.sessionSvc.GetSession(sessionID)
	if err != nil {
		return
	}

	info.mu.Lock()
	hr := info.HostRemote
	info.mu.Unlock()
	if hr == nil {
		return
	}

	text := t.buildHostControlText(sessState)
	kb := HostControlKeyboard(sessionID, sessState.Status, sessState.CurrentQuestion, sessState.TotalQuestions)

	msgID := t.sendOrEditHost(hr, text, kb)
	if msgID > 0 {
		info.mu.Lock()
		if info.HostRemote != nil {
			info.HostRemote.MessageID = msgID
		}
		info.mu.Unlock()
	}
}

func (t *SessionTracker) buildHostControlText(s *services.SessionState) string {
	participantCount := len(s.Participants)

	switch s.Status {
	case "waiting":
		return fmt.Sprintf("🎯 <b>Пульт ведущего</b>\n\n📋 Квиз: <b>%s</b>\n🔑 Код: <b>%s</b>\n👥 Участников: <b>%d</b>\n📝 Вопросов: <b>%d</b>\n\n⏳ Ожидание участников...",
			s.Quiz.Title, s.Code, participantCount, s.TotalQuestions)

	case "question":
		qText := ""
		catText := ""
		if s.CurrentQuestionData != nil {
			qText = s.CurrentQuestionData.Text
			if s.CurrentQuestionData.CategoryName != "" {
				catText = fmt.Sprintf("\n📁 Категория: <b>%s</b>", s.CurrentQuestionData.CategoryName)
			}
		}
		return fmt.Sprintf("🎯 <b>Пульт ведущего</b>\n\n❓ <b>Вопрос %d из %d</b>%s\n\n%s\n\n📊 Ответили: <b>%d</b> из <b>%d</b>",
			s.CurrentQuestion, s.TotalQuestions, catText, qText, s.AnswerCount, participantCount)

	case "revealed":
		qText := ""
		correctText := ""
		if s.CurrentQuestionData != nil {
			qText = s.CurrentQuestionData.Text
			for _, opt := range s.CurrentQuestionData.Options {
				if opt.IsCorrect != nil && *opt.IsCorrect {
					correctText = fmt.Sprintf("\n\n✅ Правильный ответ: <b>%s</b>", opt.Text)
					break
				}
			}
		}
		return fmt.Sprintf("🎯 <b>Пульт ведущего</b>\n\n❓ <b>Вопрос %d из %d</b>\n\n%s%s\n\n📊 Ответили: <b>%d</b> из <b>%d</b>",
			s.CurrentQuestion, s.TotalQuestions, qText, correctText, s.AnswerCount, participantCount)

	case "finished":
		return fmt.Sprintf("🎯 <b>Пульт ведущего</b>\n\n🏆 <b>Квиз завершён!</b>\n📋 %s\n👥 Участников: <b>%d</b>",
			s.Quiz.Title, participantCount)

	default:
		return "🎯 <b>Пульт ведущего</b>\n\nСессия не найдена."
	}
}

func (t *SessionTracker) sendOrEditHost(hr *HostRemoteInfo, text string, kb interface{}) int64 {
	if hr.MessageID > 0 {
		if err := t.client.EditMessageText(hr.ChatID, hr.MessageID, text, "HTML", kb); err == nil {
			return 0
		}
	}
	msgID, err := t.client.SendMessage(hr.ChatID, text, "HTML", kb)
	if err != nil {
		log.Printf("send host control to %d: %v", hr.ChatID, err)
		return 0
	}
	return msgID
}

// SyncParticipant immediately sends the current session state to one participant.
// Used after (re)join to avoid waiting for the next poll cycle.
func (t *SessionTracker) SyncParticipant(sessionID uint, telegramID int64) {
	t.mu.Lock()
	info, ok := t.sessions[sessionID]
	t.mu.Unlock()
	if !ok {
		return
	}

	sessState, err := t.sessionSvc.GetSession(sessionID)
	if err != nil {
		return
	}

	info.mu.Lock()
	p, pOk := info.Participants[telegramID]
	info.mu.Unlock()
	if !pOk {
		return
	}

	status := sessState.Status

	switch status {
	case "question":
		if sessState.CurrentQuestionData == nil {
			return
		}
		t.syncSendQuestion(info, sessState, telegramID, p)
	case "revealed":
		t.syncSendResult(info, sessState, telegramID, p)
	case "finished":
		// will be handled by the poll loop
	}
}

func (t *SessionTracker) syncSendQuestion(info *SessionInfo, sessState *services.SessionState, tgID int64, p *ParticipantInfo) {
	qd := sessState.CurrentQuestionData
	current := sessState.CurrentQuestion
	total := sessState.TotalQuestions
	text := fmt.Sprintf("❓ <b>Вопрос %d из %d</b>\n\n%s", current, total, qd.Text)

	var opts []QuestionOption
	for _, o := range qd.Options {
		opts = append(opts, QuestionOption{ID: o.ID, Text: o.Text})
	}
	kb := AnswerKeyboard(info.SessionID, opts, 0)

	msgID := t.sendOrEdit(p, text, kb)
	if msgID > 0 {
		info.mu.Lock()
		if pp, ok := info.Participants[tgID]; ok {
			pp.MessageID = msgID
		}
		info.mu.Unlock()
	}

	t.updateFSM(tgID, info.SessionID, qd.Text, opts, current, total)
}

func (t *SessionTracker) syncSendResult(info *SessionInfo, sessState *services.SessionState, tgID int64, p *ParticipantInfo) {
	qd := sessState.CurrentQuestionData
	current := sessState.CurrentQuestion
	total := sessState.TotalQuestions

	result, err := t.sessionSvc.GetParticipantResult(info.SessionID, tgID)
	if err != nil {
		return
	}

	text := t.buildResultText(qd, result, current, total)
	msgID := t.sendOrEdit(p, text, nil)
	if msgID > 0 {
		info.mu.Lock()
		if pp, ok := info.Participants[tgID]; ok {
			pp.MessageID = msgID
		}
		info.mu.Unlock()
	}
}

func (t *SessionTracker) removeSession(sessionID uint) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.sessions, sessionID)
	if ch, ok := t.stopChs[sessionID]; ok {
		close(ch)
		delete(t.stopChs, sessionID)
	}
}

func (t *SessionTracker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for id, ch := range t.stopChs {
		close(ch)
		delete(t.stopChs, id)
	}
	t.sessions = make(map[uint]*SessionInfo)
}

func (t *SessionTracker) pollLoop(sessionID uint, stopCh chan struct{}) {
	ticker := time.NewTicker(t.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			t.checkSession(sessionID)
		}
	}
}

func (t *SessionTracker) checkSession(sessionID uint) {
	t.mu.Lock()
	info, ok := t.sessions[sessionID]
	t.mu.Unlock()
	if !ok {
		return
	}

	sessState, err := t.sessionSvc.GetSession(sessionID)
	if err != nil {
		return
	}

	status := sessState.Status
	currentQ := sessState.CurrentQuestion
	ansCount := sessState.AnswerCount

	info.mu.Lock()
	prevStatus := info.LastStatus
	prevQ := info.LastQuestion
	prevAns := info.LastAnswerCount
	hasHostRemote := info.HostRemote != nil

	statusChanged := prevStatus != status || prevQ != currentQ
	answerCountChanged := prevAns != ansCount

	if statusChanged {
		info.LastStatus = status
		info.LastQuestion = currentQ
	}
	if answerCountChanged {
		info.LastAnswerCount = ansCount
	}
	info.mu.Unlock()

	if !statusChanged && !answerCountChanged {
		return
	}

	if statusChanged {
		if status == "question" && sessState.CurrentQuestionData != nil && currentQ != prevQ {
			t.sendQuestion(info, sessState)
		} else if status == "revealed" && prevStatus == "question" {
			t.sendResults(info, sessState)
		} else if status == "finished" && prevStatus != "finished" {
			t.sendLeaderboard(info)

			if hasHostRemote {
				t.updateHostControl(info, sessState)
			}
			t.removeSession(sessionID)
			return
		}
	}

	if hasHostRemote && (statusChanged || answerCountChanged) {
		t.updateHostControl(info, sessState)
	}
}

func (t *SessionTracker) updateHostControl(info *SessionInfo, sessState *services.SessionState) {
	info.mu.Lock()
	hr := info.HostRemote
	info.mu.Unlock()
	if hr == nil {
		return
	}

	text := t.buildHostControlText(sessState)
	kb := HostControlKeyboard(info.SessionID, sessState.Status, sessState.CurrentQuestion, sessState.TotalQuestions)

	msgID := t.sendOrEditHost(hr, text, kb)
	if msgID > 0 {
		info.mu.Lock()
		if info.HostRemote != nil {
			info.HostRemote.MessageID = msgID
		}
		info.mu.Unlock()
	}
}

// sendOrEdit tries to edit the existing message; on failure sends a new one.
// Returns the new messageID if a new message was sent, 0 if edit succeeded.
func (t *SessionTracker) sendOrEdit(p *ParticipantInfo, text string, kb interface{}) int64 {
	if p.MessageID > 0 {
		if err := t.client.EditMessageText(p.ChatID, p.MessageID, text, "HTML", kb); err == nil {
			return 0
		}
	}
	msgID, err := t.client.SendMessage(p.ChatID, text, "HTML", kb)
	if err != nil {
		log.Printf("send msg to %d: %v", p.ChatID, err)
		return 0
	}
	return msgID
}

func (t *SessionTracker) sendQuestion(info *SessionInfo, sessState *services.SessionState) {
	qd := sessState.CurrentQuestionData
	current := sessState.CurrentQuestion
	total := sessState.TotalQuestions
	text := fmt.Sprintf("❓ <b>Вопрос %d из %d</b>\n\n%s", current, total, qd.Text)

	var opts []QuestionOption
	for _, o := range qd.Options {
		opts = append(opts, QuestionOption{ID: o.ID, Text: o.Text})
	}
	kb := AnswerKeyboard(info.SessionID, opts, 0)

	info.mu.Lock()
	participants := make(map[int64]*ParticipantInfo, len(info.Participants))
	for k, v := range info.Participants {
		participants[k] = v
	}
	info.mu.Unlock()

	for tgID, p := range participants {
		msgID := t.sendOrEdit(p, text, kb)
		if msgID > 0 {
			info.mu.Lock()
			if pp, ok := info.Participants[tgID]; ok {
				pp.MessageID = msgID
			}
			info.mu.Unlock()
		}
		t.updateFSM(tgID, info.SessionID, qd.Text, opts, current, total)
	}
}

func (t *SessionTracker) updateFSM(userID int64, sessionID uint, qText string, opts []QuestionOption, current, total int) {
	t.state.UpdateField(userID, func(s *UserState) {
		s.QuestionData = &QuestionData{
			Text:      qText,
			SessionID: sessionID,
			Options:   opts,
		}
		s.CurrentQNum = current
		s.TotalQuestions = total
		s.SelectedOptionID = 0
	})
}

func (t *SessionTracker) buildResultText(qd *services.QuestionResponse, result *services.ParticipantResult, current, total int) string {
	var resultLine, scoreLine string
	if !result.Answered {
		resultLine = "⏰ Вы не успели ответить"
		scoreLine = fmt.Sprintf("\nВсего очков: <b>%d</b>", result.TotalScore)
	} else if result.IsCorrect {
		resultLine = "✅ <b>Правильно!</b>"
		scoreLine = fmt.Sprintf("\nОчки за вопрос: <b>+%d</b> | Всего: <b>%d</b>", result.Score, result.TotalScore)
	} else {
		resultLine = "❌ <b>Неправильно</b>"
		scoreLine = fmt.Sprintf("\nВсего очков: <b>%d</b>", result.TotalScore)
	}

	correctText := ""
	if qd != nil {
		for _, opt := range qd.Options {
			if opt.IsCorrect != nil && *opt.IsCorrect {
				correctText = fmt.Sprintf("\n\nПравильный ответ: <b>%s</b>", opt.Text)
				break
			}
		}
	}

	questionText := ""
	if qd != nil {
		questionText = qd.Text
	}

	return fmt.Sprintf("❓ <b>Вопрос %d из %d</b>\n\n%s\n\n%s%s%s\n\n⏳ Ожидайте следующий вопрос...",
		current, total, questionText, resultLine, scoreLine, correctText)
}

func (t *SessionTracker) sendResults(info *SessionInfo, sessState *services.SessionState) {
	qd := sessState.CurrentQuestionData
	current := sessState.CurrentQuestion
	total := sessState.TotalQuestions

	info.mu.Lock()
	participants := make(map[int64]*ParticipantInfo, len(info.Participants))
	for k, v := range info.Participants {
		participants[k] = v
	}
	info.mu.Unlock()

	for tgID, p := range participants {
		result, err := t.sessionSvc.GetParticipantResult(info.SessionID, tgID)
		if err != nil {
			continue
		}

		text := t.buildResultText(qd, result, current, total)
		msgID := t.sendOrEdit(p, text, nil)
		if msgID > 0 {
			info.mu.Lock()
			if pp, ok := info.Participants[tgID]; ok {
				pp.MessageID = msgID
			}
			info.mu.Unlock()
		}
	}
}

func (t *SessionTracker) sendLeaderboard(info *SessionInfo) {
	entries, err := t.sessionSvc.GetLeaderboard(info.SessionID)
	if err != nil {
		return
	}

	medals := map[int]string{1: "🥇", 2: "🥈", 3: "🥉"}
	lines := []string{"🏆 <b>Квиз завершён! Итоги:</b>\n"}
	for _, e := range entries {
		medal, ok := medals[e.Position]
		if !ok {
			medal = fmt.Sprintf("%d.", e.Position)
		}
		lines = append(lines, fmt.Sprintf("%s <b>%s</b> — %d очков", medal, e.Nickname, e.TotalScore))
	}
	baseText := strings.Join(lines, "\n")

	info.mu.Lock()
	participants := make(map[int64]*ParticipantInfo, len(info.Participants))
	for k, v := range info.Participants {
		participants[k] = v
	}
	info.mu.Unlock()

	for tgID, p := range participants {
		personal := baseText
		for _, e := range entries {
			if e.TelegramID == tgID {
				personal += fmt.Sprintf("\n\n📍 Ваше место: <b>%d</b>", e.Position)
				break
			}
		}
		personal += "\n\nДля новой игры нажмите /start"

		if p.MessageID > 0 {
			t.client.EditMessageText(p.ChatID, p.MessageID, personal, "HTML", nil)
		} else {
			t.client.SendMessage(p.ChatID, personal, "HTML", nil)
		}

		t.state.Clear(tgID)
	}
}
