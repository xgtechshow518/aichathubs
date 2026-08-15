package whatsapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"database/sql"

	"smart-live-chats/internal/broadcast"
	"smart-live-chats/internal/database"
	"smart-live-chats/internal/gemini"
	"smart-live-chats/internal/models"

	qrcode "github.com/skip2/go-qrcode"
	sqlitedriver "modernc.org/sqlite"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

func init() {
	sql.Register("sqlite3", &sqlitedriver.Driver{})
	// Break import cycle: gemini tools call back into WhatsApp for image send.
	gemini.SendProductImageFunc = func(deviceID uint, jid, imageURL, caption string) error {
		if GlobalManager == nil {
			return fmt.Errorf("whatsapp manager not initialized")
		}
		return GlobalManager.SendImageViaDevice(deviceID, jid, imageURL, caption)
	}
}

type Manager struct {
	mu       sync.RWMutex
	clients  map[uint]*clientEntry // deviceID -> client
	pending  map[uint]*clientEntry // temp slot ID -> client (scanning, no DB device yet)
	nextSlot uint
	sqlStore *sqlstore.Container

	// autoReplyMu guards autoReplyInFlight. Only one tryAutoReply goroutine
	// can be active per chat session at a time; the active goroutine loops
	// over any new customer messages that arrive while it is thinking.
	autoReplyMu       sync.Mutex
	autoReplyInFlight map[uint]bool // sessionID -> in-flight
}

type clientEntry struct {
	client   *whatsmeow.Client
	deviceID uint
	userID   uint
	cancel   context.CancelFunc
}

var GlobalManager *Manager

func NewManager(dbPath string) (*Manager, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open whatsapp database: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}
	db.SetMaxOpenConns(1)

	store := sqlstore.NewWithDB(db, "sqlite3", nil)
	if err := store.Upgrade(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to upgrade whatsmeow store: %w", err)
	}

	m := &Manager{
		clients:           make(map[uint]*clientEntry),
		pending:           make(map[uint]*clientEntry),
		nextSlot:          1,
		sqlStore:          store,
		autoReplyInFlight: make(map[uint]bool),
	}
	GlobalManager = m
	return m, nil
}

func (m *Manager) ReconnectAll() {
	var devices []models.WhatsAppDevice
	database.DB.Where("status = ?", "connected").Find(&devices)

	for _, dev := range devices {
		go func(d models.WhatsAppDevice) {
			if err := m.reconnect(d); err != nil {
				log.Printf("Failed to reconnect WhatsApp device %d for user %d: %v", d.ID, d.UserID, err)
				database.DB.Model(&d).Update("status", "disconnected")
			}
		}(dev)
	}
}

func (m *Manager) reconnect(dev models.WhatsAppDevice) error {
	allDevices, err := m.sqlStore.GetAllDevices(context.Background())
	if err != nil {
		return err
	}

	for _, d := range allDevices {
		if d.ID != nil && d.ID.String() == dev.JID {
			client := whatsmeow.NewClient(d, nil)
			ctx, cancel := context.WithCancel(context.Background())

			entry := &clientEntry{
				client:   client,
				deviceID: dev.ID,
				userID:   dev.UserID,
				cancel:   cancel,
			}

			client.AddEventHandler(func(evt interface{}) {
				m.handleEvent(entry, evt)
			})

			if err := client.Connect(); err != nil {
				cancel()
				return err
			}

			m.mu.Lock()
			m.clients[dev.ID] = entry
			m.mu.Unlock()

			go func() { <-ctx.Done() }()

			log.Printf("Reconnected WhatsApp device %d for user %d (JID: %s)", dev.ID, dev.UserID, dev.JID)
			return nil
		}
	}

	return fmt.Errorf("stored session not found for JID %s", dev.JID)
}

