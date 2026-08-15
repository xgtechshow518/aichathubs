package gemini

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"smart-live-chats/internal/config"
	"smart-live-chats/internal/database"
	"smart-live-chats/internal/models"
)

const baseURL = "https://generativelanguage.googleapis.com/v1beta"

// PlatformSystemPromptKey is the key used in the platform_settings table
// to store the globally managed system prompt for the AI sales bot.
const PlatformSystemPromptKey = "gemini_system_prompt"

// systemPromptCacheTTL controls how long the resolved system prompt is cached
// in-memory. Short TTL keeps admin edits fast to apply without hammering the DB.
const systemPromptCacheTTL = 30 * time.Second

type Service struct {
	apiKey              string
	model               string
	systemPrompt        string // env-level default; used only when no DB override exists
	matchScoreThreshold float64
	cacheMu             sync.RWMutex

	promptMu       sync.RWMutex
	promptCache    string
	promptCacheExp time.Time

	toolCtxMu sync.Mutex
	toolCtx   ToolContext
}

// SetToolContext binds session/device info for tool side effects (image send, leads).
func (s *Service) SetToolContext(ctx ToolContext) {
	s.toolCtxMu.Lock()
	s.toolCtx = ctx
	s.toolCtxMu.Unlock()
}

func (s *Service) getToolContext() ToolContext {
	s.toolCtxMu.Lock()
	c := s.toolCtx
	s.toolCtxMu.Unlock()
	return c
}

var GlobalService *Service

// ErrAINotConfigured is returned by Service methods when no Gemini API key is
// set. Callers should treat this as "AI features are disabled", not a failure.
var ErrAINotConfigured = errors.New("ai_not_configured")

// Enabled reports whether a Gemini API key is configured. When false, all AI
// features (auto-reply, knowledge test, sync) are turned off gracefully.
func (s *Service) Enabled() bool {
	return s != nil && s.apiKey != ""
}

func NewService(cfg *config.Config) *Service {
	s := &Service{
		apiKey:              cfg.GeminiAPIKey,
		model:               cfg.GeminiModel,
		systemPrompt:        cfg.GeminiSystemPrompt,
		matchScoreThreshold: cfg.MatchScoreThreshold,
	}
	GlobalService = s
	return s
}

// ResolveSystemPrompt returns the effective system prompt for the AI bot.
// Precedence: platform_settings row (admin-managed) → env-level default.
// Uses a short in-memory cache to avoid a DB hit on every Gemini call.
func (s *Service) ResolveSystemPrompt() string {
	s.promptMu.RLock()
	if s.promptCache != "" && time.Now().Before(s.promptCacheExp) {
		cached := s.promptCache
		s.promptMu.RUnlock()
		return cached
	}
	s.promptMu.RUnlock()

	s.promptMu.Lock()
	defer s.promptMu.Unlock()
	// Re-check after acquiring the write lock
	if s.promptCache != "" && time.Now().Before(s.promptCacheExp) {
		return s.promptCache
	}

	prompt := s.systemPrompt
	var setting models.PlatformSetting
	if err := database.DB.Where("key = ?", PlatformSystemPromptKey).First(&setting).Error; err == nil {
		if strings.TrimSpace(setting.Value) != "" {
			prompt = setting.Value
		}
	}
	s.promptCache = prompt
	s.promptCacheExp = time.Now().Add(systemPromptCacheTTL)
	return prompt
}

// ResolveSystemPromptForUser returns the effective system prompt for a single
// user. Precedence: the user's own knowledge_bases.system_prompt (set by the
// platform admin or the user themselves) → platform_settings row → env default.
// A non-empty per-user prompt fully replaces the platform/default persona for
// that user only; other users keep the platform default.
func (s *Service) ResolveSystemPromptForUser(userID uint) string {
	if userID > 0 {
		var kb models.KnowledgeBase
		if err := database.DB.Select("system_prompt").Where("user_id = ?", userID).First(&kb).Error; err == nil {
			if p := strings.TrimSpace(kb.SystemPrompt); p != "" {
				return p
			}
		}
	}
	return s.ResolveSystemPrompt()
}

// FriendlyErrorMessage converts an internal service/Gemini error into a safe,
// user-facing message. It never leaks raw API responses, quota figures, API
// keys, URLs, or stack details. Callers should log the original error and show
// only this string to end users.
func FriendlyErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, ErrAINotConfigured) {
		return "The AI assistant is not configured on this server."
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "status 429"), strings.Contains(msg, "RESOURCE_EXHAUSTED"):
		return "Sorry I am handling a lot of requests right now. Please wait a moment and try again."
	case strings.Contains(msg, "status 401"), strings.Contains(msg, "status 403"),
		strings.Contains(msg, "API key"), strings.Contains(msg, "API_KEY"):
		return "The assistant is temporarily unavailable. Please try again later."
	default:
		return "Sorry, I couldn't generate a reply right now. Please try again in a moment."
	}
}

