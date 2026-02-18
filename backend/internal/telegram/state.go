package telegram

import "sync"

const (
	StateNone          = ""
	StateEnterCode     = "enter_code"
	StateEnterNickname = "enter_nickname"
	StateInSession     = "in_session"
	StateHostPassword  = "host_password"
	StateHostRemote    = "host_remote"
)

type QuestionOption struct {
	ID   uint   `json:"id"`
	Text string `json:"text"`
}

type QuestionData struct {
	Text      string           `json:"text"`
	SessionID uint             `json:"session_id"`
	Options   []QuestionOption `json:"options"`
}

type UserState struct {
	State            string
	Code             string
	Nickname         string
	SessionID        uint
	QuestionData     *QuestionData
	CurrentQNum      int
	TotalQuestions   int
	SelectedOptionID uint
}

type StateManager struct {
	mu    sync.RWMutex
	users map[int64]*UserState
}

func NewStateManager() *StateManager {
	return &StateManager{
		users: make(map[int64]*UserState),
	}
}

func (m *StateManager) Get(userID int64) *UserState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.users[userID]
	if !ok {
		return &UserState{}
	}
	cp := *s
	return &cp
}

func (m *StateManager) Set(userID int64, state *UserState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[userID] = state
}

func (m *StateManager) Clear(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.users, userID)
}

func (m *StateManager) UpdateField(userID int64, fn func(s *UserState)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.users[userID]
	if !ok {
		s = &UserState{}
		m.users[userID] = s
	}
	fn(s)
}
