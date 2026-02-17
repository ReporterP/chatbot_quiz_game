package telegram

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"quiz-game-backend/internal/models"
	"quiz-game-backend/internal/services"
	"quiz-game-backend/internal/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BotInstance struct {
	Token   string
	Secret  string
	HostID  uint
	Client  *Client
	State   *StateManager
	Tracker *SessionTracker
	Handler *UpdateHandler
}

type BotManager struct {
	db              *gorm.DB
	sessionSvc      *services.SessionService
	tgUserSvc       *services.TelegramUserService
	hub             *ws.Hub
	webhookBaseURL  string
	webhookSecret   string
	pollInterval    time.Duration
	refreshInterval time.Duration

	mu   sync.RWMutex
	bots map[string]*BotInstance // secret -> bot

	stopCh chan struct{}
}

func NewBotManager(
	db *gorm.DB,
	sessionSvc *services.SessionService,
	tgUserSvc *services.TelegramUserService,
	hub *ws.Hub,
	webhookBaseURL string,
	webhookSecret string,
	pollInterval time.Duration,
	refreshInterval time.Duration,
) *BotManager {
	return &BotManager{
		db:              db,
		sessionSvc:      sessionSvc,
		tgUserSvc:       tgUserSvc,
		hub:             hub,
		webhookBaseURL:  webhookBaseURL,
		webhookSecret:   webhookSecret,
		pollInterval:    pollInterval,
		refreshInterval: refreshInterval,
		bots:            make(map[string]*BotInstance),
		stopCh:          make(chan struct{}),
	}
}

func tokenSecret(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h[:16])
}

func (m *BotManager) Start() {
	m.refreshTokens()
	go m.refreshLoop()
	log.Println("[BotManager] started")
}

func (m *BotManager) Stop() {
	close(m.stopCh)
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, bot := range m.bots {
		bot.Tracker.Stop()
		bot.Client.DeleteWebhook()
	}
	m.bots = make(map[string]*BotInstance)
	log.Println("[BotManager] stopped")
}

func (m *BotManager) refreshLoop() {
	ticker := time.NewTicker(m.refreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.refreshTokens()
		}
	}
}

func (m *BotManager) refreshTokens() {
	var hosts []models.Host
	m.db.Where("bot_token != '' AND bot_token IS NOT NULL").Find(&hosts)

	newSecrets := make(map[string]models.Host)
	for _, h := range hosts {
		secret := tokenSecret(h.BotToken)
		newSecrets[secret] = h
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for secret, bot := range m.bots {
		if _, exists := newSecrets[secret]; !exists {
			log.Printf("[BotManager] removing bot for host %d", bot.HostID)
			bot.Tracker.Stop()
			go bot.Client.DeleteWebhook()
			delete(m.bots, secret)
		}
	}

	for secret, host := range newSecrets {
		if _, exists := m.bots[secret]; exists {
			continue
		}

		client := NewClient(host.BotToken)
		stateM := NewStateManager()
		tracker := NewSessionTracker(client, stateM, m.sessionSvc, m.pollInterval)
		handler := NewUpdateHandler(client, stateM, tracker, m.sessionSvc, m.tgUserSvc, m.hub, host.ID)

		bot := &BotInstance{
			Token:   host.BotToken,
			Secret:  secret,
			HostID:  host.ID,
			Client:  client,
			State:   stateM,
			Tracker: tracker,
			Handler: handler,
		}

		webhookURL := fmt.Sprintf("%s/webhook/bot/%s", m.webhookBaseURL, secret)
		if err := client.SetWebhook(webhookURL, m.webhookSecret); err != nil {
			log.Printf("[BotManager] failed to set webhook for host %d: %v", host.ID, err)
			continue
		}

		m.bots[secret] = bot
		log.Printf("[BotManager] registered bot for host %d (webhook: %s)", host.ID, webhookURL)
	}

	log.Printf("[BotManager] active bots: %d", len(m.bots))
}

func (m *BotManager) HandleWebhook(c *gin.Context) {
	secret := c.Param("secret")

	if m.webhookSecret != "" {
		headerSecret := c.GetHeader("X-Telegram-Bot-Api-Secret-Token")
		if headerSecret != m.webhookSecret {
			c.Status(http.StatusUnauthorized)
			return
		}
	}

	m.mu.RLock()
	bot, ok := m.bots[secret]
	m.mu.RUnlock()

	if !ok {
		c.Status(http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	var upd Update
	if err := json.Unmarshal(body, &upd); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	go bot.Handler.Handle(upd)

	c.Status(http.StatusOK)
}