// Connect starts a new WhatsApp device connection for the user.
// Returns a temporary slot ID used to track this pending connection.
func (m *Manager) Connect(userID uint) (uint, error) {
	deviceStore := m.sqlStore.NewDevice()
	client := whatsmeow.NewClient(deviceStore, nil)
	ctx, cancel := context.WithCancel(context.Background())

	m.mu.Lock()
	slotID := m.nextSlot
	m.nextSlot++
	m.mu.Unlock()

	entry := &clientEntry{
		client: client,
		userID: userID,
		cancel: cancel,
	}

	client.AddEventHandler(func(evt interface{}) {
		m.handleEvent(entry, evt)
	})

	qrChan, _ := client.GetQRChannel(ctx)

	if err := client.Connect(); err != nil {
		cancel()
		return 0, fmt.Errorf("failed to connect: %w", err)
	}

	m.mu.Lock()
	m.pending[slotID] = entry
	m.mu.Unlock()

	broadcast.ToUser(userID, &broadcast.Message{
		Type:    "whatsapp_status",
		Payload: map[string]string{"status": "scanning"},
	})

	go func() {
		for evt := range qrChan {
			switch evt.Event {
			case "code":
				png, err := qrcode.Encode(evt.Code, qrcode.Medium, 512)
				if err != nil {
					log.Printf("Failed to generate QR code: %v", err)
					continue
				}
				b64 := base64.StdEncoding.EncodeToString(png)
				broadcast.ToUser(userID, &broadcast.Message{
					Type: "whatsapp_qr",
					Payload: map[string]string{
						"qr_image": "data:image/png;base64," + b64,
					},
				})

			case "success":
				log.Printf("WhatsApp login success for user %d", userID)

			case "timeout":
				log.Printf("WhatsApp QR timeout for user %d", userID)
				broadcast.ToUser(userID, &broadcast.Message{
					Type:    "whatsapp_status",
					Payload: map[string]string{"status": "timeout"},
				})
				m.removePending(slotID)
			}
		}
	}()

	go func() { <-ctx.Done() }()

	return slotID, nil
}

func (m *Manager) removePending(slotID uint) {
	m.mu.Lock()
	entry, ok := m.pending[slotID]
	if ok {
		delete(m.pending, slotID)
	}
	m.mu.Unlock()

	if ok {
		entry.client.Disconnect()
		entry.cancel()
	}
}

// DisconnectDevice disconnects a specific device by its DB ID.
func (m *Manager) DisconnectDevice(deviceID uint) {
	m.mu.Lock()
	entry, ok := m.clients[deviceID]
	if ok {
		delete(m.clients, deviceID)
	}
	m.mu.Unlock()

	if !ok {
		return
	}

	entry.client.Disconnect()
	entry.cancel()

	database.DB.Model(&models.WhatsAppDevice{}).Where("id = ?", deviceID).
		Update("status", "disconnected")

	broadcast.ToUser(entry.userID, &broadcast.Message{
		Type:    "whatsapp_status",
		Payload: map[string]interface{}{"status": "disconnected", "device_id": deviceID},
	})

	log.Printf("WhatsApp device %d disconnected for user %d", deviceID, entry.userID)
}

// LogoutDevice disconnects and permanently removes a device.
func (m *Manager) LogoutDevice(deviceID uint) error {
	m.mu.RLock()
	entry, ok := m.clients[deviceID]
	m.mu.RUnlock()

	var userID uint
	if ok {
		userID = entry.userID
		entry.client.Logout(context.Background())
	}

	m.DisconnectDevice(deviceID)

	database.DB.Where("id = ?", deviceID).Delete(&models.WhatsAppDevice{})

	if userID > 0 {
		broadcast.ToUser(userID, &broadcast.Message{
			Type:    "whatsapp_status",
			Payload: map[string]interface{}{"status": "disconnected", "device_id": deviceID},
		})
	}

	return nil
}

// SendMessageViaDevice sends a message through a specific device.
func (m *Manager) SendMessageViaDevice(deviceID uint, jid string, text string) error {
	m.mu.RLock()
	entry, ok := m.clients[deviceID]
	m.mu.RUnlock()

	if !ok || !entry.client.IsConnected() {
		return fmt.Errorf("whatsapp device not connected")
	}

	targetJID, err := types.ParseJID(jid)
	if err != nil {
		return fmt.Errorf("invalid JID: %w", err)
	}

	_, err = entry.client.SendMessage(context.Background(), targetJID, &waE2E.Message{
		Conversation: proto.String(text),
	})
	return err
}

// SendImageViaDevice downloads imageURL and sends it as a WhatsApp image message.
func (m *Manager) SendImageViaDevice(deviceID uint, jid, imageURL, caption string) error {
	m.mu.RLock()
	entry, ok := m.clients[deviceID]
	m.mu.RUnlock()

	if !ok || !entry.client.IsConnected() {
		return fmt.Errorf("whatsapp device not connected")
	}

	targetJID, err := types.ParseJID(jid)
	if err != nil {
		return fmt.Errorf("invalid JID: %w", err)
	}

	resp, err := http.Get(imageURL)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("image download status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read image: %w", err)
	}
	if len(data) == 0 {
		return fmt.Errorf("empty image data")
	}

	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}

	uploaded, err := entry.client.Upload(context.Background(), data, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("whatsapp upload failed: %w", err)
	}

	img := &waE2E.ImageMessage{
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uploaded.FileLength),
		Mimetype:      proto.String(mimeType),
	}
	if caption != "" {
		img.Caption = proto.String(caption)
	}

	_, err = entry.client.SendMessage(context.Background(), targetJID, &waE2E.Message{
		ImageMessage: img,
	})
	return err
}