// BuildSystemPromptForUser assembles persona + catalog hint + FAQ + customer memory.
// Product specifics come from the structured products table and are reached via
// tools (search_products, get_product_by_sku).
func (s *Service) BuildSystemPromptForUser(userID uint, sessionID uint) string {
	prompt := s.ResolveSystemPromptForUser(userID)

	var parts []string
	parts = append(parts, prompt)

	// Customer memory (PR-4)
	if sessionID > 0 {
		var session models.ChatSession
		if err := database.DB.First(&session, sessionID).Error; err == nil && session.CustomerPhone != "" {
			var profile models.CustomerProfile
			if err := database.DB.Where("user_id = ? AND customer_phone = ?", userID, session.CustomerPhone).First(&profile).Error; err == nil {
				if notes := strings.TrimSpace(profile.Notes); notes != "" {
					parts = append(parts, "\n\n---\nKNOWN ABOUT THIS CUSTOMER:\n"+notes)
				}
			}
		}
	}

	var productCount int64
	database.DB.Model(&models.Product{}).
		Where("user_id = ? AND active = ? AND deleted_at IS NULL", userID, true).
		Count(&productCount)

	if productCount > 0 {
		parts = append(parts, "\n\n---\n"+RenderProductCatalogSummary(userID, true))
	}

	if faq := renderTopFAQ(userID, 10); faq != "" {
		parts = append(parts, "\n\n---\n"+faq)
	}

	return strings.Join(parts, "")
}

// InvalidateSystemPromptCache forces the next ResolveSystemPrompt call to re-read
// from the database. Call this right after an admin updates the prompt.
func (s *Service) InvalidateSystemPromptCache() {
	s.promptMu.Lock()
	s.promptCache = ""
	s.promptCacheExp = time.Time{}
	s.promptMu.Unlock()
}

// InvalidateUserContextCache clears a single user's Gemini context cache.
// Call this when any input baked into the cache (system prompt, product info,
// QA content) has changed and the cached version would now be stale.
func (s *Service) InvalidateUserContextCache(userID uint) {
	var kb models.KnowledgeBase
	if err := database.DB.Where("user_id = ?", userID).First(&kb).Error; err != nil {
		return
	}
	if kb.CacheName != "" {
		s.deleteCache(kb.CacheName)
	}
	database.DB.Model(&models.KnowledgeBase{}).
		Where("id = ?", kb.ID).
		Updates(map[string]interface{}{
			"cache_name":       "",
			"cache_expires_at": nil,
		})
	log.Printf("Invalidated context cache for user %d", userID)
}

// InvalidateAllContextCaches clears every user's Gemini context cache. Required
// after the admin changes the global system prompt, because existing context
// caches bundle the old prompt and would otherwise keep serving it until TTL.
func (s *Service) InvalidateAllContextCaches() {
	var kbs []models.KnowledgeBase
	if err := database.DB.Where("cache_name <> ''").Find(&kbs).Error; err != nil {
		log.Printf("InvalidateAllContextCaches: failed to list KBs: %v", err)
		return
	}
	for _, kb := range kbs {
		if kb.CacheName != "" {
			s.deleteCache(kb.CacheName)
		}
		database.DB.Model(&models.KnowledgeBase{}).
			Where("id = ?", kb.ID).
			Updates(map[string]interface{}{
				"cache_name":       "",
				"cache_expires_at": nil,
			})
	}
	log.Printf("Invalidated context cache for %d knowledge base(s)", len(kbs))
}

// SyncQAItems replaces the Gemini store with fresh Q&A data and creates/refreshes the context cache.
func (s *Service) SyncQAItems(userID uint) error {
	if !s.Enabled() {
		return ErrAINotConfigured
	}
	var items []models.QAItem
	database.DB.Where("user_id = ?", userID).Find(&items)

	if len(items) == 0 {
		return nil
	}

	var kb models.KnowledgeBase
	result := database.DB.Where("user_id = ?", userID).First(&kb)
	if result.Error != nil {
		kb = models.KnowledgeBase{UserID: userID}
		database.DB.Create(&kb)
	}

	// Delete old store if exists
	if kb.GeminiStoreName != "" {
		s.deleteStore(kb.GeminiStoreName)
	}

	// Delete old cache if exists
	if kb.CacheName != "" {
		s.deleteCache(kb.CacheName)
		kb.CacheName = ""
		kb.CacheExpiresAt = nil
	}

	// Create fresh store
	storeID, storeName, err := s.createStore(fmt.Sprintf("user_%d_qa_store", userID))
	if err != nil {
		return fmt.Errorf("failed to create Gemini store: %w", err)
	}

	kb.GeminiStoreID = storeID
	kb.GeminiStoreName = storeName

	// Build Q&A content
	var content strings.Builder
	content.WriteString("# Knowledge Base - Questions and Answers\n\n")
	for i, item := range items {
		content.WriteString(fmt.Sprintf("## Q%d", i+1))
		if item.Category != "" {
			content.WriteString(fmt.Sprintf(" [%s]", item.Category))
		}
		content.WriteString(fmt.Sprintf("\n**Question:** %s\n**Answer:** %s\n\n", item.Question, item.Answer))
	}

	tmpFile, err := os.CreateTemp("", "qa_*.txt")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content.String()); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	if err := s.uploadToStore(storeName, tmpFile.Name(), "qa_knowledge_base.txt"); err != nil {
		return fmt.Errorf("failed to upload to Gemini store: %w", err)
	}

	// Create explicit context cache with the Q&A content (24h TTL)
	cacheName, err := s.createCache(content.String(), userID)
	if err != nil {
		log.Printf("Failed to create context cache for user %d (will use uncached): %v", userID, err)
	} else {
		cacheExpiry := time.Now().Add(24 * time.Hour)
		kb.CacheName = cacheName
		kb.CacheExpiresAt = &cacheExpiry
		log.Printf("Created context cache for user %d: %s (expires in 24h)", userID, cacheName)
	}

	now := time.Now()
	kb.LastSyncedAt = &now
	database.DB.Save(&kb)

	log.Printf("Synced %d QA items to fresh Gemini store for user %d", len(items), userID)
	return nil
}

