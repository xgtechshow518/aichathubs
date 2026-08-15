package handlers

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"smart-live-chats/internal/database"
	"smart-live-chats/internal/middleware"
	"smart-live-chats/internal/models"
	"smart-live-chats/internal/whatsapp"

	"github.com/labstack/echo/v4"
)

// ChatHandler handles chat-related endpoints
type ChatHandler struct{}

// NewChatHandler creates a new ChatHandler
func NewChatHandler() *ChatHandler {
	return &ChatHandler{}
}

// ChatListResponse represents the response for listing chats
type ChatListResponse struct {
	Chats      []models.ChatSession `json:"chats"`
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	TotalPages int                  `json:"total_pages"`
}

// ListChats returns all chat sessions with pagination
func (h *ChatHandler) ListChats(c echo.Context) error {
	// Parse query parameters
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	filter := c.QueryParam("filter") // 'all', 'unread', 'unreplied'
	platform := c.QueryParam("platform")

	// Build query
	query := database.DB.Model(&models.ChatSession{})

	// Apply filters
	switch filter {
	case "unread":
		query = query.Where("unread_count > 0")
	case "unreplied":
		// This would need more complex logic based on last message sender
		query = query.Where("unread_count > 0")
	}

	if platform != "" {
		query = query.Where("platform = ?", platform)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get paginated results
	var sessions []models.ChatSession
	offset := (page - 1) * pageSize
	if err := query.Order("last_message_at DESC").Offset(offset).Limit(pageSize).Find(&sessions).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch chats")
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return c.JSON(http.StatusOK, ChatListResponse{
		Chats:      sessions,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

// GetChat returns a single chat session by ID
func (h *ChatHandler) GetChat(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid chat ID")
	}

	var session models.ChatSession
	if err := database.DB.Preload("AssignedTo").First(&session, id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "chat not found")
	}

	return c.JSON(http.StatusOK, session)
}

// GetChatMessages returns all messages for a chat session
func (h *ChatHandler) GetChatMessages(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid chat ID")
	}

	// Check if session exists
	var session models.ChatSession
	if err := database.DB.First(&session, id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "chat not found")
	}

	// Get messages with pagination
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	messages := make([]models.ChatMessage, 0)
	offset := (page - 1) * pageSize
	if err := database.DB.Where("session_id = ?", id).
		Order("created_at ASC").
		Offset(offset).
		Limit(pageSize).
		Find(&messages).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch messages")
	}

	// Get total count
	var total int64
	database.DB.Model(&models.ChatMessage{}).Where("session_id = ?", id).Count(&total)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"messages":    messages,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (int(total) + pageSize - 1) / pageSize,
	})
}

// SendMessageRequest represents a request to send a message
type SendMessageRequest struct {
	Content     string `json:"content" validate:"required"`
	MessageType string `json:"message_type"` // 'text', 'image', 'file'
	MediaURL    string `json:"media_url"`
}

// SendMessage sends a new message to a chat session
func (h *ChatHandler) SendMessage(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid chat ID")
	}

	// Check if session exists
	var session models.ChatSession
	if err := database.DB.First(&session, id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "chat not found")
	}

	// Parse request body
	var req SendMessageRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Content == "" && req.MediaURL == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "message content or media is required")
	}

	if req.MessageType == "" {
		req.MessageType = "text"
	}

	// Get current user
	user := middleware.GetUserFromContext(c)
	var senderID *uint
	if user != nil {
		senderID = &user.ID
	}

	// Create message
	message := models.ChatMessage{
		SessionID:   uint(id),
		SenderType:  "agent",
		SenderID:    senderID,
		Content:     req.Content,
		MessageType: req.MessageType,
		MediaURL:    req.MediaURL,
		IsRead:      false,
	}

	if err := database.DB.Create(&message).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to send message")
	}

	session.LastMessage = req.Content
	now := message.CreatedAt
	session.LastMessageAt = &now
	session.LastSenderType = "agent"
	database.DB.Save(&session)

	// Broadcast message via WebSocket
	BroadcastMessage(uint(id), &message)

	// If this is a WhatsApp session, also send the message via WhatsApp
	if session.Platform == "whatsapp" && session.CustomerPhone != "" && session.DeviceID != nil {
		// Try @lid first (newer WhatsApp LID format), then @s.whatsapp.net
		// WhatsApp silently accepts sends to wrong JID formats without error,
		// so we must try the correct format first.
		lidJid := session.CustomerPhone + "@lid"
		if err := whatsapp.GlobalManager.SendMessageViaDevice(*session.DeviceID, lidJid, req.Content); err != nil {
			log.Printf("Send via @lid failed, trying @s.whatsapp.net: %v", err)
			phoneJid := session.CustomerPhone + "@s.whatsapp.net"
			if err2 := whatsapp.GlobalManager.SendMessageViaDevice(*session.DeviceID, phoneJid, req.Content); err2 != nil {
				log.Printf("Failed to send WhatsApp message via both JID formats: %v", err2)
			}
		}
	}

	return c.JSON(http.StatusCreated, message)
}

