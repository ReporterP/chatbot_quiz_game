package handlers

import (
	"log"
	"net/http"
	"strconv"

	"quiz-game-backend/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WSHandler struct {
	hub *ws.Hub
}

func NewWSHandler(hub *ws.Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// HandleWebSocket godoc
// @Summary      WebSocket connection for session updates
// @Description  Connect via WebSocket to receive real-time session updates
// @Tags         websocket
// @Param        id path int true "Session ID"
// @Router       /ws/session/{id} [get]
func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	sessionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid session id"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}

	sid := uint(sessionID)
	h.hub.AddConnection(sid, conn)
	defer h.hub.RemoveConnection(sid, conn)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
