package broadcast

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type      string      `json:"type"`
	SessionID uint        `json:"session_id,omitempty"`
	Payload   interface{} `json:"payload,omitempty"`
}

var (
	Clients      = make(map[*websocket.Conn]uint) // conn -> user_id
	ClientsMutex sync.RWMutex

	SessionSubs = make(map[uint]map[*websocket.Conn]bool)
	SubsMutex   sync.RWMutex

	// Per-connection write locks to prevent concurrent gorilla/websocket writes
	connMutexes  = make(map[*websocket.Conn]*sync.Mutex)
	connMuMutex  sync.RWMutex
)

func getConnMutex(conn *websocket.Conn) *sync.Mutex {
	connMuMutex.RLock()
	mu, ok := connMutexes[conn]
	connMuMutex.RUnlock()
	if ok {
		return mu
	}
	connMuMutex.Lock()
	mu, ok = connMutexes[conn]
	if !ok {
		mu = &sync.Mutex{}
		connMutexes[conn] = mu
	}
	connMuMutex.Unlock()
	return mu
}

func RegisterConn(conn *websocket.Conn, userID uint) {
	ClientsMutex.Lock()
	Clients[conn] = userID
	ClientsMutex.Unlock()
	getConnMutex(conn)
}

func UnregisterConn(conn *websocket.Conn) {
	ClientsMutex.Lock()
	delete(Clients, conn)
	ClientsMutex.Unlock()

	SubsMutex.Lock()
	for _, subs := range SessionSubs {
		delete(subs, conn)
	}
	SubsMutex.Unlock()

	connMuMutex.Lock()
	delete(connMutexes, conn)
	connMuMutex.Unlock()
}

func safeWrite(conn *websocket.Conn, data []byte) error {
	mu := getConnMutex(conn)
	mu.Lock()
	defer mu.Unlock()
	return conn.WriteMessage(websocket.TextMessage, data)
}

func ToSession(sessionID uint, msg *Message, exclude *websocket.Conn) {
	SubsMutex.RLock()
	subs := SessionSubs[sessionID]
	SubsMutex.RUnlock()

	if subs == nil {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal broadcast message: %v", err)
		return
	}

	for conn := range subs {
		if conn != exclude {
			if err := safeWrite(conn, data); err != nil {
				log.Printf("Failed to send broadcast message: %v", err)
			}
		}
	}
}

func SessionMessage(sessionID uint, payload interface{}) {
	msg := &Message{
		Type:      "message",
		SessionID: sessionID,
		Payload:   payload,
	}

	SubsMutex.RLock()
	subs := SessionSubs[sessionID]
	SubsMutex.RUnlock()

	if subs == nil {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal broadcast message: %v", err)
		return
	}

	for conn := range subs {
		if err := safeWrite(conn, data); err != nil {
			log.Printf("Failed to send broadcast message: %v", err)
		}
	}
}

func ToUser(userID uint, msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal broadcast message: %v", err)
		return
	}

	ClientsMutex.RLock()
	defer ClientsMutex.RUnlock()

	for conn, uid := range Clients {
		if uid == userID {
			if err := safeWrite(conn, data); err != nil {
				log.Printf("Failed to send broadcast message to user %d: %v", userID, err)
			}
		}
	}
}

func ToAll(msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal broadcast message: %v", err)
		return
	}

	ClientsMutex.RLock()
	defer ClientsMutex.RUnlock()

	for conn := range Clients {
		if err := safeWrite(conn, data); err != nil {
			log.Printf("Failed to send broadcast message: %v", err)
		}
	}
}
