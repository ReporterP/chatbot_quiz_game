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

func (s *RoomService) GetRoom(roomID uint) (*models.Room, error) {
	var room models.Room
	if err := s.db.Preload("Members").First(&room, roomID).Error; err != nil {
		return nil, errors.New("room not found")
	}
	return &room, nil
}

func (s *RoomService) GetRoomByCode(code string) (*models.Room, error) {
	var room models.Room
	if err := s.db.Where("code = ? AND status = ?", code, models.RoomStatusActive).
		Preload("Members").First(&room).Error; err != nil {
		return nil, errors.New("room not found or closed")
	}
	return &room, nil
}

func (s *RoomService) GetActiveRooms(hostID uint) ([]models.Room, error) {
	var rooms []models.Room
	if err := s.db.Where("host_id = ? AND status = ?", hostID, models.RoomStatusActive).
		Preload("Members").
		Order("created_at DESC").
		Find(&rooms).Error; err != nil {
		return nil, err
	}
	return rooms, nil
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
			return &RoomJoinResult{Room: *room, Member: existing, IsRejoin: true}, nil
		}
	}
	if telegramID > 0 {
		if err := s.db.Where("room_id = ? AND telegram_id = ?", room.ID, telegramID).
			First(&existing).Error; err == nil {
			if nickname != "" && nickname != existing.Nickname {
				existing.Nickname = nickname
				s.db.Save(&existing)
			}
			return &RoomJoinResult{Room: *room, Member: existing, IsRejoin: true}, nil
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

	return &RoomJoinResult{Room: *room, Member: member}, nil
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

	return &RoomJoinResult{Room: *room, Member: member, IsRejoin: true}, nil
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

type RoomJoinResult struct {
	Room     models.Room       `json:"room"`
	Member   models.RoomMember `json:"member"`
	IsRejoin bool              `json:"is_rejoin"`
}

type RoomState struct {
	Room           models.Room         `json:"room"`
	Members        []models.RoomMember `json:"members"`
	CurrentSession *SessionState       `json:"current_session,omitempty"`
}
