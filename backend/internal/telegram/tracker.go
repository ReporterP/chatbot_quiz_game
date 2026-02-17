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

type SessionInfo struct {
	SessionID    uint
	LastStatus   string
	LastQuestion int
	Participants map[int64]*ParticipantInfo
	mu           sync.Mutex
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

	info.mu.Lock()
	prevStatus := info.LastStatus
	prevQ := info.LastQuestion

	if prevStatus == status && prevQ == currentQ {
		info.mu.Unlock()
		return
	}

	info.LastStatus = status
	info.LastQuestion = currentQ
	info.mu.Unlock()

	if status == "question" && sessState.CurrentQuestionData != nil && currentQ != prevQ {
		t.sendQuestion(info, sessState)
	} else if status == "revealed" && prevStatus == "question" {
		t.sendResults(info, sessState)
	} else if status == "finished" && prevStatus != "finished" {
		t.sendLeaderboard(info)
		t.removeSession(sessionID)
	}
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
		if p.MessageID > 0 {
			if err := t.client.EditMessageText(p.ChatID, p.MessageID, text, "HTML", kb); err == nil {
				t.updateFSM(tgID, info.SessionID, opts, current, total)
				continue
			}
		}

		msgID, err := t.client.SendMessage(p.ChatID, text, "HTML", kb)
		if err != nil {
			log.Printf("send question to %d: %v", p.ChatID, err)
			continue
		}

		info.mu.Lock()
		if pp, ok := info.Participants[tgID]; ok {
			pp.MessageID = msgID
		}
		info.mu.Unlock()

		t.updateFSM(tgID, info.SessionID, opts, current, total)
	}
}

func (t *SessionTracker) updateFSM(userID int64, sessionID uint, opts []QuestionOption, current, total int) {
	t.state.UpdateField(userID, func(s *UserState) {
		s.QuestionData = &QuestionData{Options: opts}
		s.CurrentQNum = current
		s.TotalQuestions = total
		s.SelectedOptionID = 0
	})
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

		var resultLine, scoreLine string
		if !result.Answered {
			resultLine = "⏰ Вы не успели ответить"
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

		text := fmt.Sprintf("❓ <b>Вопрос %d из %d</b>\n\n%s\n\n%s%s%s\n\n⏳ Ожидайте следующий вопрос...",
			current, total, questionText, resultLine, scoreLine, correctText)

		if p.MessageID > 0 {
			if err := t.client.EditMessageText(p.ChatID, p.MessageID, text, "HTML", nil); err != nil {
				log.Printf("edit result for %d: %v", p.ChatID, err)
			}
		} else {
			msgID, err := t.client.SendMessage(p.ChatID, text, "HTML", nil)
			if err == nil {
				info.mu.Lock()
				if pp, ok := info.Participants[tgID]; ok {
					pp.MessageID = msgID
				}
				info.mu.Unlock()
			}
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