// IsUserConnected returns whether the user has at least one active WhatsApp connection.
func (m *Manager) IsUserConnected(userID uint) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, entry := range m.clients {
		if entry.userID == userID && entry.client.IsConnected() {
			return true
		}
	}
	return false
}

// IsDeviceConnected returns whether a specific device is connected at runtime.
func (m *Manager) IsDeviceConnected(deviceID uint) bool {
	m.mu.RLock()
	entry, ok := m.clients[deviceID]
	m.mu.RUnlock()
	return ok && entry.client.IsConnected()
}

// ConnectedDeviceCount returns the number of connected devices for a user.
func (m *Manager) ConnectedDeviceCount(userID uint) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, entry := range m.clients {
		if entry.userID == userID && entry.client.IsConnected() {
			count++
		}
	}
	return count
}

func (m *Manager) handleEvent(entry *clientEntry, evt interface{}) {
	switch v := evt.(type) {
	case *events.Connected:
		log.Printf("WhatsApp connected for user %d", entry.userID)
		now := time.Now()
		jid := ""
		pushName := ""
		phone := ""
		if entry.client.Store.ID != nil {
			jid = entry.client.Store.ID.String()
			phone = entry.client.Store.ID.User
		}
		pushName = entry.client.Store.PushName

		// Reject duplicate: if another runtime client is already connected with the same phone
		if phone != "" && entry.deviceID == 0 {
			m.mu.RLock()
			for _, existing := range m.clients {
				if existing.userID == entry.userID && existing.client.IsConnected() &&
					existing.client.Store != nil && existing.client.Store.ID != nil &&
					existing.client.Store.ID.User == phone {
					m.mu.RUnlock()
					log.Printf("Duplicate WhatsApp account +%s rejected for user %d", phone, entry.userID)
					entry.client.Logout(context.Background())
					entry.client.Disconnect()
					entry.cancel()
					m.mu.Lock()
					for slotID, p := range m.pending {
						if p == entry {
							delete(m.pending, slotID)
							break
						}
					}
					m.mu.Unlock()
					broadcast.ToUser(entry.userID, &broadcast.Message{
						Type: "whatsapp_status",
						Payload: map[string]interface{}{
							"status":  "error",
							"message": "This WhatsApp account (+"+phone+") is already connected. Please scan a different account.",
						},
					})
					return
				}
			}
			m.mu.RUnlock()
		}

		var device models.WhatsAppDevice
		if entry.deviceID > 0 {
			if err := database.DB.First(&device, entry.deviceID).Error; err == nil {
				device.JID = jid
				device.PushName = pushName
				device.Phone = phone
				device.Status = "connected"
				device.ConnectedAt = &now
				database.DB.Save(&device)
			}
		} else {
			// Look up any existing DB device row with the same JID for this user,
			// including soft-deleted rows (Unscoped). Reusing the original row
			// keeps its primary key stable, which means every ChatSession /
			// ChatMessage previously anchored to this device via device_id stays
			// correctly linked after a disconnect -> reconnect cycle. Without
			// Unscoped here the old row would be ignored and a new id minted,
			// orphaning the prior chat history.
			result := database.DB.Unscoped().
				Where("user_id = ? AND jid = ?", entry.userID, jid).
				First(&device)
			if result.Error != nil {
				device = models.WhatsAppDevice{
					UserID:      entry.userID,
					JID:         jid,
					PushName:    pushName,
					Phone:       phone,
					Status:      "connected",
					ConnectedAt: &now,
				}
				database.DB.Create(&device)
			} else {
				// Clear deleted_at so the device is "undeleted"; update the
				// live fields. Use Unscoped().Save so GORM doesn't add a
				// `deleted_at IS NULL` filter to the UPDATE.
				device.DeletedAt = gorm.DeletedAt{}
				device.PushName = pushName
				device.Phone = phone
				device.Status = "connected"
				device.ConnectedAt = &now
				database.DB.Unscoped().Save(&device)
				if result.RowsAffected > 0 && device.ID > 0 {
					log.Printf("Restored WhatsApp device %d for user %d (jid=%s)", device.ID, entry.userID, jid)
				}
			}
			entry.deviceID = device.ID

			m.mu.Lock()
			for slotID, p := range m.pending {
				if p == entry {
					delete(m.pending, slotID)
					break
				}
			}
			m.clients[device.ID] = entry
			m.mu.Unlock()
		}

		broadcast.ToUser(entry.userID, &broadcast.Message{
			Type: "whatsapp_status",
			Payload: map[string]interface{}{
				"status":    "connected",
				"device_id": device.ID,
				"jid":       jid,
				"push_name": pushName,
				"phone":     phone,
			},
		})

	case *events.Disconnected:
		log.Printf("WhatsApp disconnected event for user %d device %d", entry.userID, entry.deviceID)

	case *events.Message:
		m.handleIncomingMessage(entry, v)

	case *events.FBMessage:
		m.handleFBMessage(entry, v)

	case *events.HistorySync:
		m.handleHistorySync(entry, v)

	case *events.UndecryptableMessage:
		log.Printf("WhatsApp undecryptable message for user %d device %d (IsFromMe=%v, chat=%s, unavailable=%v)",
			entry.userID, entry.deviceID, v.Info.IsFromMe, v.Info.Chat.String(), v.IsUnavailable)

	case *events.Receipt, *events.ChatPresence, *events.Presence,
		*events.OfflineSyncPreview, *events.OfflineSyncCompleted,
		*events.AppStateSyncComplete, *events.PushName, *events.MarkChatAsRead:
		// Silently ignore common non-message events

	default:
		log.Printf("WhatsApp unhandled event type %T for user %d device %d", evt, entry.userID, entry.deviceID)
	}
}

