package handlers

import (
	"net/http"
	"strconv"

	"quiz-game-backend/internal/services"
	"quiz-game-backend/internal/ws"

	"github.com/gin-gonic/gin"
)

type RoomHandler struct {
	roomService    *services.RoomService
	sessionService *services.SessionService
	hub            *ws.Hub
}

func NewRoomHandler(roomService *services.RoomService, sessionService *services.SessionService, hub *ws.Hub) *RoomHandler {
	return &RoomHandler{roomService: roomService, sessionService: sessionService, hub: hub}
}

type CreateRoomRequest struct {
	Mode string `json:"mode" example:"web"`
}

type StartQuizInRoomRequest struct {
	QuizID uint `json:"quiz_id" binding:"required"`
}

func (h *RoomHandler) CreateRoom(c *gin.Context) {
	hostID := c.GetUint("host_id")
	var req CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Mode = "web"
	}
	room, err := h.roomService.CreateRoom(hostID, req.Mode)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, room)
}

func (h *RoomHandler) GetRoom(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid room id"})
		return
	}
	room, err := h.roomService.GetRoom(uint(roomID))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	currentSession, _ := h.roomService.GetCurrentSession(room.ID)
	var sessionState *services.SessionState
	if currentSession != nil {
		sessionState, _ = h.sessionService.GetSession(currentSession.ID)
	}

	if sessionState == nil {
		latestSession, _ := h.roomService.GetLatestSession(room.ID)
		if latestSession != nil {
			sessionState, _ = h.sessionService.GetSession(latestSession.ID)
		}
	}

	pastSessions, _ := h.roomService.GetRoomSessions(room.ID)

	c.JSON(http.StatusOK, gin.H{
		"room":            room,
		"current_session": sessionState,
		"past_sessions":   pastSessions,
	})
}

func (h *RoomHandler) ListActiveRooms(c *gin.Context) {
	hostID := c.GetUint("host_id")
	rooms, err := h.roomService.GetActiveRooms(hostID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, rooms)
}

func (h *RoomHandler) CloseRoom(c *gin.Context) {
	hostID := c.GetUint("host_id")
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid room id"})
		return
	}
	if err := h.roomService.CloseRoom(uint(roomID), hostID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	h.hub.BroadcastToRoom(uint(roomID), ws.WSMessage{Type: "room_closed", Data: nil})

	c.JSON(http.StatusOK, MessageResponse{Message: "room closed"})
}

func (h *RoomHandler) StartQuizInRoom(c *gin.Context) {
	hostID := c.GetUint("host_id")
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid room id"})
		return
	}

	var req StartQuizInRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	session, err := h.sessionService.CreateSessionInRoom(uint(roomID), req.QuizID, hostID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	state, _ := h.sessionService.GetSession(session.ID)

	h.hub.BroadcastToRoom(uint(roomID), ws.WSMessage{
		Type: "quiz_started",
		Data: state,
	})

	c.JSON(http.StatusOK, state)
}

func (h *RoomHandler) SessionReveal(c *gin.Context) {
	hostID := c.GetUint("host_id")
	roomID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	currentSession, _ := h.roomService.GetCurrentSession(uint(roomID))
	if currentSession == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "no active session"})
		return
	}

	state, err := h.sessionService.RevealAnswer(currentSession.ID, hostID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	h.hub.BroadcastToRoom(uint(roomID), ws.WSMessage{Type: "revealed", Data: state})
	h.hub.Broadcast(currentSession.ID, ws.WSMessage{Type: "revealed", Data: state})

	c.JSON(http.StatusOK, state)
}

func (h *RoomHandler) SessionNext(c *gin.Context) {
	hostID := c.GetUint("host_id")
	roomID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	currentSession, _ := h.roomService.GetCurrentSession(uint(roomID))
	if currentSession == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "no active session"})
		return
	}

	state, err := h.sessionService.NextQuestion(currentSession.ID, hostID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	msgType := "question"
	if state.Status == "finished" {
		msgType = "finished"
	}

	h.hub.BroadcastToRoom(uint(roomID), ws.WSMessage{Type: msgType, Data: state})
	h.hub.Broadcast(currentSession.ID, ws.WSMessage{Type: msgType, Data: state})

	c.JSON(http.StatusOK, state)
}

func (h *RoomHandler) SessionFinish(c *gin.Context) {
	hostID := c.GetUint("host_id")
	roomID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	currentSession, _ := h.roomService.GetCurrentSession(uint(roomID))
	if currentSession == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "no active session"})
		return
	}

	state, err := h.sessionService.ForceFinish(currentSession.ID, hostID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	h.hub.BroadcastToRoom(uint(roomID), ws.WSMessage{Type: "finished", Data: state})
	h.hub.Broadcast(currentSession.ID, ws.WSMessage{Type: "finished", Data: state})

	c.JSON(http.StatusOK, state)
}

func (h *RoomHandler) GetRoomLeaderboard(c *gin.Context) {
	roomID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	session, err := h.roomService.GetLatestSession(uint(roomID))
	if err != nil || session == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "no session found"})
		return
	}

	entries, err := h.sessionService.GetLeaderboard(session.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, entries)
}

func (h *RoomHandler) ListRoomHistory(c *gin.Context) {
	hostID := c.GetUint("host_id")
	rooms, err := h.roomService.ListAllRooms(hostID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, rooms)
}
