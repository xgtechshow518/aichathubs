package gemini

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"smart-live-chats/internal/database"
	"smart-live-chats/internal/models"
)

const (
	summaryMinInterval = time.Hour
	summaryTurnLimit   = 20
)

var (
	summaryMu       sync.Mutex
	summaryInFlight = make(map[string]bool) // "userID:phone"
)

// ScheduleCustomerSummary runs a background customer-notes summarizer after a
// bot reply. Coalesced per (user, phone) and capped to once per hour.
func ScheduleCustomerSummary(userID, sessionID uint) {
	go func() {
		var session models.ChatSession
		if err := database.DB.First(&session, sessionID).Error; err != nil {
			return
		}
		if session.CustomerPhone == "" {
			return
		}
		summarizeCustomerNotes(userID, session.CustomerPhone, sessionID)
	}()
}

func summarizeCustomerNotes(userID uint, customerPhone string, sessionID uint) {
	key := fmt.Sprintf("%d:%s", userID, customerPhone)
	summaryMu.Lock()
	if summaryInFlight[key] {
		summaryMu.Unlock()
		return
	}
	summaryInFlight[key] = true
	summaryMu.Unlock()
	defer func() {
		summaryMu.Lock()
		delete(summaryInFlight, key)
		summaryMu.Unlock()
	}()

	var profile models.CustomerProfile
	err := database.DB.Where("user_id = ? AND customer_phone = ?", userID, customerPhone).First(&profile).Error
	if err != nil {
		profile = models.CustomerProfile{
			UserID:        userID,
			CustomerPhone: customerPhone,
			LeadStage:     "cold",
		}
		database.DB.Create(&profile)
	}

	if profile.LastSummaryAt != nil && time.Since(*profile.LastSummaryAt) < summaryMinInterval {
		return
	}

	if GlobalService == nil {
		return
	}

	var messages []models.ChatMessage
	database.DB.Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Limit(summaryTurnLimit).
		Find(&messages)
	if len(messages) < 3 {
		return
	}

	var transcript strings.Builder
	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		role := m.SenderType
		if role == "bot" {
			role = "assistant"
		}
		transcript.WriteString(fmt.Sprintf("%s: %s\n", role, m.Content))
	}

	prompt := `Summarize what we now know about this customer in 80 words or fewer.
Focus on: preferences, budget signals, products discussed, objections, and lead stage (cold/warm/hot/won/lost).
Reply with only the summary paragraph, no bullet labels.`

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": "Conversation transcript:\n" + transcript.String() + "\n\n" + prompt},
				},
			},
		},
	}

	summary, err := GlobalService.callGenerateContent(reqBody)
	if err != nil {
		log.Printf("Customer summary failed for user %d phone %s: %v", userID, customerPhone, err)
		return
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return
	}

	now := time.Now()
	profile.Notes = summary
	profile.LastSummaryAt = &now
	lower := strings.ToLower(summary)
	for _, stage := range []string{"hot", "warm", "won", "lost", "cold"} {
		if strings.Contains(lower, "lead stage: "+stage) || strings.Contains(lower, stage+" lead") {
			profile.LeadStage = stage
			break
		}
	}
	database.DB.Save(&profile)
	log.Printf("Updated customer profile notes for user %d phone %s", userID, customerPhone)
}