// UploadFile uploads a user-provided file to their Gemini store (additive, no replace)
func (s *Service) UploadFile(userID uint, filePath, displayName string) error {
	if !s.Enabled() {
		return ErrAINotConfigured
	}
	kb, err := s.ensureStore(userID)
	if err != nil {
		return err
	}

	if err := s.uploadToStore(kb.GeminiStoreName, filePath, displayName); err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	// Invalidate cache since content changed
	if kb.CacheName != "" {
		s.deleteCache(kb.CacheName)
		kb.CacheName = ""
		kb.CacheExpiresAt = nil
	}

	now := time.Now()
	kb.LastSyncedAt = &now
	database.DB.Save(kb)

	return nil
}

type ChatTurn struct {
	Role    string // "user" or "model"
	Content string
}

// GetAnswerWithContext is like GetAnswer but includes conversation history for multi-turn context.
// When a local DB match is found, the answer is enhanced through Gemini with follow-up questions.
func (s *Service) GetAnswerWithContext(userID uint, question string, sessionID uint) (string, error) {
	if !s.Enabled() {
		return "", ErrAINotConfigured
	}
	history := s.loadChatHistory(sessionID, 20)
	return s.GetAnswerForConversation(userID, question, history, sessionID)
}

// GetAnswerForConversation runs the full multi-turn AI pipeline against an
// explicitly supplied conversation history. The persisted-session entry point
// (GetAnswerWithContext) is a thin wrapper that loads history from the DB first.
//
// sessionID may be 0 for ephemeral conversations (e.g. the in-dashboard "Test
// Bot" sandbox) that are not backed by a stored ChatSession. In that case the
// pipeline behaves identically for the customer-facing reply, but tool side
// effects that require a real session (lead capture, sending images, customer
// memory) are skipped — see the SessionID guards in the tool executors.
func (s *Service) GetAnswerForConversation(userID uint, question string, history []ChatTurn, sessionID uint) (string, error) {
	if !s.Enabled() {
		return "", ErrAINotConfigured
	}
	var kb models.KnowledgeBase
	if err := database.DB.Where("user_id = ?", userID).First(&kb).Error; err != nil {
		return "", fmt.Errorf("knowledge base not found")
	}

	// Per-user system prompt (if set) overrides the global admin-managed prompt,
	// then the user's product catalog blob is appended.
	systemPrompt := s.BuildSystemPromptForUser(userID, sessionID)

	answer, score := s.localMatch(userID, question)

	if score >= s.matchScoreThreshold {
		log.Printf("Local match for user %d (score=%.2f): enhancing with AI context", userID, score)
		enhanced, err := s.enhanceAnswer(answer, history, systemPrompt)
		if err != nil {
			log.Printf("Enhancement failed, returning raw answer: %v", err)
			return answer, nil
		}
		return enhanced, nil
	}

	if kb.GeminiStoreName == "" {
		if score >= s.matchScoreThreshold*0.7 && answer != "" {
			enhanced, err := s.enhanceAnswer(answer, history, systemPrompt)
			if err != nil {
				return answer, nil
			}
			return enhanced, nil
		}
		// Tool-only path when no Gemini file-search store (products + tools still work).
		ans, err := s.queryWithToolsOnly(history, systemPrompt, userID, sessionID)
		if err == nil && ans != "" {
			return ans, nil
		}
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("no AI store configured")
	}

	log.Printf("No local match (best score=%.2f), falling back to AI with history for user %d", score, userID)

	if kb.CacheName != "" && kb.CacheExpiresAt != nil && time.Now().Before(*kb.CacheExpiresAt) {
		ans, err := s.queryWithCacheAndHistory(kb.CacheName, history)
		if err == nil && ans != "" {
			return ans, nil
		}
		if isStaleResourceError(err) {
			log.Printf("Cache %s no longer accessible (likely API key changed); clearing", kb.CacheName)
			database.DB.Model(&kb).Updates(map[string]interface{}{
				"cache_name":       "",
				"cache_expires_at": nil,
			})
		} else {
			log.Printf("Cache query with history failed: %v", err)
		}
	}

	ans, err := s.queryWithFileSearchAndHistory(kb.GeminiStoreName, history, systemPrompt, userID, sessionID)
	if err != nil && isStaleResourceError(err) {
		log.Printf("File search store %s no longer accessible (likely API key changed); clearing. Please re-sync knowledge base.", kb.GeminiStoreName)
		database.DB.Model(&kb).Updates(map[string]interface{}{
			"gemini_store_id":   "",
			"gemini_store_name": "",
			"cache_name":        "",
			"cache_expires_at":  nil,
		})
		if score >= s.matchScoreThreshold*0.7 && answer != "" {
			enhanced, eErr := s.enhanceAnswer(answer, history, systemPrompt)
			if eErr != nil {
				return answer, nil
			}
			return enhanced, nil
		}
		return "", fmt.Errorf("knowledge base store is no longer accessible; please re-sync your knowledge base in Settings")
	}
	return ans, err
}