// MarkAsRead marks all messages in a session as read
func (h *ChatHandler) MarkAsRead(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid chat ID")
	}

	// Update messages
	if err := database.DB.Model(&models.ChatMessage{}).
		Where("session_id = ? AND is_read = ?", id, false).
		Update("is_read", true).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to mark as read")
	}

	// Update session unread count
	if err := database.DB.Model(&models.ChatSession{}).
		Where("id = ?", id).
		Update("unread_count", 0).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update unread count")
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// UploadImage handles image upload for a chat session
func (h *ChatHandler) UploadImage(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid chat ID")
	}

	var session models.ChatSession
	if err := database.DB.First(&session, id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "chat not found")
	}

	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "file is required")
	}

	// For now, store as a data URL (in production, upload to S3/GCS)
	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to open file")
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to read file")
	}

	mimeType := http.DetectContentType(data)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data))

	user := middleware.GetUserFromContext(c)
	var senderID *uint
	if user != nil {
		senderID = &user.ID
	}

	message := models.ChatMessage{
		SessionID:   uint(id),
		SenderType:  "agent",
		SenderID:    senderID,
		Content:     file.Filename,
		MessageType: "image",
		MediaURL:    dataURL,
		IsRead:      false,
	}

	if err := database.DB.Create(&message).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to send image")
	}

	session.LastMessage = "[Image]"
	now := message.CreatedAt
	session.LastMessageAt = &now
	session.LastSenderType = "agent"
	database.DB.Save(&session)

	BroadcastMessage(uint(id), &message)

	return c.JSON(http.StatusCreated, message)
}

// AcceptConversation assigns the current user to the chat session
func (h *ChatHandler) AcceptConversation(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid chat ID")
	}

	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	if err := database.DB.Model(&models.ChatSession{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"assigned_to_id": user.ID,
			"status":         "active",
		}).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to accept conversation")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":      "accepted",
		"assigned_to": user.ID,
	})
}

// BlacklistCustomer marks a chat session as blacklisted and closes it
func (h *ChatHandler) BlacklistCustomer(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid chat ID")
	}

	if err := database.DB.Model(&models.ChatSession{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status": "blacklisted",
		}).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to blacklist customer")
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "blacklisted"})
}

// GetBlacklisted returns all blacklisted chat sessions
func (h *ChatHandler) GetBlacklisted(c echo.Context) error {
	var sessions []models.ChatSession
	database.DB.Where("status = ?", "blacklisted").Order("updated_at DESC").Find(&sessions)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"sessions": sessions,
		"total":    len(sessions),
	})
}

// UnblacklistCustomer removes a customer from the blacklist
func (h *ChatHandler) UnblacklistCustomer(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid chat ID")
	}

	result := database.DB.Model(&models.ChatSession{}).Where("id = ? AND status = ?", id, "blacklisted").
		Update("status", "active")
	if result.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "blacklisted session not found")
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "active"})
}

// GetChatStats returns statistics about chats
func (h *ChatHandler) GetChatStats(c echo.Context) error {
	var totalChats int64
	var unreadChats int64
	var activeChats int64

	database.DB.Model(&models.ChatSession{}).Count(&totalChats)
	database.DB.Model(&models.ChatSession{}).Where("unread_count > 0").Count(&unreadChats)
	database.DB.Model(&models.ChatSession{}).Where("status = ?", "active").Count(&activeChats)

	return c.JSON(http.StatusOK, map[string]int64{
		"total_chats":   totalChats,
		"unread_chats":  unreadChats,
		"active_chats":  activeChats,
	})
}

// GetReportData returns analytics data for the reports page
func (h *ChatHandler) GetReportData(c echo.Context) error {
	var totalMessages int64
	var customerMessages int64
	var agentMessages int64
	var totalCustomers int64

	database.DB.Model(&models.ChatMessage{}).Count(&totalMessages)
	database.DB.Model(&models.ChatMessage{}).Where("sender_type = ?", "customer").Count(&customerMessages)
	database.DB.Model(&models.ChatMessage{}).Where("sender_type = ?", "agent").Count(&agentMessages)
	database.DB.Model(&models.ChatSession{}).Count(&totalCustomers)

	// Messages per day (last 7 days)
	type DayStat struct {
		Day   string `json:"day"`
		Count int64  `json:"count"`
	}
	var dailyStats []DayStat
	database.DB.Model(&models.ChatMessage{}).
		Select("DATE(created_at) as day, COUNT(*) as count").
		Where("created_at >= NOW() - INTERVAL '7 days'").
		Group("DATE(created_at)").
		Order("day ASC").
		Scan(&dailyStats)

	// Top customers by message count
	type TopCustomer struct {
		CustomerName string `json:"customer_name"`
		Platform     string `json:"platform"`
		MessageCount int64  `json:"message_count"`
	}
	var topCustomers []TopCustomer
	database.DB.Model(&models.ChatSession{}).
		Select("customer_name, platform, (SELECT COUNT(*) FROM chat_messages WHERE session_id = chat_sessions.id) as message_count").
		Order("message_count DESC").
		Limit(5).
		Scan(&topCustomers)

	// Platform breakdown
	type PlatformStat struct {
		Platform string `json:"platform"`
		Count    int64  `json:"count"`
	}
	var platformStats []PlatformStat
	database.DB.Model(&models.ChatSession{}).
		Select("platform, COUNT(*) as count").
		Group("platform").
		Scan(&platformStats)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"total_messages":    totalMessages,
		"customer_messages": customerMessages,
		"agent_messages":    agentMessages,
		"total_customers":   totalCustomers,
		"auto_reply_rate":   func() float64 { if customerMessages == 0 { return 0 }; return float64(agentMessages) / float64(customerMessages) * 100 }(),
		"daily_stats":       dailyStats,
		"top_customers":     topCustomers,
		"platform_stats":    platformStats,
	})
}

