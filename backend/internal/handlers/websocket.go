package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"smart-live-chats/internal/broadcast"
	"smart-live-chats/internal/config"
	"smart-live-chats/internal/models"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type WebSocketMessage = broadcast.Message

type WebSocketHandler struct {
	upgrader websocket.Upgrader
}

func NewWebSocketHandler(cfg *config.Config) *WebSocketHandler {
	allowedOrigins := map[string]bool{cfg.FrontendURL: true}
	for _, o := range cfg.CORSOrigins {
		allowedOrigins[o] = true
	}

	return &WebSocketHandler{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				return allowedOrigins[origin]
			},
		},
	}
}

func (h *WebSocketHandler) HandleWebSocket(c echo.Context) error {
	conn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return err
	}
	defer conn.Close()

	userID := uint(0)
	if id := c.QueryParam("user_id"); id != "" {
		if parsed, err := strconv.ParseUint(id, 10, 32); err == nil {
			userID = uint(parsed)
		}
	}

	broadcast.RegisterConn(conn, userID)
	log.Printf("WebSocket client connected. Total clients: %d", len(broadcast.Clients))

	defer func() {
		broadcast.UnregisterConn(conn)
		log.Printf("WebSocket client disconnected. Total clients: %d", len(broadcast.Clients))
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var wsMsg broadcast.Message
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("Failed to parse WebSocket message: %v", err)
			continue
		}

		h.handleMessage(conn, &wsMsg)
	}

	return nil
}

func (h *WebSocketHandler) handleMessage(conn *websocket.Conn, msg *broadcast.Message) {
	switch msg.Type {
	case "subscribe":
		broadcast.SubsMutex.Lock()
		if broadcast.SessionSubs[msg.SessionID] == nil {
			broadcast.SessionSubs[msg.SessionID] = make(map[*websocket.Conn]bool)
		}
		broadcast.SessionSubs[msg.SessionID][conn] = true
		broadcast.SubsMutex.Unlock()

	case "unsubscribe":
		broadcast.SubsMutex.Lock()
		if subs := broadcast.SessionSubs[msg.SessionID]; subs != nil {
			delete(subs, conn)
		}
		broadcast.SubsMutex.Unlock()

	case "typing":
		broadcast.ToSession(msg.SessionID, &broadcast.Message{
			Type:      "typing",
			SessionID: msg.SessionID,
			Payload:   msg.Payload,
		}, conn)

	case "read":
		broadcast.ToSession(msg.SessionID, &broadcast.Message{
			Type:      "read",
			SessionID: msg.SessionID,
			Payload:   msg.Payload,
		}, conn)
	}
}

// Convenience wrappers for backward compatibility
func BroadcastMessage(sessionID uint, message *models.ChatMessage) {
	broadcast.SessionMessage(sessionID, message)
}

func BroadcastToUser(userID uint, msg *broadcast.Message) {
	broadcast.ToUser(userID, msg)
}

func BroadcastToAll(msg *broadcast.Message) {
	broadcast.ToAll(msg)
}
