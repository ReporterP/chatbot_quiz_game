package services

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"quiz-game-backend/internal/models"

	"gorm.io/gorm"
)

type SessionService struct {
	db      *gorm.DB
	scoring *ScoringService
}

func NewSessionService(db *gorm.DB, scoring *ScoringService) *SessionService {
	return &SessionService{db: db, scoring: scoring}
}

func (s *SessionService) getOrderedQuestions(quizID uint) []questionWithMeta {
	var categories []models.Category
	s.db.Where("quiz_id = ?", quizID).
		Order("order_num ASC").
		Preload("Questions", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_num ASC")
		}).
		Preload("Questions.Options").
		Preload("Questions.Images", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_num ASC")
		}).
		Find(&categories)

	var result []questionWithMeta
	for _, cat := range categories {
		for _, q := range cat.Questions {
			result = append(result, questionWithMeta{Question: q, CategoryName: cat.Title})
		}
	}

	var orphans []models.Question
	s.db.Where("quiz_id = ? AND category_id IS NULL", quizID).
		Order("order_num ASC").
		Preload("Options").
		Preload("Images", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_num ASC")
		}).
		Find(&orphans)

	for _, q := range orphans {
		result = append(result, questionWithMeta{Question: q, CategoryName: ""})
	}

	return result
}

func (s *SessionService) CreateSessionInRoom(roomID, quizID, hostID uint) (*models.Session, error) {
	var quiz models.Quiz
	if err := s.db.Where("id = ? AND host_id = ?", quizID, hostID).First(&quiz).Error; err != nil {
		return nil, errors.New("quiz not found")
	}

	questions := s.getOrderedQuestions(quizID)
	if len(questions) == 0 {
		return nil, errors.New("quiz must have at least one question")
	}

	// Finish any active session in this room
	s.db.Model(&models.Session{}).
		Where("room_id = ? AND status != ?", roomID, models.SessionStatusFinished).
		Update("status", models.SessionStatusFinished)

	code := s.generateUniqueCode()
	session := models.Session{
		RoomID:          roomID,
		QuizID:          quizID,
		HostID:          hostID,
		Code:            code,
		Status:          models.SessionStatusWaiting,
		CurrentQuestion: 0,
	}
	if err := s.db.Create(&session).Error; err != nil {
		return nil, err
	}

	// Auto-create participants for all room members
	var members []models.RoomMember
	s.db.Where("room_id = ?", roomID).Find(&members)
	for _, m := range members {
		p := models.Participant{
			SessionID:  session.ID,
			MemberID:   m.ID,
			Nickname:   m.Nickname,
			TotalScore: 0,
			JoinedAt:   time.Now(),
		}
		s.db.Create(&p)
	}

	s.db.Preload("Quiz").First(&session, session.ID)
	return &session, nil
}