func (s *Service) loadChatHistory(sessionID uint, limit int) []ChatTurn {
	var messages []models.ChatMessage
	database.DB.Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages)

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	var turns []ChatTurn
	for _, msg := range messages {
		role := "user"
		if msg.SenderType == "bot" || msg.SenderType == "agent" {
			role = "model"
		}
		if msg.Content != "" {
			turns = append(turns, ChatTurn{Role: role, Content: msg.Content})
		}
	}
	return turns
}

// buildContentsFromHistory builds a Gemini-compatible multi-turn contents array.
// Merges consecutive same-role messages and ensures proper user/model alternation.
func (s *Service) buildContentsFromHistory(history []ChatTurn) []map[string]interface{} {
	if len(history) == 0 {
		return nil
	}

	var merged []ChatTurn
	for _, turn := range history {
		if len(merged) > 0 && merged[len(merged)-1].Role == turn.Role {
			merged[len(merged)-1].Content += "\n" + turn.Content
		} else {
			merged = append(merged, turn)
		}
	}

	for len(merged) > 0 && merged[0].Role != "user" {
		merged = merged[1:]
	}
	for len(merged) > 0 && merged[len(merged)-1].Role != "user" {
		merged = merged[:len(merged)-1]
	}

	if len(merged) == 0 {
		return nil
	}

	var contents []map[string]interface{}
	for _, turn := range merged {
		contents = append(contents, map[string]interface{}{
			"role": turn.Role,
			"parts": []map[string]string{
				{"text": turn.Content},
			},
		})
	}
	return contents
}

// enhanceAnswer sends a DB-matched answer through Gemini with conversation context
// so the AI can deliver the answer naturally and add proactive follow-up questions.
func (s *Service) enhanceAnswer(dbAnswer string, history []ChatTurn, systemPrompt string) (string, error) {
	contents := s.buildContentsFromHistory(history)
	if contents == nil {
		return dbAnswer, nil
	}

	enhancedPrompt := systemPrompt + "\n\n---\n" +
		"KNOWLEDGE BASE REFERENCE (verified answer for the current question — use as factual basis, do not contradict):\n" +
		dbAnswer

	reqBody := map[string]interface{}{
		"contents": contents,
		"systemInstruction": map[string]interface{}{
			"parts": []map[string]string{
				{"text": enhancedPrompt},
			},
		},
	}

	return s.callGenerateContent(reqBody)
}

func (s *Service) queryWithCacheAndHistory(cacheName string, history []ChatTurn) (string, error) {
	contents := s.buildContentsFromHistory(history)
	if contents == nil {
		return "", fmt.Errorf("no valid conversation history")
	}

	reqBody := map[string]interface{}{
		"contents":      contents,
		"cachedContent": cacheName,
	}

	return s.callGenerateContent(reqBody)
}

func (s *Service) queryWithToolsOnly(history []ChatTurn, systemPrompt string, userID, sessionID uint) (string, error) {
	contents := s.buildContentsFromHistory(history)
	if contents == nil {
		return "", fmt.Errorf("no valid conversation history")
	}
	reqBody := map[string]interface{}{
		"contents": contents,
		"systemInstruction": map[string]interface{}{
			"parts": []map[string]string{{"text": systemPrompt}},
		},
	}
	return s.callGenerateContentForSession(userID, sessionID, reqBody, "")
}

