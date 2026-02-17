package handlers

import (
	"net/http"

	"quiz-game-backend/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SettingsHandler struct {
	db *gorm.DB
}

func NewSettingsHandler(db *gorm.DB) *SettingsHandler {
	return &SettingsHandler{db: db}
}

type SettingsResponse struct {
	BotToken string `json:"bot_token"`
	BotLink  string `json:"bot_link"`
}

type UpdateSettingsRequest struct {
	BotToken string `json:"bot_token" example:"123456:ABC-DEF"`
	BotLink  string `json:"bot_link" example:"https://t.me/my_quiz_bot"`
}

// GetSettings godoc
// @Summary      Get host settings
// @Description  Get bot token and link settings
// @Tags         settings
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} SettingsResponse
// @Failure      404 {object} ErrorResponse
// @Router       /api/v1/settings [get]
func (h *SettingsHandler) GetSettings(c *gin.Context) {
	hostID := c.GetUint("host_id")

	var host models.Host
	if err := h.db.First(&host, hostID).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "host not found"})
		return
	}

	c.JSON(http.StatusOK, SettingsResponse{
		BotToken: host.BotToken,
		BotLink:  host.BotLink,
	})
}

// UpdateSettings godoc
// @Summary      Update host settings
// @Description  Update bot token and link
// @Tags         settings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body UpdateSettingsRequest true "Settings data"
// @Success      200 {object} SettingsResponse
// @Failure      400 {object} ErrorResponse
// @Router       /api/v1/settings [put]
func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	hostID := c.GetUint("host_id")

	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.db.Model(&models.Host{}).Where("id = ?", hostID).Updates(map[string]interface{}{
		"bot_token": req.BotToken,
		"bot_link":  req.BotLink,
	}).Error; err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SettingsResponse{
		BotToken: req.BotToken,
		BotLink:  req.BotLink,
	})
}

type BotTokenEntry struct {
	HostID   uint   `json:"host_id"`
	BotToken string `json:"bot_token"`
}

// GetBotTokens godoc
// @Summary      Get all bot tokens
// @Description  Internal endpoint for bot service to fetch all registered bot tokens
// @Tags         internal
// @Produce      json
// @Param        X-Bot-API-Key header string true "Bot API Key"
// @Success      200 {array} BotTokenEntry
// @Router       /api/v1/internal/bot-tokens [get]
func (h *SettingsHandler) GetBotTokens(c *gin.Context) {
	var hosts []models.Host
	h.db.Where("bot_token != '' AND bot_token IS NOT NULL").Find(&hosts)

	entries := make([]BotTokenEntry, 0, len(hosts))
	for _, host := range hosts {
		entries = append(entries, BotTokenEntry{
			HostID:   host.ID,
			BotToken: host.BotToken,
		})
	}

	c.JSON(http.StatusOK, entries)
}