// handleFBMessage handles messages in the newer Armadillo/FBMessage format.
// WhatsApp increasingly sends messages (especially outgoing syncs from the phone)
// using this format instead of the legacy events.Message format.
func (m *Manager) handleFBMessage(entry *clientEntry, msg *events.FBMessage) {
	content := ""
	msgType := "text"

	ca := msg.GetConsumerApplication()
	if ca != nil {
		payload := ca.GetPayload()
		if payload != nil {
			c := payload.GetContent()
			if c != nil {
				if mt := c.GetMessageText(); mt != nil {
					content = mt.GetText()
				} else if ext := c.GetExtendedTextMessage(); ext != nil {
					if t := ext.GetText(); t != nil {
						content = t.GetText()
					}
				} else if c.GetImageMessage() != nil {
					msgType = "image"
				}
			}
		}
	}

	if content == "" && msgType == "text" {
		log.Printf("FBMessage with no extractable text content from user %d device %d (IsFromMe=%v, chat=%s)",
			entry.userID, entry.deviceID, msg.Info.IsFromMe, msg.Info.Chat.String())
		return
	}

	log.Printf("FBMessage received: user %d device %d, IsFromMe=%v, chat=%s, content=%s",
		entry.userID, entry.deviceID, msg.Info.IsFromMe, msg.Info.Chat.String(), content)

	if msg.Info.IsFromMe {
		m.handleOutgoingSyncFB(entry, msg, content, msgType)
		return
	}

	m.handleIncomingMessageFB(entry, msg, content, msgType)
}

func (m *Manager) handleIncomingMessageFB(entry *clientEntry, msg *events.FBMessage, content, msgType string) {
	senderPhone := msg.Info.Sender.User
	pushName := msg.Info.PushName

	customerName := pushName
	if customerName == "" {
		customerName = senderPhone
	}

	var session models.ChatSession
	result := database.DB.Where("customer_phone = ? AND platform = ? AND device_id = ?",
		senderPhone, "whatsapp", entry.deviceID).First(&session)

	now := time.Now()
	if result.Error != nil {
		session = models.ChatSession{
			CustomerName:   customerName,
			CustomerPhone:  senderPhone,
			Platform:       "whatsapp",
			DeviceID:       &entry.deviceID,
			UnreadCount:    1,
			LastMessage:    content,
			LastMessageAt:  &now,
			LastSenderType: "customer",
			Status:         "active",
		}
		database.DB.Create(&session)
	} else {
		session.UnreadCount++
		session.LastMessage = content
		session.LastMessageAt = &now
		session.LastSenderType = "customer"
		if customerName != "" && session.CustomerName != customerName {
			session.CustomerName = customerName
		}
		database.DB.Save(&session)
	}

	chatMsg := models.ChatMessage{
		SessionID:   session.ID,
		SenderType:  "customer",
		Content:     content,
		MessageType: msgType,
	}
	database.DB.Create(&chatMsg)

	broadcast.SessionMessage(session.ID, &chatMsg)

	broadcast.ToUser(entry.userID, &broadcast.Message{
		Type:      "message",
		SessionID: session.ID,
		Payload:   &chatMsg,
	})

	broadcast.ToAll(&broadcast.Message{
		Type: "session_update",
		Payload: map[string]interface{}{
			"session_id":    session.ID,
			"last_message":  content,
			"unread_count":  session.UnreadCount,
			"customer_name": session.CustomerName,
		},
	})

	log.Printf("FBMessage from %s (%s) to user %d device %d: %s",
		customerName, msg.Info.Sender.String(), entry.userID, entry.deviceID, content)

	if msgType == "text" && content != "" {
		go m.tryAutoReply(entry, session.ID, msg.Info.Chat.String(), content)
	}
}