func (s *Service) queryWithFileSearchAndHistory(storeName string, history []ChatTurn, systemPrompt string, userID, sessionID uint) (string, error) {
	contents := s.buildContentsFromHistory(history)
	if contents == nil {
		return "", fmt.Errorf("no valid conversation history")
	}

	reqBody := map[string]interface{}{
		"contents": contents,
		"systemInstruction": map[string]interface{}{
			"parts": []map[string]string{
				{"text": systemPrompt},
			},
		},
	}

	return s.callGenerateContentForSession(userID, sessionID, reqBody, storeName)
}

// GetAnswer first tries exact matching from the local DB, then falls back to Gemini.
func (s *Service) GetAnswer(userID uint, question string) (string, error) {
	if !s.Enabled() {
		return "", ErrAINotConfigured
	}
	answer, score := s.localMatch(userID, question)
	if score >= s.matchScoreThreshold {
		log.Printf("Local match for user %d (score=%.2f, threshold=%.2f): %q -> %q", userID, score, s.matchScoreThreshold, question, answer)
		return answer, nil
	}

	var kb models.KnowledgeBase
	if err := database.DB.Where("user_id = ?", userID).First(&kb).Error; err != nil {
		return "", fmt.Errorf("knowledge base not found")
	}

	if kb.GeminiStoreName == "" {
		if score >= s.matchScoreThreshold*0.7 && answer != "" {
			return answer, nil
		}
		return "", fmt.Errorf("no AI store configured")
	}

	log.Printf("No local match (best score=%.2f), falling back to AI for user %d", score, userID)

	if kb.CacheName != "" && kb.CacheExpiresAt != nil && time.Now().Before(*kb.CacheExpiresAt) {
		ans, err := s.queryWithCache(kb.CacheName, question)
		if err == nil && ans != "" {
			return ans, nil
		}
		if isStaleResourceError(err) {
			log.Printf("Cache %s no longer accessible (likely API key changed); clearing", kb.CacheName)
			database.DB.Model(&kb).Updates(map[string]interface{}{
				"cache_name":       "",
				"cache_expires_at": nil,
			})
		} else {
			log.Printf("Cache query failed, trying file search: %v", err)
		}
	}

	ans, err := s.queryWithFileSearch(kb.GeminiStoreName, question, s.BuildSystemPromptForUser(userID, 0))
	if err != nil && isStaleResourceError(err) {
		log.Printf("File search store %s no longer accessible (likely API key changed); clearing. Please re-sync knowledge base.", kb.GeminiStoreName)
		database.DB.Model(&kb).Updates(map[string]interface{}{
			"gemini_store_id":   "",
			"gemini_store_name": "",
			"cache_name":        "",
			"cache_expires_at":  nil,
		})
		if score >= s.matchScoreThreshold*0.7 && answer != "" {
			return answer, nil
		}
		return "", fmt.Errorf("knowledge base store is no longer accessible; please re-sync your knowledge base in Settings")
	}
	return ans, err
}

// isStaleResourceError returns true if the Gemini error indicates a store/cache
// that no longer exists or is not accessible under the current API key
// (e.g. after the GEMINI_API_KEY / Google project was changed).
func isStaleResourceError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "PERMISSION_DENIED") ||
		strings.Contains(msg, "NOT_FOUND") ||
		strings.Contains(msg, "status 403") ||
		strings.Contains(msg, "status 404")
}

// localMatch uses PostgreSQL trigram similarity to find the best matching Q&A pair
func (s *Service) localMatch(userID uint, question string) (string, float64) {
	var result struct {
		Answer     string  `json:"answer"`
		Similarity float64 `json:"similarity"`
	}

	err := database.DB.Raw(`
		SELECT answer, similarity(lower(question), lower(?)) as similarity
		FROM qa_items
		WHERE user_id = ? AND deleted_at IS NULL
		ORDER BY similarity DESC
		LIMIT 1
	`, question, userID).Scan(&result).Error

	if err != nil {
		log.Printf("Trigram similarity query failed (pg_trgm may not be installed): %v", err)
		return s.keywordMatch(userID, question)
	}

	if result.Answer == "" {
		return "", 0
	}

	return result.Answer, result.Similarity
}

// keywordMatch is a fallback when pg_trgm is unavailable — uses ILIKE keyword search
func (s *Service) keywordMatch(userID uint, question string) (string, float64) {
	words := strings.Fields(strings.ToLower(question))
	if len(words) == 0 {
		return "", 0
	}

	// Try exact substring match first
	var item models.QAItem
	err := database.DB.Where("user_id = ? AND deleted_at IS NULL AND lower(question) LIKE ?",
		userID, "%"+strings.ToLower(question)+"%").First(&item).Error
	if err == nil {
		return item.Answer, 0.8
	}

	// Try matching individual keywords (at least 3 chars)
	for _, word := range words {
		if len(word) < 3 {
			continue
		}
		err := database.DB.Where("user_id = ? AND deleted_at IS NULL AND lower(question) LIKE ?",
			userID, "%"+word+"%").First(&item).Error
		if err == nil {
			return item.Answer, 0.4
		}
	}

	return "", 0
}

