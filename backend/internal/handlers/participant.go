package handlers

import (
	"net/http"
	"strconv"

	"quiz-game-backend/internal/services"
	"quiz-game-backend/internal/ws"

	"github.com/gin-gonic/gin"
)

type ParticipantHandler struct {
	sessionService *services.SessionService
	hub            *ws.Hub
}

func NewParticipantHandler(sessionService *services.SessionService, hub *ws.Hub) *ParticipantHandler {
	return &ParticipantHandler{sessionService: sessionService, hub: hub}
}

type JoinSessionRequest struct {
	Code       string `json:"code" binding:"required" example:"123456"`
	TelegramID int64  `json:"telegram_id" binding:"required" example:"123456789"`
	Nickname   string `json:"nickname" binding:"required,min=1,max=100" example:"Player1"`
}

type SubmitAnswerRequest struct {
	TelegramID int64 `json:"telegram_id" binding:"required" example:"123456789"`
	OptionID   uint  `json:"option_id" binding:"required" example:"1"`
}

// JoinSession godoc
// @Summary      Join a quiz session
// @Description  Join a session by code with telegram ID and nickname
// @Tags         participants
// @Accept       json
// @Produce      json
// @Param        X-Bot-API-Key header string true "Bot API Key"
// @Param        request body JoinSessionRequest true "Join data"
// @Success      200 {object} services.JoinResult
// @Failure      400 {object} ErrorResponse
// @Router       /api/v1/sessions/join [post]
func (h *ParticipantHandler) JoinSession(c *gin.Context) {
	var req JoinSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	result, err := h.sessionService.JoinSession(req.Code, req.TelegramID, req.Nickname)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	h.hub.Broadcast(result.SessionID, ws.WSMessage{
		Type: "participant_joined",
		Data: result.Participant,
	})

	c.JSON(http.StatusOK, result)
}

// SubmitAnswer godoc
// @Summary      Submit an answer
// @Description  Submit an answer to the current question
// @Tags         participants
// @Accept       json
// @Produce      json
// @Param        X-Bot-API-Key header string true "Bot API Key"
// @Param        id path int true "Session ID"
// @Param        request body SubmitAnswerRequest true "Answer data"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse
// @Router       /api/v1/sessions/{id}/answer [post]
func (h *ParticipantHandler) SubmitAnswer(c *gin.Context) {
	sessionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid session id"})
		return
	}

	var req SubmitAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.sessionService.SubmitAnswer(uint(sessionID), req.TelegramID, req.OptionID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	h.hub.Broadcast(uint(sessionID), ws.WSMessage{
		Type: "answer_received",
		Data: gin.H{"session_id": sessionID},
	})

	c.JSON(http.StatusOK, MessageResponse{Message: "answer accepted"})
}

// GetMyResult godoc
// @Summary      Get participant result for current question
// @Description  Get whether the participant answered correctly and their score
// @Tags         participants
// @Produce      json
// @Param        X-Bot-API-Key header string true "Bot API Key"
// @Param        id path int true "Session ID"
// @Param        telegram_id query int true "Telegram ID"
// @Success      200 {object} services.ParticipantResult
// @Failure      400 {object} ErrorResponse
// @Router       /api/v1/sessions/{id}/my-result [get]
func (h *ParticipantHandler) GetMyResult(c *gin.Context) {
	sessionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid session id"})
		return
	}

	telegramIDStr := c.Query("telegram_id")
	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid telegram_id"})
		return
	}

	result, err := h.sessionService.GetParticipantResult(uint(sessionID), telegramID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