func (m *Manager) handleOutgoingSyncFB(entry *clientEntry, msg *events.FBMessage, content, msgType string) {
	chatJID := msg.Info.Chat

	if chatJID.Server != "s.whatsapp.net" && chatJID.Server != "lid" {
		return
	}

	recipientPhone := chatJID.User
	if recipientPhone == "" {
		return
	}

	log.Printf("FBMessage outgoing sync: user %d device %d -> %s: %s", entry.userID, entry.deviceID, recipientPhone, content)

	// Try to find session by phone number with either the chat JID user or by partial match
	var session models.ChatSession
	result := database.DB.Where("customer_phone = ? AND platform = ? AND device_id = ?",
		recipientPhone, "whatsapp", entry.deviceID).First(&session)

	if result.Error != nil {
		return
	}

	now := time.Now()
	session.LastMessage = content
	session.LastMessageAt = &now
	session.LastSenderType = "agent"
	database.DB.Save(&session)

	chatMsg := models.ChatMessage{
		SessionID:   session.ID,
		SenderType:  "agent",
		Content:     content,
		MessageType: msgType,
		IsRead:      true,
	}
	if err := database.DB.Create(&chatMsg).Error; err != nil {
		log.Printf("Failed to save FBMessage outgoing sync: %v", err)
		return
	}

	broadcast.SessionMessage(session.ID, &chatMsg)

	broadcast.ToUser(entry.userID, &broadcast.Message{
		Type:      "message",
		SessionID: session.ID,
		Payload:   &chatMsg,
	})

	broadcast.ToAll(&broadcast.Message{
		Type: "session_update",
		Payload: map[string]interface{}{
			"session_id":   session.ID,
			"last_message": content,
		},
	})

	log.Printf("Synced FBMessage outgoing to %s from user %d device %d", recipientPhone, entry.userID, entry.deviceID)
}

// handleHistorySync processes messages from WhatsApp history sync events.
// Recent outgoing messages sent from the phone often arrive this way when
// real-time message events are undecryptable.
func (m *Manager) handleHistorySync(entry *clientEntry, evt *events.HistorySync) {
	if evt.Data == nil {
		return
	}

	// Only process recent sync types that may contain new messages
	cutoff := time.Now().Add(-5 * time.Minute)

	for _, conv := range evt.Data.GetConversations() {
		remoteJID := conv.GetID()
		if remoteJID == "" {
			continue
		}

		// Extract phone number from JID (e.g., "144732411945057@lid" or "1234@s.whatsapp.net")
		phone := strings.Split(remoteJID, "@")[0]
		if phone == "" {
			continue
		}

		for _, hsMsg := range conv.GetMessages() {
			webMsg := hsMsg.GetMessage()
			if webMsg == nil {
				continue
			}

			// Only process recent messages
			ts := time.Unix(int64(webMsg.GetMessageTimestamp()), 0)
			if ts.Before(cutoff) {
				continue
			}

			key := webMsg.GetKey()
			if key == nil {
				continue
			}

			waMsg := webMsg.GetMessage()
			if waMsg == nil {
				continue
			}

			content := ""
			msgType := "text"
			if waMsg.GetConversation() != "" {
				content = waMsg.GetConversation()
			} else if waMsg.GetExtendedTextMessage() != nil {
				content = waMsg.GetExtendedTextMessage().GetText()
			} else if waMsg.GetImageMessage() != nil {
				msgType = "image"
				content = waMsg.GetImageMessage().GetCaption()
			}

			if content == "" && msgType == "text" {
				continue
			}

			isFromMe := key.GetFromMe()

			// Find the matching session
			var session models.ChatSession
			result := database.DB.Where("customer_phone = ? AND platform = ? AND device_id = ?",
				phone, "whatsapp", entry.deviceID).First(&session)
			if result.Error != nil {
				// Also try matching by LID JID if available
				lidJID := conv.GetLidJID()
				if lidJID != "" {
					lidPhone := strings.Split(lidJID, "@")[0]
					result = database.DB.Where("customer_phone = ? AND platform = ? AND device_id = ?",
						lidPhone, "whatsapp", entry.deviceID).First(&session)
				}
				if result.Error != nil {
					continue
				}
			}

			// Check if this message already exists (dedup by content + timestamp within 2s window)
			var existingCount int64
			database.DB.Model(&models.ChatMessage{}).
				Where("session_id = ? AND content = ? AND created_at > ? AND created_at < ?",
					session.ID, content, ts.Add(-2*time.Second), ts.Add(2*time.Second)).
				Count(&existingCount)
			if existingCount > 0 {
				continue
			}

			senderType := "customer"
			if isFromMe {
				senderType = "agent"
			}

			log.Printf("HistorySync message: user %d device %d, fromMe=%v, phone=%s, content=%s",
				entry.userID, entry.deviceID, isFromMe, phone, content)

			now := time.Now()
			session.LastMessage = content
			session.LastMessageAt = &now
			session.LastSenderType = senderType
			if !isFromMe {
				session.UnreadCount++
			}
			database.DB.Save(&session)

			chatMsg := models.ChatMessage{
				SessionID:   session.ID,
				SenderType:  senderType,
				Content:     content,
				MessageType: msgType,
				IsRead:      isFromMe,
			}
			if err := database.DB.Create(&chatMsg).Error; err != nil {
				log.Printf("Failed to save history sync message: %v", err)
				continue
			}

			broadcast.SessionMessage(session.ID, &chatMsg)

			broadcast.ToUser(entry.userID, &broadcast.Message{
				Type:      "message",
				SessionID: session.ID,
				Payload:   &chatMsg,
			})

			broadcast.ToAll(&broadcast.Message{
				Type: "session_update",
				Payload: map[string]interface{}{
					"session_id":    session.ID,
					"last_message":  content,
					"unread_count":  session.UnreadCount,
					"customer_name": session.CustomerName,
				},
			})
		}
	}
}