func (s *Service) queryWithCache(cacheName, question string) (string, error) {
	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": question},
				},
			},
		},
		"cachedContent": cacheName,
	}

	return s.callGenerateContent(reqBody)
}

func (s *Service) queryWithFileSearch(storeName, question, systemPrompt string) (string, error) {
	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": question},
				},
			},
		},
		"systemInstruction": map[string]interface{}{
			"parts": []map[string]string{
				{"text": systemPrompt},
			},
		},
		"tools": []map[string]interface{}{
			{
				"fileSearch": map[string]interface{}{
					"fileSearchStoreNames": []string{storeName},
				},
			},
		},
	}

	return s.callGenerateContent(reqBody)
}

// generateContentMaxRetries and generateContentBaseDelay control how
// aggressively we retry transient Gemini errors (429 / 5xx). In practice
// 503 "model is experiencing high demand" bursts resolve within a few
// seconds, so a short exponential backoff recovers the auto-reply without
// making the customer wait too long.
const (
	generateContentMaxRetries = 3
	generateContentBaseDelay  = 1500 * time.Millisecond
)

// isRetryableGenerateContentStatus returns true for upstream statuses that
// are safe to retry (rate-limit + transient server errors). 4xx other than
// 429 mean the request itself is bad and retrying would just repeat the
// error, so we surface those immediately.
func isRetryableGenerateContentStatus(status int) bool {
	switch status {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

func (s *Service) callGenerateContent(reqBody map[string]interface{}) (string, error) {
	jsonBody, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", baseURL, s.model, s.apiKey)

	var (
		body   []byte
		status int
		lastErr error
	)

	for attempt := 0; attempt <= generateContentMaxRetries; attempt++ {
		resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBody))
		if err != nil {
			lastErr = fmt.Errorf("Gemini API request failed: %w", err)
			// Network-level failures are also worth retrying.
			if attempt < generateContentMaxRetries {
				backoff := generateContentBaseDelay << attempt
				log.Printf("Gemini request error (attempt %d/%d): %v — retrying in %s",
					attempt+1, generateContentMaxRetries+1, err, backoff)
				time.Sleep(backoff)
				continue
			}
			return "", lastErr
		}

		body, _ = io.ReadAll(resp.Body)
		status = resp.StatusCode
		resp.Body.Close()

		if status == http.StatusOK {
			break
		}

		if isRetryableGenerateContentStatus(status) && attempt < generateContentMaxRetries {
			backoff := generateContentBaseDelay << attempt
			log.Printf("Gemini API transient error (status %d, attempt %d/%d): retrying in %s",
				status, attempt+1, generateContentMaxRetries+1, backoff)
			time.Sleep(backoff)
			continue
		}

		return "", fmt.Errorf("Gemini API error (status %d): %s", status, string(body))
	}

	if status != http.StatusOK {
		return "", fmt.Errorf("Gemini API error (status %d): %s", status, string(body))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			CachedContentTokenCount int `json:"cachedContentTokenCount"`
			TotalTokenCount         int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	if result.UsageMetadata.CachedContentTokenCount > 0 {
		log.Printf("Cache hit: %d/%d tokens from cache",
			result.UsageMetadata.CachedContentTokenCount, result.UsageMetadata.TotalTokenCount)
	}

	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		answer := strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text)
		if answer == "" {
			return "", fmt.Errorf("empty response from AI")
		}
		return answer, nil
	}

	return "", fmt.Errorf("no response from AI")
}

// geminiPart represents a single part in a Gemini candidate (text or function call).
type geminiPart struct {
	Text         string `json:"text"`
	FunctionCall *struct {
		Name string          `json:"name"`
		Args json.RawMessage `json:"args"`
	} `json:"functionCall"`
}

type geminiCandidate struct {
	Content struct {
		Role  string       `json:"role"`
		Parts []geminiPart `json:"parts"`
	} `json:"content"`
}

// callGenerateContentForSession runs generateContent with function-calling tools
// (and optional file search), executing up to maxToolHops tool rounds.
// retrieveFileSearchContext performs a file_search-only generateContent call
// (no function declarations) so it can be safely combined later with the
// function-calling loop. It reuses the conversation contents and system
// instruction from reqBody and returns the grounded text answer.
func (s *Service) retrieveFileSearchContext(reqBody map[string]interface{}, fileSearchStore string) (string, error) {
	fsReq := map[string]interface{}{
		"contents": reqBody["contents"],
		"tools": []map[string]interface{}{
			{
				"fileSearch": map[string]interface{}{
					"fileSearchStoreNames": []string{fileSearchStore},
				},
			},
		},
	}
	if si, ok := reqBody["systemInstruction"]; ok {
		fsReq["systemInstruction"] = si
	}
	return s.callGenerateContent(fsReq)
}