// AddLateParticipant adds a participant who joined the room after the session started
func (s *SessionService) AddLateParticipant(sessionID, memberID uint, nickname string) (*models.Participant, error) {
	var existing models.Participant
	if err := s.db.Where("session_id = ? AND member_id = ?", sessionID, memberID).
		First(&existing).Error; err == nil {
		return &existing, nil
	}

	p := models.Participant{
		SessionID:  sessionID,
		MemberID:   memberID,
		Nickname:   nickname,
		TotalScore: 0,
		JoinedAt:   time.Now(),
	}
	if err := s.db.Create(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *SessionService) GetSession(sessionID uint) (*SessionState, error) {
	var session models.Session
	if err := s.db.Preload("Quiz").
		Preload("Participants", func(db *gorm.DB) *gorm.DB {
			return db.Order("total_score DESC")
		}).
		First(&session, sessionID).Error; err != nil {
		return nil, errors.New("session not found")
	}

	questions := s.getOrderedQuestions(session.QuizID)

	state := &SessionState{
		Session:        session,
		TotalQuestions: len(questions),
	}

	if session.CurrentQuestion > 0 && session.CurrentQuestion <= len(questions) {
		qm := questions[session.CurrentQuestion-1]
		q := qm.Question

		qr := QuestionResponse{
			ID:           q.ID,
			Text:         q.Text,
			OrderNum:     q.OrderNum,
			CategoryName: qm.CategoryName,
		}

		for _, img := range q.Images {
			qr.Images = append(qr.Images, ImageResponse{ID: img.ID, URL: img.URL})
		}

		for _, o := range q.Options {
			opt := OptionResponse{
				ID:    o.ID,
				Text:  o.Text,
				Color: o.Color,
			}
			if session.Status == models.SessionStatusRevealed || session.Status == models.SessionStatusFinished {
				correct := o.IsCorrect
				opt.IsCorrect = &correct
			}
			qr.Options = append(qr.Options, opt)
		}
		state.CurrentQuestionData = &qr

		var answerCount int64
		s.db.Model(&models.Answer{}).
			Where("session_id = ? AND question_id = ?", sessionID, q.ID).
			Count(&answerCount)
		state.AnswerCount = int(answerCount)
	}

	return state, nil
}

func (s *SessionService) StartQuiz(sessionID, hostID uint) (*SessionState, error) {
	var session models.Session
	if err := s.db.Where("id = ? AND host_id = ?", sessionID, hostID).First(&session).Error; err != nil {
		return nil, errors.New("session not found")
	}

	if session.Status != models.SessionStatusWaiting {
		return nil, errors.New("quiz already started")
	}

	questions := s.getOrderedQuestions(session.QuizID)
	if len(questions) == 0 {
		return nil, errors.New("no questions in quiz")
	}

	session.Status = models.SessionStatusQuestion
	session.CurrentQuestion = 1
	s.db.Save(&session)

	return s.GetSession(sessionID)
}

func (s *SessionService) NextQuestion(sessionID, hostID uint) (*SessionState, error) {
	var session models.Session
	if err := s.db.Where("id = ? AND host_id = ?", sessionID, hostID).First(&session).Error; err != nil {
		return nil, errors.New("session not found")
	}

	if session.Status == models.SessionStatusWaiting {
		return s.StartQuiz(sessionID, hostID)
	}

	if session.Status != models.SessionStatusRevealed {
		return nil, errors.New("must reveal answer before moving to next question")
	}

	questions := s.getOrderedQuestions(session.QuizID)

	if session.CurrentQuestion >= len(questions) {
		session.Status = models.SessionStatusFinished
		s.db.Save(&session)
		return s.GetSession(sessionID)
	}

	session.CurrentQuestion++
	session.Status = models.SessionStatusQuestion
	s.db.Save(&session)

	return s.GetSession(sessionID)
}

func (s *SessionService) RevealAnswer(sessionID, hostID uint) (*SessionState, error) {
	var session models.Session
	if err := s.db.Where("id = ? AND host_id = ?", sessionID, hostID).First(&session).Error; err != nil {
		return nil, errors.New("session not found")
	}

	if session.Status != models.SessionStatusQuestion {
		return nil, errors.New("no active question to reveal")
	}

	questions := s.getOrderedQuestions(session.QuizID)
	if session.CurrentQuestion < 1 || session.CurrentQuestion > len(questions) {
		return nil, errors.New("invalid question index")
	}

	currentQ := questions[session.CurrentQuestion-1].Question

	var answers []models.Answer
	s.db.Where("session_id = ? AND question_id = ?", sessionID, currentQ.ID).
		Order("answered_at ASC").
		Find(&answers)

	var totalParticipants int64
	s.db.Model(&models.Participant{}).Where("session_id = ?", sessionID).Count(&totalParticipants)

	answers = s.scoring.CalculateScores(answers, int(totalParticipants))

	tx := s.db.Begin()
	for _, a := range answers {
		tx.Model(&models.Answer{}).Where("id = ?", a.ID).Update("score", a.Score)
		tx.Model(&models.Participant{}).Where("id = ?", a.ParticipantID).
			Update("total_score", gorm.Expr("total_score + ?", a.Score))
	}

	session.Status = models.SessionStatusRevealed
	tx.Save(&session)
	tx.Commit()

	return s.GetSession(sessionID)
}

func (s *SessionService) ForceFinish(sessionID, hostID uint) (*SessionState, error) {
	var session models.Session
	if err := s.db.Where("id = ? AND host_id = ?", sessionID, hostID).First(&session).Error; err != nil {
		return nil, errors.New("session not found")
	}

	if session.Status == models.SessionStatusFinished {
		return nil, errors.New("session already finished")
	}

	session.Status = models.SessionStatusFinished
	s.db.Save(&session)

	return s.GetSession(sessionID)
}

func (s *SessionService) GetLeaderboard(sessionID uint) ([]LeaderboardEntry, error) {
	var participants []models.Participant
	if err := s.db.Where("session_id = ?", sessionID).
		Order("total_score DESC").
		Find(&participants).Error; err != nil {
		return nil, err
	}

	entries := make([]LeaderboardEntry, len(participants))
	for i, p := range participants {
		entries[i] = LeaderboardEntry{
			Position:   i + 1,
			Nickname:   p.Nickname,
			TotalScore: p.TotalScore,
			MemberID:   p.MemberID,
			TelegramID: p.TelegramID,
		}
	}
	return entries, nil
}

func (s *SessionService) GetActiveSessions(hostID uint) ([]SessionSummary, error) {
	var sessions []models.Session
	if err := s.db.Where("host_id = ? AND status != ?", hostID, models.SessionStatusFinished).
		Preload("Quiz").
		Order("created_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, err
	}

	result := make([]SessionSummary, 0, len(sessions))
	for _, sess := range sessions {
		var participantCount int64
		s.db.Model(&models.Participant{}).Where("session_id = ?", sess.ID).Count(&participantCount)

		result = append(result, SessionSummary{
			ID:               sess.ID,
			QuizTitle:        sess.Quiz.Title,
			Code:             sess.Code,
			Status:           sess.Status,
			ParticipantCount: int(participantCount),
			CreatedAt:        sess.CreatedAt,
		})
	}
	return result, nil
}

func (s *SessionService) ListSessions(hostID uint) ([]SessionSummary, error) {
	var sessions []models.Session
	if err := s.db.Where("host_id = ?", hostID).
		Preload("Quiz").
		Order("created_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, err
	}

	result := make([]SessionSummary, len(sessions))
	for i, sess := range sessions {
		var participantCount int64
		s.db.Model(&models.Participant{}).Where("session_id = ?", sess.ID).Count(&participantCount)

		result[i] = SessionSummary{
			ID:               sess.ID,
			QuizTitle:        sess.Quiz.Title,
			Code:             sess.Code,
			Status:           sess.Status,
			ParticipantCount: int(participantCount),
			CreatedAt:        sess.CreatedAt,
		}
	}
	return result, nil
}

func (s *SessionService) SubmitAnswerByMember(sessionID, memberID, optionID uint) error {
	var session models.Session
	if err := s.db.First(&session, sessionID).Error; err != nil {
		return errors.New("session not found")
	}

	if session.Status != models.SessionStatusQuestion {
		return errors.New("session is not accepting answers")
	}

	var participant models.Participant
	if err := s.db.Where("session_id = ? AND member_id = ?", sessionID, memberID).
		First(&participant).Error; err != nil {
		return errors.New("participant not found in session")
	}

	questions := s.getOrderedQuestions(session.QuizID)
	if session.CurrentQuestion < 1 || session.CurrentQuestion > len(questions) {
		return errors.New("invalid question state")
	}
	currentQ := questions[session.CurrentQuestion-1].Question

	var option models.Option
	if err := s.db.Where("id = ? AND question_id = ?", optionID, currentQ.ID).
		First(&option).Error; err != nil {
		return errors.New("invalid option for current question")
	}

	var existingAnswer models.Answer
	if err := s.db.Where("session_id = ? AND participant_id = ? AND question_id = ?",
		sessionID, participant.ID, currentQ.ID).First(&existingAnswer).Error; err == nil {
		existingAnswer.OptionID = optionID
		existingAnswer.IsCorrect = option.IsCorrect
		existingAnswer.AnsweredAt = time.Now()
		return s.db.Save(&existingAnswer).Error
	}

	answer := models.Answer{
		SessionID:     sessionID,
		ParticipantID: participant.ID,
		QuestionID:    currentQ.ID,
		OptionID:      optionID,
		IsCorrect:     option.IsCorrect,
		Score:         0,
		AnsweredAt:    time.Now(),
	}
	return s.db.Create(&answer).Error
}

func (s *SessionService) GetParticipantResultByMember(sessionID, memberID uint) (*ParticipantResult, error) {
	var session models.Session
	if err := s.db.First(&session, sessionID).Error; err != nil {
		return nil, errors.New("session not found")
	}

	var participant models.Participant
	if err := s.db.Where("session_id = ? AND member_id = ?", sessionID, memberID).
		First(&participant).Error; err != nil {
		return nil, errors.New("participant not found")
	}

	if session.Status != models.SessionStatusRevealed && session.Status != models.SessionStatusFinished {
		return &ParticipantResult{
			TotalScore: participant.TotalScore,
			Answered:   false,
		}, nil
	}

	questions := s.getOrderedQuestions(session.QuizID)
	if session.CurrentQuestion < 1 || session.CurrentQuestion > len(questions) {
		return nil, errors.New("invalid question state")
	}
	currentQ := questions[session.CurrentQuestion-1].Question

	var answer models.Answer
	if err := s.db.Where("session_id = ? AND participant_id = ? AND question_id = ?",
		sessionID, participant.ID, currentQ.ID).First(&answer).Error; err != nil {
		return &ParticipantResult{
			TotalScore: participant.TotalScore,
			Answered:   false,
		}, nil
	}

	return &ParticipantResult{
		QuestionID: currentQ.ID,
		OptionID:   answer.OptionID,
		IsCorrect:  answer.IsCorrect,
		Score:      answer.Score,
		TotalScore: participant.TotalScore,
		Answered:   true,
	}, nil
}

func (s *SessionService) generateUniqueCode() string {
	for {
		code := fmt.Sprintf("%06d", rand.Intn(1000000))
		var count int64
		s.db.Model(&models.Session{}).
			Where("code = ? AND status != ?", code, models.SessionStatusFinished).
			Count(&count)
		if count == 0 {
			return code
		}
	}
}

type questionWithMeta struct {
	Question     models.Question
	CategoryName string
}

type SessionState struct {
	models.Session
	TotalQuestions       int               `json:"total_questions"`
	CurrentQuestionData *QuestionResponse  `json:"current_question_data,omitempty"`
	AnswerCount         int                `json:"answer_count"`
}

type QuestionResponse struct {
	ID           uint             `json:"id"`
	Text         string           `json:"text"`
	OrderNum     int              `json:"order_num"`
	CategoryName string           `json:"category_name,omitempty"`
	Options      []OptionResponse `json:"options"`
	Images       []ImageResponse  `json:"images,omitempty"`
}

type OptionResponse struct {
	ID        uint   `json:"id"`
	Text      string `json:"text"`
	Color     string `json:"color,omitempty"`
	IsCorrect *bool  `json:"is_correct,omitempty"`
}

type ImageResponse struct {
	ID  uint   `json:"id"`
	URL string `json:"url"`
}

type LeaderboardEntry struct {
	Position   int    `json:"position"`
	Nickname   string `json:"nickname"`
	TotalScore int    `json:"total_score"`
	MemberID   uint   `json:"member_id"`
	TelegramID int64  `json:"telegram_id,omitempty"`
}

type ParticipantResult struct {
	QuestionID uint `json:"question_id,omitempty"`
	OptionID   uint `json:"option_id,omitempty"`
	IsCorrect  bool `json:"is_correct"`
	Score      int  `json:"score"`
	TotalScore int  `json:"total_score"`
	Answered   bool `json:"answered"`
}

type SessionSummary struct {
	ID               uint      `json:"id"`
	QuizTitle        string    `json:"quiz_title"`
	Code             string    `json:"code"`
	Status           string    `json:"status"`
	ParticipantCount int       `json:"participant_count"`
	CreatedAt        time.Time `json:"created_at"`
}

type JoinResult struct {
	SessionID   uint               `json:"session_id"`
	Participant models.Participant `json:"participant"`
	IsRejoin    bool               `json:"is_rejoin"`
}

// Legacy methods for backward compatibility with bot/participant handlers

func (s *SessionService) CreateSession(quizID, hostID uint) (*models.Session, error) {
	var quiz models.Quiz
	if err := s.db.Where("id = ? AND host_id = ?", quizID, hostID).First(&quiz).Error; err != nil {
		return nil, errors.New("quiz not found")
	}

	questions := s.getOrderedQuestions(quizID)
	if len(questions) == 0 {
		return nil, errors.New("quiz must have at least one question")
	}

	code := s.generateUniqueCode()
	session := models.Session{
		QuizID:          quizID,
		HostID:          hostID,
		Code:            code,
		Status:          models.SessionStatusWaiting,
		CurrentQuestion: 0,
	}
	if err := s.db.Create(&session).Error; err != nil {
		return nil, err
	}

	s.db.Preload("Quiz").First(&session, session.ID)
	return &session, nil
}

func (s *SessionService) JoinSession(code string, telegramID int64, nickname string) (*JoinResult, error) {
	var session models.Session
	if err := s.db.Where("code = ? AND status != ?", code, models.SessionStatusFinished).
		First(&session).Error; err != nil {
		return nil, errors.New("session not found or already finished")
	}

	var existing models.Participant
	if err := s.db.Where("session_id = ? AND telegram_id = ?", session.ID, telegramID).
		First(&existing).Error; err == nil {
		return &JoinResult{SessionID: session.ID, Participant: existing, IsRejoin: true}, nil
	}

	if session.Status != models.SessionStatusWaiting && session.Status != models.SessionStatusQuestion {
		return nil, errors.New("session is not accepting new participants")
	}

	participant := models.Participant{
		SessionID:  session.ID,
		TelegramID: telegramID,
		Nickname:   nickname,
		TotalScore: 0,
		JoinedAt:   time.Now(),
	}
	if err := s.db.Create(&participant).Error; err != nil {
		return nil, fmt.Errorf("failed to join session: %w", err)
	}

	return &JoinResult{SessionID: session.ID, Participant: participant}, nil
}

func (s *SessionService) SubmitAnswer(sessionID uint, telegramID int64, optionID uint) error {
	var session models.Session
	if err := s.db.First(&session, sessionID).Error; err != nil {
		return errors.New("session not found")
	}

	if session.Status != models.SessionStatusQuestion {
		return errors.New("session is not accepting answers")
	}

	var participant models.Participant
	if err := s.db.Where("session_id = ? AND telegram_id = ?", sessionID, telegramID).
		First(&participant).Error; err != nil {
		return errors.New("participant not found in session")
	}

	questions := s.getOrderedQuestions(session.QuizID)
	if session.CurrentQuestion < 1 || session.CurrentQuestion > len(questions) {
		return errors.New("invalid question state")
	}
	currentQ := questions[session.CurrentQuestion-1].Question

	var option models.Option
	if err := s.db.Where("id = ? AND question_id = ?", optionID, currentQ.ID).
		First(&option).Error; err != nil {
		return errors.New("invalid option for current question")
	}

	var existingAnswer models.Answer
	if err := s.db.Where("session_id = ? AND participant_id = ? AND question_id = ?",
		sessionID, participant.ID, currentQ.ID).First(&existingAnswer).Error; err == nil {
		existingAnswer.OptionID = optionID
		existingAnswer.IsCorrect = option.IsCorrect
		existingAnswer.AnsweredAt = time.Now()
		return s.db.Save(&existingAnswer).Error
	}

	answer := models.Answer{
		SessionID:     sessionID,
		ParticipantID: participant.ID,
		QuestionID:    currentQ.ID,
		OptionID:      optionID,
		IsCorrect:     option.IsCorrect,
		Score:         0,
		AnsweredAt:    time.Now(),
	}
	return s.db.Create(&answer).Error
}

func (s *SessionService) GetParticipantResult(sessionID uint, telegramID int64) (*ParticipantResult, error) {
	var session models.Session
	if err := s.db.First(&session, sessionID).Error; err != nil {
		return nil, errors.New("session not found")
	}

	var participant models.Participant
	if err := s.db.Where("session_id = ? AND telegram_id = ?", sessionID, telegramID).
		First(&participant).Error; err != nil {
		return nil, errors.New("participant not found")
	}

	if session.Status != models.SessionStatusRevealed && session.Status != models.SessionStatusFinished {
		return &ParticipantResult{TotalScore: participant.TotalScore, Answered: false}, nil
	}

	questions := s.getOrderedQuestions(session.QuizID)
	if session.CurrentQuestion < 1 || session.CurrentQuestion > len(questions) {
		return nil, errors.New("invalid question state")
	}
	currentQ := questions[session.CurrentQuestion-1].Question

	var answer models.Answer
	if err := s.db.Where("session_id = ? AND participant_id = ? AND question_id = ?",
		sessionID, participant.ID, currentQ.ID).First(&answer).Error; err != nil {
		return &ParticipantResult{TotalScore: participant.TotalScore, Answered: false}, nil
	}

	return &ParticipantResult{
		QuestionID: currentQ.ID,
		OptionID:   answer.OptionID,
		IsCorrect:  answer.IsCorrect,
		Score:      answer.Score,
		TotalScore: participant.TotalScore,
		Answered:   true,
	}, nil
}