func (m *Manager) handleIncomingMessage(entry *clientEntry, msg *events.Message) {
	content := ""
	msgType := "text"

	if msg.Message.GetConversation() != "" {
		content = msg.Message.GetConversation()
	} else if msg.Message.GetExtendedTextMessage() != nil {
		content = msg.Message.GetExtendedTextMessage().GetText()
	} else if msg.Message.GetImageMessage() != nil {
		msgType = "image"
		content = msg.Message.GetImageMessage().GetCaption()
	}

	if content == "" && msgType == "text" {
		return
	}

	if msg.Info.IsFromMe {
		m.handleOutgoingSync(entry, msg, content, msgType)
		return
	}

	senderJID := msg.Info.Sender.String()
	senderPhone := msg.Info.Sender.User
	pushName := msg.Info.PushName

	customerName := pushName
	if customerName == "" {
		customerName = senderPhone
	}

	var session models.ChatSession
	result := database.DB.Where("customer_phone = ? AND platform = ? AND device_id = ?",
		senderPhone, "whatsapp", entry.deviceID).First(&session)

	now := time.Now()
	if result.Error != nil {
		session = models.ChatSession{
			CustomerName:   customerName,
			CustomerPhone:  senderPhone,
			Platform:       "whatsapp",
			DeviceID:       &entry.deviceID,
			UnreadCount:    1,
			LastMessage:    content,
			LastMessageAt:  &now,
			LastSenderType: "customer",
			Status:         "active",
		}
		database.DB.Create(&session)
	} else {
		session.UnreadCount++
		session.LastMessage = content
		session.LastMessageAt = &now
		session.LastSenderType = "customer"
		if customerName != "" && session.CustomerName != customerName {
			session.CustomerName = customerName
		}
		database.DB.Save(&session)
	}

	chatMsg := models.ChatMessage{
		SessionID:   session.ID,
		SenderType:  "customer",
		Content:     content,
		MessageType: msgType,
	}
	database.DB.Create(&chatMsg)

	broadcast.SessionMessage(session.ID, &chatMsg)

	broadcast.ToUser(entry.userID, &broadcast.Message{
		Type:      "message",
		SessionID: session.ID,
		Payload:   &chatMsg,
	})

	broadcast.ToAll(&broadcast.Message{
		Type: "session_update",
		Payload: map[string]interface{}{
			"session_id":    session.ID,
			"last_message":  content,
			"unread_count":  session.UnreadCount,
			"customer_name": session.CustomerName,
		},
	})

	log.Printf("WhatsApp message from %s (%s) to user %d device %d: %s",
		customerName, senderJID, entry.userID, entry.deviceID, content)

	if msgType == "text" && content != "" {
		go m.tryAutoReply(entry, session.ID, msg.Info.Chat.String(), content)
	}
}

