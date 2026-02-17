package handlers

import (
	"net/http"
	"strconv"

	"quiz-game-backend/internal/models"
	"quiz-game-backend/internal/services"
	"quiz-game-backend/internal/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SessionHandler struct {
	sessionService *services.SessionService
	hub            *ws.Hub
	db             *gorm.DB
}

func NewSessionHandler(sessionService *services.SessionService, hub *ws.Hub, db *gorm.DB) *SessionHandler {
	return &SessionHandler{sessionService: sessionService, hub: hub, db: db}
}

type CreateSessionRequest struct {
	QuizID uint `json:"quiz_id" binding:"required" example:"1"`
}

// CreateSession godoc
// @Summary      Create a quiz session
// @Description  Start a new quiz session, generates a join code
// @Tags         sessions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateSessionRequest true "Session data"
// @Success      201 {object} services.SessionState
// @Failure      400 {object} ErrorResponse
// @Router       /api/v1/sessions [post]
func (h *SessionHandler) CreateSession(c *gin.Context) {
	hostID := c.GetUint("host_id")

	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	session, err := h.sessionService.CreateSession(req.QuizID, hostID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	state, _ := h.sessionService.GetSession(session.ID)

	var host models.Host
	h.db.First(&host, hostID)
	resp := gin.H{
		"session":  state,
		"bot_link": host.BotLink,
	}

	c.JSON(http.StatusCreated, resp)
}

// ListSessions godoc
// @Summary      List host sessions
// @Description  Get all sessions for the authenticated host
// @Tags         sessions
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} services.SessionSummary
// @Router       /api/v1/sessions [get]
func (h *SessionHandler) ListSessions(c *gin.Context) {
	hostID := c.GetUint("host_id")

	sessions, err := h.sessionService.ListSessions(hostID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, sessions)
}

// GetSession godoc
// @Summary      Get session state
// @Description  Get current state of a quiz session including current question
// @Tags         sessions
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Session ID"
// @Success      200 {object} services.SessionState
// @Failure      404 {object} ErrorResponse
// @Router       /api/v1/sessions/{id} [get]
func (h *SessionHandler) GetSession(c *gin.Context) {
	sessionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid session id"})
		return
	}

	state, err := h.sessionService.GetSession(uint(sessionID))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, state)
}

// RevealAnswer godoc
// @Summary      Reveal the correct answer
// @Description  Show correct answer and calculate scores for current question
// @Tags         sessions
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Session ID"
// @Success      200 {object} services.SessionState
// @Failure      400 {object} ErrorResponse
// @Router       /api/v1/sessions/{id}/reveal [post]
func (h *SessionHandler) RevealAnswer(c *gin.Context) {
	hostID := c.GetUint("host_id")
	sessionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid session id"})
		return
	}

	state, err := h.sessionService.RevealAnswer(uint(sessionID), hostID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	h.hub.Broadcast(uint(sessionID), ws.WSMessage{
		Type: "revealed",
		Data: state,
	})

	c.JSON(http.StatusOK, state)
}

// NextQuestion godoc
// @Summary      Move to next question
// @Description  Start quiz or advance to the next question. Finishes quiz if no more questions.
// @Tags         sessions
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Session ID"
// @Success      200 {object} services.SessionState
// @Failure      400 {object} ErrorResponse
// @Router       /api/v1/sessions/{id}/next [post]
func (h *SessionHandler) NextQuestion(c *gin.Context) {
	hostID := c.GetUint("host_id")
	sessionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid session id"})
		return
	}

	state, err := h.sessionService.NextQuestion(uint(sessionID), hostID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	msgType := "question"
	if state.Status == "finished" {
		msgType = "finished"
	}

	h.hub.Broadcast(uint(sessionID), ws.WSMessage{
		Type: msgType,
		Data: state,
	})

	c.JSON(http.StatusOK, state)
}

// GetLeaderboard godoc
// @Summary      Get leaderboard
// @Description  Get session leaderboard sorted by score
// @Tags         sessions
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Session ID"
// @Success      200 {array} services.LeaderboardEntry
// @Failure      404 {object} ErrorResponse
// @Router       /api/v1/sessions/{id}/leaderboard [get]
func (h *SessionHandler) GetLeaderboard(c *gin.Context) {
	sessionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid session id"})
		return
	}

	entries, err := h.sessionService.GetLeaderboard(uint(sessionID))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, entries)
}