// appendSystemInstructionText returns a systemInstruction with an extra text
// part appended, preserving any existing parts. It tolerates a nil/absent
// instruction.
func appendSystemInstructionText(si interface{}, extra string) map[string]interface{} {
	parts := []map[string]string{}
	if m, ok := si.(map[string]interface{}); ok {
		if existing, ok := m["parts"].([]map[string]string); ok {
			parts = append(parts, existing...)
		}
	}
	parts = append(parts, map[string]string{"text": extra})
	return map[string]interface{}{"parts": parts}
}

func (s *Service) callGenerateContentForSession(userID, sessionID uint, reqBody map[string]interface{}, fileSearchStore string) (string, error) {
	toolCtx := s.getToolContext()
	if toolCtx.UserID == 0 {
		toolCtx = ToolContext{UserID: userID, SessionID: sessionID}
	}

	// Gemini rejects requests that combine the built-in file_search tool with
	// custom function-calling declarations ("Built-in tools and Function Calling
	// cannot be combined in the same request"). When a KB store is available we
	// therefore do it in two steps: (1) retrieve grounded knowledge-base context
	// using file_search alone, then (2) inject that context into the system
	// instruction and run the function-calling loop (products, leads, images)
	// with tools only.
	if fileSearchStore != "" {
		kbContext, err := s.retrieveFileSearchContext(reqBody, fileSearchStore)
		if err != nil {
			if isStaleResourceError(err) {
				// Surface stale store/permission errors so the caller can clear
				// the store and prompt a re-sync.
				return "", err
			}
			log.Printf("File search retrieval failed for user %d (continuing with tools only): %v", userID, err)
		} else if kbContext != "" {
			reqBody["systemInstruction"] = appendSystemInstructionText(reqBody["systemInstruction"],
				"\n\n---\nRELEVANT KNOWLEDGE BASE CONTEXT (retrieved for the current question — "+
					"use as the factual basis for your reply and do not contradict it):\n"+kbContext)
		}
	}

	toolsPayload := []map[string]interface{}{
		{"functionDeclarations": s.toolDeclarations()},
	}
	reqBody["tools"] = toolsPayload

	registry := make(map[string]toolDef)
	for _, t := range s.toolRegistry() {
		registry[t.Name] = t
	}

	for hop := 0; hop <= maxToolHops; hop++ {
		body, err := s.postGenerateContent(reqBody)
		if err != nil {
			return "", err
		}

		var parsed struct {
			Candidates []geminiCandidate `json:"candidates"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil {
			return "", fmt.Errorf("failed to parse Gemini response: %w", err)
		}
		if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
			return "", fmt.Errorf("no response from AI")
		}

		parts := parsed.Candidates[0].Content.Parts
		var fc *geminiPart
		for i := range parts {
			if parts[i].FunctionCall != nil {
				fc = &parts[i]
				break
			}
		}

		if fc == nil || fc.FunctionCall == nil || hop == maxToolHops {
			for _, p := range parts {
				if t := strings.TrimSpace(p.Text); t != "" {
					return t, nil
				}
			}
			return "", fmt.Errorf("empty response from AI")
		}

		name := fc.FunctionCall.Name
		tool, ok := registry[name]
		if !ok {
			return "", fmt.Errorf("unknown tool: %s", name)
		}
		result, execErr := tool.Exec(toolCtx, argsToMap(fc.FunctionCall.Args))
		if execErr != nil {
			result = map[string]any{"error": execErr.Error()}
		}

		contents, _ := reqBody["contents"].([]map[string]interface{})
		contents = append(contents,
			map[string]interface{}{
				"role": "model",
				"parts": []map[string]interface{}{
					{"functionCall": map[string]interface{}{
						"name": name,
						"args": argsToMap(fc.FunctionCall.Args),
					}},
				},
			},
			map[string]interface{}{
				"role": "user",
				"parts": []map[string]interface{}{
					{"functionResponse": map[string]interface{}{
						"name":     name,
						"response": result,
					}},
				},
			},
		)
		reqBody["contents"] = contents
		log.Printf("Tool hop %d: executed %s for user %d session %d", hop+1, name, userID, sessionID)
	}

	return "", fmt.Errorf("tool loop exhausted")
}

// postGenerateContent performs the HTTP call with retries (shared by both generate paths).
func (s *Service) postGenerateContent(reqBody map[string]interface{}) ([]byte, error) {
	jsonBody, _ := json.Marshal(reqBody)
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", baseURL, s.model, s.apiKey)

	var (
		body    []byte
		status  int
		lastErr error
	)

	for attempt := 0; attempt <= generateContentMaxRetries; attempt++ {
		resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBody))
		if err != nil {
			lastErr = fmt.Errorf("Gemini API request failed: %w", err)
			if attempt < generateContentMaxRetries {
				time.Sleep(generateContentBaseDelay << attempt)
				continue
			}
			return nil, lastErr
		}
		body, _ = io.ReadAll(resp.Body)
		status = resp.StatusCode
		resp.Body.Close()

		if status == http.StatusOK {
			return body, nil
		}
		if isRetryableGenerateContentStatus(status) && attempt < generateContentMaxRetries {
			time.Sleep(generateContentBaseDelay << attempt)
			continue
		}
		return nil, fmt.Errorf("Gemini API error (status %d): %s", status, string(body))
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Gemini API error (status %d): %s", status, string(body))
	}
	return body, nil
}

// createCache creates an explicit context cache with the Q&A content and system prompt
func (s *Service) createCache(qaContent string, userID uint) (string, error) {
	b64Content := base64.StdEncoding.EncodeToString([]byte(qaContent))

	reqBody := map[string]interface{}{
		"model": fmt.Sprintf("models/%s", s.model),
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]interface{}{
					{
						"inline_data": map[string]string{
							"mime_type": "text/plain",
							"data":      b64Content,
						},
					},
				},
			},
		},
		"systemInstruction": map[string]interface{}{
			"parts": []map[string]string{
				{"text": s.BuildSystemPromptForUser(userID, 0)},
			},
		},
		"displayName": fmt.Sprintf("user_%d_qa_cache", userID),
		"ttl":         "86400s",
	}

	jsonBody, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/cachedContents?key=%s", baseURL, s.apiKey)
	resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("create cache failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Name, nil
}

func (s *Service) deleteCache(cacheName string) {
	url := fmt.Sprintf("%s/%s?key=%s", baseURL, cacheName, s.apiKey)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
	log.Printf("Deleted old cache: %s", cacheName)
}

// ensureStore creates a store if the user doesn't have one yet
func (s *Service) ensureStore(userID uint) (*models.KnowledgeBase, error) {
	var kb models.KnowledgeBase
	result := database.DB.Where("user_id = ?", userID).First(&kb)
	if result.Error == nil && kb.GeminiStoreName != "" {
		return &kb, nil
	}

	storeID, storeName, err := s.createStore(fmt.Sprintf("user_%d_qa_store", userID))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini store: %w", err)
	}

	if result.Error != nil {
		kb = models.KnowledgeBase{
			UserID:          userID,
			GeminiStoreID:   storeID,
			GeminiStoreName: storeName,
		}
		database.DB.Create(&kb)
	} else {
		kb.GeminiStoreID = storeID
		kb.GeminiStoreName = storeName
		database.DB.Save(&kb)
	}

	return &kb, nil
}

func (s *Service) createStore(displayName string) (string, string, error) {
	reqBody := map[string]interface{}{
		"displayName": displayName,
	}
	jsonBody, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/fileSearchStores?key=%s", baseURL, s.apiKey)
	resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("create store failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", err
	}

	return result.DisplayName, result.Name, nil
}

func (s *Service) deleteStore(storeName string) {
	url := fmt.Sprintf("%s/%s?force=true&key=%s", baseURL, storeName, s.apiKey)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		log.Printf("Failed to create delete request for store %s: %v", storeName, err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Failed to delete store %s: %v", storeName, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		log.Printf("Deleted old Gemini store: %s", storeName)
	} else {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Failed to delete store %s (status %d): %s", storeName, resp.StatusCode, string(body))
	}
}

var mimeTypes = map[string]string{
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".xls":  "application/vnd.ms-excel",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".doc":  "application/msword",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".pdf":  "application/pdf",
	".csv":  "text/csv",
	".txt":  "text/plain",
	".json": "application/json",
	".html": "text/html",
}

func (s *Service) uploadToStore(storeName, filePath, displayName string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	ext := filepath.Ext(displayName)
	mimeType, ok := mimeTypes[ext]
	if !ok {
		mimeType = "application/octet-stream"
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	meta := map[string]interface{}{
		"file": map[string]string{
			"displayName": displayName,
		},
	}
	if mimeType != "application/octet-stream" {
		meta["file"].(map[string]string)["mimeType"] = mimeType
	}
	metaJSON, _ := json.Marshal(meta)
	metaPart, _ := writer.CreateFormField("metadata")
	metaPart.Write(metaJSON)

	filePart, _ := writer.CreateFormFile("file", displayName)
	io.Copy(filePart, file)
	writer.Close()

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/upload/v1beta/%s:uploadToFileSearchStore?key=%s", storeName, s.apiKey)

	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("upload failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var opResult struct {
		Name string `json:"name"`
		Done bool   `json:"done"`
	}
	if err := json.Unmarshal(respBody, &opResult); err == nil && opResult.Name != "" && !opResult.Done {
		for i := 0; i < 30; i++ {
			time.Sleep(2 * time.Second)
			opURL := fmt.Sprintf("%s/%s?key=%s", baseURL, opResult.Name, s.apiKey)
			opResp, err := http.Get(opURL)
			if err != nil {
				continue
			}
			opBody, _ := io.ReadAll(opResp.Body)
			opResp.Body.Close()
			json.Unmarshal(opBody, &opResult)
			if opResult.Done {
				break
			}
		}
	}

	return nil
}