// handleOutgoingSync syncs messages sent from the user's WhatsApp phone app back to the platform
func (m *Manager) handleOutgoingSync(entry *clientEntry, msg *events.Message, content, msgType string) {
	chatJID := msg.Info.Chat

	// Only sync personal chats (not groups, status broadcasts, etc.)
	if chatJID.Server != "s.whatsapp.net" && chatJID.Server != "lid" {
		return
	}

	recipientPhone := chatJID.User
	if recipientPhone == "" {
		return
	}

	log.Printf("Outgoing sync: user %d device %d -> %s: %s", entry.userID, entry.deviceID, recipientPhone, content)

	var session models.ChatSession
	result := database.DB.Where("customer_phone = ? AND platform = ? AND device_id = ?",
		recipientPhone, "whatsapp", entry.deviceID).First(&session)

	if result.Error != nil {
		return
	}

	now := time.Now()
	session.LastMessage = content
	session.LastMessageAt = &now
	session.LastSenderType = "agent"
	database.DB.Save(&session)

	chatMsg := models.ChatMessage{
		SessionID:   session.ID,
		SenderType:  "agent",
		Content:     content,
		MessageType: msgType,
		IsRead:      true,
	}
	if err := database.DB.Create(&chatMsg).Error; err != nil {
		log.Printf("Failed to save outgoing sync message: %v", err)
		return
	}

	broadcast.SessionMessage(session.ID, &chatMsg)

	// Also broadcast directly to the device owner so the message appears
	// even if the frontend isn't subscribed to this session's channel.
	broadcast.ToUser(entry.userID, &broadcast.Message{
		Type:      "message",
		SessionID: session.ID,
		Payload:   &chatMsg,
	})

	broadcast.ToAll(&broadcast.Message{
		Type: "session_update",
		Payload: map[string]interface{}{
			"session_id":   session.ID,
			"last_message": content,
		},
	})

	log.Printf("Synced outgoing WhatsApp message to %s from user %d device %d", recipientPhone, entry.userID, entry.deviceID)
}

// maxRethinkIterations bounds the re-think loop in tryAutoReply so that a
// customer who keeps typing during the AI delay can't stall the response
// indefinitely. After this many iterations we give up; the next inbound
// customer message will trigger a fresh cycle.
const maxRethinkIterations = 5

// humanRepliedAfter reports whether any agent (human dashboard) message
// exists in this session with id strictly greater than the given customer
// message id. ID is monotonic per-session in our SQLite-backed store, so
// this is a reliable "human took over since the trigger" signal.
func humanRepliedAfter(sessionID, customerMsgID uint) bool {
	var count int64
	database.DB.Model(&models.ChatMessage{}).
		Where("session_id = ? AND sender_type = ? AND id > ?", sessionID, "agent", customerMsgID).
		Count(&count)
	return count > 0
}

