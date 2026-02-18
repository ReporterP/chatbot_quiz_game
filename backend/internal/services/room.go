package services

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"quiz-game-backend/internal/models"

	"gorm.io/gorm"
)

type RoomService struct {
	db *gorm.DB
}

func NewRoomService(db *gorm.DB) *RoomService {
	return &RoomService{db: db}
}

type RoomWithMembers struct {
	models.Room
	Members []models.RoomMember `json:"members"`
}

func (s *RoomService) loadMembers(room *models.Room) *RoomWithMembers {
	var members []models.RoomMember
	s.db.Where("room_id = ?", room.ID).Order("joined_at ASC").Find(&members)
	return &RoomWithMembers{Room: *room, Members: members}
}

func (s *RoomService) CreateRoom(hostID uint, mode string) (*models.Room, error) {
	if mode != models.RoomModeWeb && mode != models.RoomModeBot {
		mode = models.RoomModeWeb
	}
	code := s.generateUniqueCode()
	room := models.Room{
		HostID: hostID,
		Code:   code,
		Mode:   mode,
		Status: models.RoomStatusActive,
	}
	if err := s.db.Create(&room).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

func (s *RoomService) GetRoom(roomID uint) (*RoomWithMembers, error) {
	var room models.Room
	if err := s.db.First(&room, roomID).Error; err != nil {
		return nil, errors.New("room not found")
	}
	return s.loadMembers(&room), nil
}

func (s *RoomService) GetRoomByCode(code string) (*RoomWithMembers, error) {
	var room models.Room
	if err := s.db.Where("code = ? AND status = ?", code, models.RoomStatusActive).
		First(&room).Error; err != nil {
		return nil, errors.New("room not found or closed")
	}
	return s.loadMembers(&room), nil
}

func (s *RoomService) GetActiveRooms(hostID uint) ([]RoomWithMembers, error) {
	var rooms []models.Room
	if err := s.db.Where("host_id = ? AND status = ?", hostID, models.RoomStatusActive).
		Order("created_at DESC").
		Find(&rooms).Error; err != nil {
		return nil, err
	}
	result := make([]RoomWithMembers, len(rooms))
	for i := range rooms {
		result[i] = *s.loadMembers(&rooms[i])
	}
	return result, nil
}

func (s *RoomService) GetCurrentSession(roomID uint) (*models.Session, error) {
	var session models.Session
	if err := s.db.Where("room_id = ? AND status != ?", roomID, models.SessionStatusFinished).
		Order("created_at DESC").
		First(&session).Error; err != nil {
		return nil, nil
	}
	return &session, nil
}

func (s *RoomService) JoinRoom(code, nickname, webToken string, telegramID int64) (*RoomJoinResult, error) {
	room, err := s.GetRoomByCode(code)
	if err != nil {
		return nil, err
	}

	var existing models.RoomMember
	if webToken != "" {
		if err := s.db.Where("room_id = ? AND web_token = ?", room.ID, webToken).
			First(&existing).Error; err == nil {
			if nickname != "" && nickname != existing.Nickname {
				existing.Nickname = nickname
				s.db.Save(&existing)
			}
			return &RoomJoinResult{Room: room.Room, Member: existing, IsRejoin: true}, nil
		}
	}
	if telegramID > 0 {
		if err := s.db.Where("room_id = ? AND telegram_id = ?", room.ID, telegramID).
			First(&existing).Error; err == nil {
			if nickname != "" && nickname != existing.Nickname {
				existing.Nickname = nickname
				s.db.Save(&existing)
			}
			return &RoomJoinResult{Room: room.Room, Member: existing, IsRejoin: true}, nil
		}
	}

	member := models.RoomMember{
		RoomID:     room.ID,
		Nickname:   nickname,
		TelegramID: telegramID,
		WebToken:   webToken,
		JoinedAt:   time.Now(),
	}
	if err := s.db.Create(&member).Error; err != nil {
		return nil, fmt.Errorf("failed to join room: %w", err)
	}

	return &RoomJoinResult{Room: room.Room, Member: member}, nil
}

func (s *RoomService) Reconnect(webToken, code string) (*RoomJoinResult, error) {
	room, err := s.GetRoomByCode(code)
	if err != nil {
		return nil, err
	}

	var member models.RoomMember
	if err := s.db.Where("room_id = ? AND web_token = ?", room.ID, webToken).
		First(&member).Error; err != nil {
		return nil, errors.New("member not found")
	}

	return &RoomJoinResult{Room: room.Room, Member: member, IsRejoin: true}, nil
}

func (s *RoomService) UpdateNickname(memberID uint, webToken, nickname string) (*models.RoomMember, error) {
	var member models.RoomMember
	if err := s.db.First(&member, memberID).Error; err != nil {
		return nil, errors.New("member not found")
	}
	if member.WebToken != webToken {
		return nil, errors.New("unauthorized")
	}
	member.Nickname = nickname
	s.db.Save(&member)
	return &member, nil
}

func (s *RoomService) RemoveMember(memberID uint) error {
	return s.db.Delete(&models.RoomMember{}, memberID).Error
}

func (s *RoomService) CloseRoom(roomID, hostID uint) error {
	var room models.Room
	if err := s.db.Where("id = ? AND host_id = ?", roomID, hostID).First(&room).Error; err != nil {
		return errors.New("room not found")
	}
	room.Status = models.RoomStatusClosed
	s.db.Save(&room)

	s.db.Model(&models.Session{}).
		Where("room_id = ? AND status != ?", roomID, models.SessionStatusFinished).
		Update("status", models.SessionStatusFinished)

	return nil
}

func (s *RoomService) ListMembers(roomID uint) ([]models.RoomMember, error) {
	var members []models.RoomMember
	s.db.Where("room_id = ?", roomID).Order("joined_at ASC").Find(&members)
	return members, nil
}

func (s *RoomService) GetMemberByToken(roomID uint, webToken string) (*models.RoomMember, error) {
	var member models.RoomMember
	if err := s.db.Where("room_id = ? AND web_token = ?", roomID, webToken).
		First(&member).Error; err != nil {
		return nil, errors.New("member not found")
	}
	return &member, nil
}

func (s *RoomService) GetMemberByTelegramID(roomID uint, telegramID int64) (*models.RoomMember, error) {
	var member models.RoomMember
	if err := s.db.Where("room_id = ? AND telegram_id = ?", roomID, telegramID).
		First(&member).Error; err != nil {
		return nil, errors.New("member not found")
	}
	return &member, nil
}

func (s *RoomService) generateUniqueCode() string {
	for {
		code := fmt.Sprintf("%06d", rand.Intn(1000000))
		var count int64
		s.db.Model(&models.Room{}).
			Where("code = ? AND status = ?", code, models.RoomStatusActive).
			Count(&count)
		if count == 0 {
			return code
		}
	}
}

func (s *RoomService) GetRoomSessions(roomID uint) ([]RoomSessionEntry, error) {
	var sessions []models.Session
	if err := s.db.Where("room_id = ?", roomID).
		Order("created_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, err
	}
	var result []RoomSessionEntry
	for _, sess := range sessions {
		var pCount int64
		s.db.Model(&models.Participant{}).Where("session_id = ?", sess.ID).Count(&pCount)
		var quiz models.Quiz
		s.db.First(&quiz, sess.QuizID)
		result = append(result, RoomSessionEntry{
			ID:               sess.ID,
			QuizTitle:        quiz.Title,
			Status:           sess.Status,
			ParticipantCount: int(pCount),
			CreatedAt:        sess.CreatedAt,
		})
	}
	return result, nil
}

func (s *RoomService) GetLatestSession(roomID uint) (*models.Session, error) {
	var session models.Session
	if err := s.db.Where("room_id = ?", roomID).
		Order("created_at DESC").
		First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *RoomService) ListAllRooms(hostID uint) ([]RoomHistoryEntry, error) {
	var rooms []models.Room
	if err := s.db.Where("host_id = ?", hostID).
		Order("created_at DESC").
		Find(&rooms).Error; err != nil {
		return nil, err
	}

	var result []RoomHistoryEntry
	for _, r := range rooms {
		var sessions []models.Session
		s.db.Where("room_id = ?", r.ID).Order("created_at ASC").Find(&sessions)

		var sessionEntries []RoomSessionEntry
		for _, sess := range sessions {
			var pCount int64
			s.db.Model(&models.Participant{}).Where("session_id = ?", sess.ID).Count(&pCount)

			var quiz models.Quiz
			s.db.First(&quiz, sess.QuizID)

			sessionEntries = append(sessionEntries, RoomSessionEntry{
				ID:               sess.ID,
				QuizTitle:        quiz.Title,
				Status:           sess.Status,
				ParticipantCount: int(pCount),
				CreatedAt:        sess.CreatedAt,
			})
		}

		var memberCount int64
		s.db.Model(&models.RoomMember{}).Where("room_id = ?", r.ID).Count(&memberCount)

		result = append(result, RoomHistoryEntry{
			ID:          r.ID,
			Code:        r.Code,
			Mode:        r.Mode,
			Status:      r.Status,
			MemberCount: int(memberCount),
			Sessions:    sessionEntries,
			CreatedAt:   r.CreatedAt,
		})
	}
	return result, nil
}

type RoomJoinResult struct {
	Room     models.Room       `json:"room"`
	Member   models.RoomMember `json:"member"`
	IsRejoin bool              `json:"is_rejoin"`
}

type RoomHistoryEntry struct {
	ID          uint               `json:"id"`
	Code        string             `json:"code"`
	Mode        string             `json:"mode"`
	Status      string             `json:"status"`
	MemberCount int                `json:"member_count"`
	Sessions    []RoomSessionEntry `json:"sessions"`
	CreatedAt   time.Time          `json:"created_at"`
}

type RoomSessionEntry struct {
	ID               uint      `json:"id"`
	QuizTitle        string    `json:"quiz_title"`
	Status           string    `json:"status"`
	ParticipantCount int       `json:"participant_count"`
	CreatedAt        time.Time `json:"created_at"`
}
