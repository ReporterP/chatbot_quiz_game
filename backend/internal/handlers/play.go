package handlers

import (
	"log"
	"net/http"
	"strconv"

	"quiz-game-backend/internal/services"
	"quiz-game-backend/internal/ws"

	"github.com/gin-gonic/gin"
)

type PlayHandler struct {
	roomService    *services.RoomService
	sessionService *services.SessionService
	hub            *ws.Hub
}

func NewPlayHandler(roomService *services.RoomService, sessionService *services.SessionService, hub *ws.Hub) *PlayHandler {
	return &PlayHandler{roomService: roomService, sessionService: sessionService, hub: hub}
}

type PlayJoinRequest struct {
	Code     string `json:"code" binding:"required"`
	Nickname string `json:"nickname" binding:"required,min=1,max=100"`
	Token    string `json:"token" binding:"required"`
}

type PlayAnswerRequest struct {
	SessionID uint   `json:"session_id" binding:"required"`
	MemberID  uint   `json:"member_id" binding:"required"`
	Token     string `json:"token" binding:"required"`
	OptionID  uint   `json:"option_id" binding:"required"`
}

type PlayNicknameRequest struct {
	Token    string `json:"token" binding:"required"`
	RoomCode string `json:"room_code" binding:"required"`
	Nickname string `json:"nickname" binding:"required,min=1,max=100"`
}

func (h *PlayHandler) Join(c *gin.Context) {
	var req PlayJoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	result, err := h.roomService.JoinRoom(req.Code, req.Nickname, req.Token, 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	currentSession, _ := h.roomService.GetCurrentSession(result.Room.ID)
	if currentSession != nil && !result.IsRejoin {
		h.sessionService.AddLateParticipant(currentSession.ID, result.Member.ID, result.Member.Nickname)
	}

	if !result.IsRejoin {
		h.hub.BroadcastToRoom(result.Room.ID, ws.WSMessage{
			Type: "member_joined",
			Data: result.Member,
		})
	}

	var sessionState *services.SessionState
	if currentSession != nil {
		sessionState, _ = h.sessionService.GetSession(currentSession.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"room":            result.Room,
		"member":          result.Member,
		"is_rejoin":       result.IsRejoin,
		"current_session": sessionState,
	})
}

func (h *PlayHandler) Reconnect(c *gin.Context) {
	token := c.Query("token")
	code := c.Query("code")
	if token == "" || code == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "token and code required"})
		return
	}

	result, err := h.roomService.Reconnect(token, code)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	currentSession, _ := h.roomService.GetCurrentSession(result.Room.ID)
	var sessionState *services.SessionState
	if currentSession != nil {
		sessionState, _ = h.sessionService.GetSession(currentSession.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"room":            result.Room,
		"member":          result.Member,
		"is_rejoin":       true,
		"current_session": sessionState,
	})
}

func (h *PlayHandler) Answer(c *gin.Context) {
	var req PlayAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.sessionService.SubmitAnswerByMember(req.SessionID, req.MemberID, req.OptionID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	var session services.SessionState
	if s, err := h.sessionService.GetSession(req.SessionID); err == nil {
		session = *s
	}

	h.hub.Broadcast(req.SessionID, ws.WSMessage{
		Type: "answer_received",
		Data: gin.H{"session_id": req.SessionID},
	})
	h.hub.BroadcastToRoom(session.RoomID, ws.WSMessage{
		Type: "answer_received",
		Data: gin.H{"session_id": req.SessionID},
	})

	c.JSON(http.StatusOK, MessageResponse{Message: "answer accepted"})
}

func (h *PlayHandler) GetState(c *gin.Context) {
	token := c.Query("token")
	code := c.Query("code")
	if token == "" || code == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "token and code required"})
		return
	}

	room, err := h.roomService.GetRoomByCode(code)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	member, err := h.roomService.GetMemberByToken(room.ID, token)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "member not found"})
		return
	}

	currentSession, _ := h.roomService.GetCurrentSession(room.ID)
	var sessionState *services.SessionState
	var myResult *services.ParticipantResult
	if currentSession != nil {
		sessionState, _ = h.sessionService.GetSession(currentSession.ID)
		myResult, _ = h.sessionService.GetParticipantResultByMember(currentSession.ID, member.ID)
	}

	members, _ := h.roomService.ListMembers(room.ID)

	c.JSON(http.StatusOK, gin.H{
		"room":            room,
		"member":          member,
		"members":         members,
		"current_session": sessionState,
		"my_result":       myResult,
	})
}

func (h *PlayHandler) UpdateNickname(c *gin.Context) {
	var req PlayNicknameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	room, err := h.roomService.GetRoomByCode(req.RoomCode)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	member, err := h.roomService.GetMemberByToken(room.ID, req.Token)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "member not found"})
		return
	}

	updated, err := h.roomService.UpdateNickname(member.ID, req.Token, req.Nickname)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	h.hub.BroadcastToRoom(room.ID, ws.WSMessage{
		Type: "member_updated",
		Data: updated,
	})

	c.JSON(http.StatusOK, updated)
}

func (h *PlayHandler) GetMyResult(c *gin.Context) {
	sessionID, err := strconv.ParseUint(c.Query("session_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid session_id"})
		return
	}
	memberID, err := strconv.ParseUint(c.Query("member_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid member_id"})
		return
	}

	result, err := h.sessionService.GetParticipantResultByMember(uint(sessionID), uint(memberID))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *PlayHandler) HandleRoomWebSocket(c *gin.Context) {
	code := c.Param("code")
	room, err := h.roomService.GetRoomByCode(code)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "room not found"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}

	h.hub.AddRoomConnection(room.ID, conn)
	defer h.hub.RemoveRoomConnection(room.ID, conn)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