func (m *Manager) tryAutoReply(entry *clientEntry, sessionID uint, chatJID string, question string) {
	// Per-session single-flight: at most one tryAutoReply goroutine works on
	// a session at a time. Subsequent customer messages exit immediately;
	// the active goroutine's re-think loop will pick up their content from
	// the DB on its next iteration.
	m.autoReplyMu.Lock()
	if m.autoReplyInFlight[sessionID] {
		m.autoReplyMu.Unlock()
		log.Printf("AutoReply: session %d already has an in-flight AI reply, skipping spawn", sessionID)
		return
	}
	m.autoReplyInFlight[sessionID] = true
	m.autoReplyMu.Unlock()
	defer func() {
		m.autoReplyMu.Lock()
		delete(m.autoReplyInFlight, sessionID)
		m.autoReplyMu.Unlock()
	}()

	var session models.ChatSession
	if err := database.DB.First(&session, sessionID).Error; err == nil && session.Status == "blacklisted" {
		log.Printf("AutoReply: skip session %d for user %d (blacklisted)", sessionID, entry.userID)
		return
	}

	var kb models.KnowledgeBase
	if err := database.DB.Where("user_id = ?", entry.userID).First(&kb).Error; err != nil {
		log.Printf("AutoReply: skip session %d for user %d (no KB: %v)", sessionID, entry.userID, err)
		return
	}
	if !kb.AutoReplyEnabled {
		log.Printf("AutoReply: skip session %d for user %d (auto_reply_enabled=false, kb id=%d)", sessionID, entry.userID, kb.ID)
		return
	}
	if gemini.GlobalService == nil || !gemini.GlobalService.Enabled() {
		log.Printf("AutoReply: skip session %d (AI assistant not configured)", sessionID)
		return
	}
	log.Printf("AutoReply: FIRING for session %d, user %d (kb id=%d), trigger=%q", sessionID, entry.userID, kb.ID, question)

	for iter := 0; iter < maxRethinkIterations; iter++ {
		// Anchor on the latest customer message in the session. Each loop
		// iteration re-derives this from the DB so any messages that arrived
		// while we were thinking are picked up here.
		var trigger models.ChatMessage
		if err := database.DB.
			Where("session_id = ? AND sender_type = ?", sessionID, "customer").
			Order("id DESC").
			First(&trigger).Error; err != nil {
			log.Printf("AutoReply: no customer message found for session %d, abandoning (%v)", sessionID, err)
			return
		}

		// Feature 1 (pre-check): if a human agent has already replied after
		// this customer message, skip the AI cycle entirely.
		if humanRepliedAfter(sessionID, trigger.ID) {
			log.Printf("AutoReply: human agent already replied in session %d (after msg %d), skipping AI", sessionID, trigger.ID)
			return
		}

		gemini.GlobalService.SetToolContext(gemini.ToolContext{
			UserID:    entry.userID,
			SessionID: sessionID,
			DeviceID:  entry.deviceID,
			ChatJID:   chatJID,
		})
		answer, err := gemini.GlobalService.GetAnswerWithContext(entry.userID, trigger.Content, sessionID)
		if err != nil {
			log.Printf("AI auto-reply error for user %d: %v", entry.userID, err)
			return
		}
		if answer == "" {
			return
		}

		// Artificial "typing" delay so auto-replies don't arrive instantly and
		// feel more natural. The range is admin-configurable (see gemini.delay).
		if delay := gemini.PickAIReplyDelay(); delay > 0 {
			log.Printf("AutoReply: delaying reply to session %d by %s (iter=%d, anchor=%d)", sessionID, delay, iter, trigger.ID)
			time.Sleep(delay)
		}

		// Re-check session/device after sleeping.
		var fresh models.ChatSession
		if err := database.DB.First(&fresh, sessionID).Error; err != nil || fresh.Status == "blacklisted" {
			log.Printf("AutoReply: abandoning delayed reply to session %d (status changed)", sessionID)
			return
		}
		if !entry.client.IsConnected() {
			log.Printf("AutoReply: abandoning delayed reply to session %d (device disconnected)", sessionID)
			return
		}

		// Feature 1 (final check): a human may have replied during our
		// thinking + delay window. If so, drop the AI answer.
		if humanRepliedAfter(sessionID, trigger.ID) {
			log.Printf("AutoReply: human agent replied during AI think for session %d (anchor=%d), dropping AI answer", sessionID, trigger.ID)
			return
		}

		// Feature 2: did a newer customer message arrive while we were
		// thinking? If yes, re-anchor and re-call Gemini with the latest
		// context.
		var latestCustomerID uint
		database.DB.Model(&models.ChatMessage{}).
			Where("session_id = ? AND sender_type = ?", sessionID, "customer").
			Select("id").Order("id DESC").Limit(1).Scan(&latestCustomerID)
		if latestCustomerID != trigger.ID {
			log.Printf("AutoReply: customer sent newer message in session %d (anchor=%d, latest=%d), re-thinking",
				sessionID, trigger.ID, latestCustomerID)
			continue
		}

		// Existing heuristic: if Gemini effectively said "I don't know",
		// don't send anything.
		lowerAnswer := strings.ToLower(answer)
		if strings.Contains(lowerAnswer, "sorry, i don't have an answer") ||
			strings.Contains(lowerAnswer, "connect you with a human") ||
			strings.Contains(lowerAnswer, "i don't have information") {
			log.Printf("AI could not find a match for user %d, skipping auto-reply", entry.userID)
			return
		}

		targetJID, err := types.ParseJID(chatJID)
		if err != nil {
			log.Printf("Failed to parse JID for auto-reply: %v", err)
			return
		}

		if _, err = entry.client.SendMessage(context.Background(), targetJID, &waE2E.Message{
			Conversation: proto.String(answer),
		}); err != nil {
			log.Printf("Failed to send auto-reply: %v", err)
			return
		}

		now := time.Now()
		replyMsg := models.ChatMessage{
			SessionID:   sessionID,
			SenderType:  "bot",
			Content:     answer,
			MessageType: "text",
		}
		database.DB.Create(&replyMsg)

		database.DB.Model(&models.ChatSession{}).Where("id = ?", sessionID).
			Updates(map[string]interface{}{"last_message": answer, "last_message_at": now, "last_sender_type": "bot"})

		broadcast.SessionMessage(sessionID, &replyMsg)
		log.Printf("Auto-replied to %s for user %d (anchor=%d, iter=%d)", chatJID, entry.userID, trigger.ID, iter)

		go gemini.ScheduleCustomerSummary(entry.userID, sessionID)
		return
	}

	log.Printf("AutoReply: hit max re-think iterations (%d) for session %d, giving up", maxRethinkIterations, sessionID)
}
