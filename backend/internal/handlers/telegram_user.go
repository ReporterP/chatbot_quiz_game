package handlers

import (
	"net/http"
	"strconv"

	"quiz-game-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type TelegramUserHandler struct {
	tgService *services.TelegramUserService
}

func NewTelegramUserHandler(tgService *services.TelegramUserService) *TelegramUserHandler {
	return &TelegramUserHandler{tgService: tgService}
}

// GetOrCreateUser godoc
// @Summary      Get or create telegram user
// @Tags         telegram
// @Accept       json
// @Produce      json
// @Param        X-Bot-API-Key header string true "Bot API Key"
// @Success      200 {object} map[string]interface{}
// @Router       /api/v1/telegram-users [post]
func (h *TelegramUserHandler) GetOrCreateUser(c *gin.Context) {
	var req struct {
		TelegramID int64  `json:"telegram_id" binding:"required"`
		HostID     uint   `json:"host_id" binding:"required"`
		Nickname   string `json:"nickname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if req.Nickname == "" {
		req.Nickname = "Player"
	}

	user, created, err := h.tgService.GetOrCreate(req.TelegramID, req.HostID, req.Nickname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":    user,
		"created": created,
	})
}

// UpdateNickname godoc
// @Summary      Update telegram user nickname
// @Tags         telegram
// @Accept       json
// @Produce      json
// @Param        X-Bot-API-Key header string true "Bot API Key"
// @Param        telegram_id path int true "Telegram ID"
// @Success      200 {object} map[string]interface{}
// @Router       /api/v1/telegram-users/{telegram_id}/nickname [put]
func (h *TelegramUserHandler) UpdateNickname(c *gin.Context) {
	tgID, err := strconv.ParseInt(c.Param("telegram_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid telegram_id"})
		return
	}

	var req struct {
		HostID   uint   `json:"host_id" binding:"required"`
		Nickname string `json:"nickname" binding:"required,min=1,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	user, err := h.tgService.UpdateNickname(tgID, req.HostID, req.Nickname)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetHistory godoc
// @Summary      Get game history for a telegram user
// @Tags         telegram
// @Produce      json
// @Param        X-Bot-API-Key header string true "Bot API Key"
// @Param        telegram_id path int true "Telegram ID"
// @Param        host_id query int true "Host ID"
// @Success      200 {array} services.GameHistoryEntry
// @Router       /api/v1/telegram-users/{telegram_id}/history [get]
func (h *TelegramUserHandler) GetHistory(c *gin.Context) {
	tgID, err := strconv.ParseInt(c.Param("telegram_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid telegram_id"})
		return
	}

	hostID, err := strconv.ParseUint(c.Query("host_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "host_id is required"})
		return
	}

	entries, err := h.tgService.GetHistory(tgID, uint(hostID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	if entries == nil {
		entries = []services.GameHistoryEntry{}
	}

	c.JSON(http.StatusOK, entries)
}
