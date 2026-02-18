package ws

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type Hub struct {
	mu       sync.RWMutex
	sessions map[uint]map[*websocket.Conn]bool
	rooms    map[uint]map[*websocket.Conn]bool
}

func NewHub() *Hub {
	return &Hub{
		sessions: make(map[uint]map[*websocket.Conn]bool),
		rooms:    make(map[uint]map[*websocket.Conn]bool),
	}
}

func (h *Hub) AddConnection(sessionID uint, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.sessions[sessionID] == nil {
		h.sessions[sessionID] = make(map[*websocket.Conn]bool)
	}
	h.sessions[sessionID][conn] = true
	log.Printf("ws: client connected to session %d (total: %d)", sessionID, len(h.sessions[sessionID]))
}

func (h *Hub) RemoveConnection(sessionID uint, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conns, ok := h.sessions[sessionID]; ok {
		delete(conns, conn)
		conn.Close()
		if len(conns) == 0 {
			delete(h.sessions, sessionID)
		}
	}
}

func (h *Hub) AddRoomConnection(roomID uint, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[*websocket.Conn]bool)
	}
	h.rooms[roomID][conn] = true
	log.Printf("ws: client connected to room %d (total: %d)", roomID, len(h.rooms[roomID]))
}

func (h *Hub) RemoveRoomConnection(roomID uint, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conns, ok := h.rooms[roomID]; ok {
		delete(conns, conn)
		conn.Close()
		if len(conns) == 0 {
			delete(h.rooms, roomID)
		}
	}
}

func (h *Hub) Broadcast(sessionID uint, message WSMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	conns, ok := h.sessions[sessionID]
	if !ok {
		return
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("ws: marshal error: %v", err)
		return
	}

	for conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("ws: write error: %v", err)
			conn.Close()
			delete(conns, conn)
		}
	}
}

func (h *Hub) BroadcastToRoom(roomID uint, message WSMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	conns, ok := h.rooms[roomID]
	if !ok {
		return
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("ws: marshal error: %v", err)
		return
	}

	for conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("ws: write error: %v", err)
			conn.Close()
			delete(conns, conn)
		}
	}
}
