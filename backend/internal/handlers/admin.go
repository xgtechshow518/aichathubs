package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"smart-live-chats/internal/config"
	"smart-live-chats/internal/database"
	"smart-live-chats/internal/gemini"
	"smart-live-chats/internal/middleware"
	"smart-live-chats/internal/models"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	config    *config.Config
	startTime time.Time
}

func NewAdminHandler(cfg *config.Config) *AdminHandler {
	return &AdminHandler{
		config:    cfg,
		startTime: time.Now(),
	}
}

func (h *AdminHandler) Login(c echo.Context) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	reqUser := strings.TrimSpace(req.Username)
	reqPass := strings.TrimSpace(req.Password)
	cfgUser := strings.TrimSpace(h.config.AdminUsername)
	cfgPass := strings.TrimSpace(h.config.AdminPassword)

	if reqUser != cfgUser || reqPass != cfgPass {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid admin credentials")
	}

	token, err := middleware.GenerateAdminToken(h.config)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"token": token,
	})
}

func (h *AdminHandler) GetDashboard(c echo.Context) error {
	var totalUsers int64
	database.DB.Model(&models.User{}).Count(&totalUsers)

	var verifiedUsers int64
	database.DB.Model(&models.User{}).Where("email_verified = ?", true).Count(&verifiedUsers)

	var activeSubscriptions int64
	database.DB.Model(&models.User{}).Where("subscription_status = ?", "active").Count(&activeSubscriptions)

	var trialingUsers int64
	database.DB.Model(&models.User{}).Where("subscription_status = ?", "trialing").Count(&trialingUsers)

	var totalDevices int64
	database.DB.Model(&models.WhatsAppDevice{}).Count(&totalDevices)

	var connectedDevices int64
	database.DB.Model(&models.WhatsAppDevice{}).Where("status = ?", "connected").Count(&connectedDevices)

	var totalChats int64
	database.DB.Model(&models.ChatSession{}).Count(&totalChats)

	var activeChats int64
	database.DB.Model(&models.ChatSession{}).Where("status = ?", "active").Count(&activeChats)

	var totalMessages int64
	database.DB.Model(&models.ChatMessage{}).Count(&totalMessages)

	today := time.Now().Truncate(24 * time.Hour)
	var messagesToday int64
	database.DB.Model(&models.ChatMessage{}).Where("created_at >= ?", today).Count(&messagesToday)

	var totalSubscriptions int64
	database.DB.Model(&models.Subscription{}).Count(&totalSubscriptions)

	var newUsersToday int64
	database.DB.Model(&models.User{}).Where("created_at >= ?", today).Count(&newUsersToday)

	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	var newUsersThisWeek int64
	database.DB.Model(&models.User{}).Where("created_at >= ?", sevenDaysAgo).Count(&newUsersThisWeek)

	// User registration trend (last 30 days)
	type DayCount struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}
	var userTrend []DayCount
	database.DB.Model(&models.User{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= ?", time.Now().AddDate(0, 0, -30)).
		Group("DATE(created_at)").
		Order("date").
		Scan(&userTrend)

	// Subscription plan distribution
	type PlanCount struct {
		Plan  string `json:"plan"`
		Count int64  `json:"count"`
	}
	var planDistribution []PlanCount
	database.DB.Model(&models.User{}).
		Select("subscription_plan as plan, COUNT(*) as count").
		Group("subscription_plan").
		Scan(&planDistribution)

	// Provider distribution
	var providerDistribution []PlanCount
	database.DB.Model(&models.User{}).
		Select("COALESCE(NULLIF(provider, ''), 'email') as plan, COUNT(*) as count").
		Group("provider").
		Scan(&providerDistribution)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"total_users":           totalUsers,
		"verified_users":        verifiedUsers,
		"active_subscriptions":  activeSubscriptions,
		"trialing_users":        trialingUsers,
		"total_devices":         totalDevices,
		"connected_devices":     connectedDevices,
		"total_chats":           totalChats,
		"active_chats":          activeChats,
		"total_messages":        totalMessages,
		"messages_today":        messagesToday,
		"total_subscriptions":   totalSubscriptions,
		"new_users_today":       newUsersToday,
		"new_users_this_week":   newUsersThisWeek,
		"user_trend":            userTrend,
		"plan_distribution":     planDistribution,
		"provider_distribution": providerDistribution,
	})
}

func (h *AdminHandler) ListUsers(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	search := c.QueryParam("search")
	plan := c.QueryParam("plan")
	status := c.QueryParam("status")
	provider := c.QueryParam("provider")

	query := database.DB.Model(&models.User{})

	if search != "" {
		query = query.Where("email ILIKE ? OR name ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if plan != "" {
		query = query.Where("subscription_plan = ?", plan)
	}
	if status != "" {
		query = query.Where("subscription_status = ?", status)
	}
	if provider != "" {
		query = query.Where("provider = ?", provider)
	}

	var total int64
	query.Count(&total)

	var users []models.User
	query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&users)

	// For each user, get device count
	type UserWithDevices struct {
		models.User
		DeviceCount int64 `json:"device_count"`
	}
	var result []UserWithDevices
	for _, u := range users {
		var deviceCount int64
		database.DB.Model(&models.WhatsAppDevice{}).Where("user_id = ?", u.ID).Count(&deviceCount)
		result = append(result, UserWithDevices{User: u, DeviceCount: deviceCount})
	}

	totalPages := (int(total) + pageSize - 1) / pageSize

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users":       result,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}

func (h *AdminHandler) GetUser(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	var devices []models.WhatsAppDevice
	database.DB.Where("user_id = ?", user.ID).Find(&devices)

	var subscriptions []models.Subscription
	database.DB.Where("user_id = ?", user.ID).Order("created_at DESC").Find(&subscriptions)

	var chatCount int64
	database.DB.Model(&models.ChatSession{}).Where("assigned_to_id = ?", user.ID).Count(&chatCount)

	var messageCount int64
	database.DB.Model(&models.ChatMessage{}).Where("sender_id = ?", user.ID).Count(&messageCount)

	var kb models.KnowledgeBase
	hasKB := database.DB.Where("user_id = ?", user.ID).First(&kb).Error == nil

	var qaCount int64
	database.DB.Model(&models.QAItem{}).Where("user_id = ?", user.ID).Count(&qaCount)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user":          user,
		"devices":       devices,
		"subscriptions": subscriptions,
		"chat_count":    chatCount,
		"message_count": messageCount,
		"has_kb":        hasKB,
		"qa_count":      qaCount,
	})
}

func (h *AdminHandler) DeleteUser(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	database.DB.Delete(&user)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "user deleted successfully",
	})
}

func (h *AdminHandler) ListDevices(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	status := c.QueryParam("status")
	search := c.QueryParam("search")

	query := database.DB.Model(&models.WhatsAppDevice{}).Preload("User")

	if status != "" {
		query = query.Where("whatsapp_devices.status = ?", status)
	}
	if search != "" {
		query = query.Joins("JOIN users ON users.id = whatsapp_devices.user_id").
			Where("whatsapp_devices.phone ILIKE ? OR whatsapp_devices.jid ILIKE ? OR users.email ILIKE ?",
				"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	type DeviceWithUser struct {
		models.WhatsAppDevice
		UserEmail string `json:"user_email"`
		UserName  string `json:"user_name"`
	}

	var devices []models.WhatsAppDevice
	query.Order("whatsapp_devices.created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&devices)

	var result []DeviceWithUser
	for _, d := range devices {
		var user models.User
		database.DB.First(&user, d.UserID)
		result = append(result, DeviceWithUser{
			WhatsAppDevice: d,
			UserEmail:      user.Email,
			UserName:       user.Name,
		})
	}

	totalPages := (int(total) + pageSize - 1) / pageSize

	return c.JSON(http.StatusOK, map[string]interface{}{
		"devices":     result,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}

func (h *AdminHandler) ListSubscriptions(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	status := c.QueryParam("status")
	plan := c.QueryParam("plan")

	query := database.DB.Model(&models.Subscription{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if plan != "" {
		query = query.Where("plan = ?", plan)
	}

	var total int64
	query.Count(&total)

	type SubWithUser struct {
		models.Subscription
		UserEmail string `json:"user_email"`
		UserName  string `json:"user_name"`
	}

	var subs []models.Subscription
	query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&subs)

	var result []SubWithUser
	for _, s := range subs {
		var user models.User
		database.DB.First(&user, s.UserID)
		result = append(result, SubWithUser{
			Subscription: s,
			UserEmail:    user.Email,
			UserName:     user.Name,
		})
	}

	totalPages := (int(total) + pageSize - 1) / pageSize

	return c.JSON(http.StatusOK, map[string]interface{}{
		"subscriptions": result,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
		"total_pages":   totalPages,
	})
}

func (h *AdminHandler) GetChatAnalytics(c echo.Context) error {
	type DayCount struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}

	// Messages per day (last 30 days)
	var messagesPerDay []DayCount
	database.DB.Model(&models.ChatMessage{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= ?", time.Now().AddDate(0, 0, -30)).
		Group("DATE(created_at)").
		Order("date").
		Scan(&messagesPerDay)

	// New sessions per day (last 30 days)
	var sessionsPerDay []DayCount
	database.DB.Model(&models.ChatSession{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= ?", time.Now().AddDate(0, 0, -30)).
		Group("DATE(created_at)").
		Order("date").
		Scan(&sessionsPerDay)

	// Platform breakdown
	type PlatformCount struct {
		Platform string `json:"platform"`
		Count    int64  `json:"count"`
	}
	var platformBreakdown []PlatformCount
	database.DB.Model(&models.ChatSession{}).
		Select("COALESCE(NULLIF(platform, ''), 'unknown') as platform, COUNT(*) as count").
		Group("platform").
		Scan(&platformBreakdown)

	// Status breakdown
	type StatusCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var statusBreakdown []StatusCount
	database.DB.Model(&models.ChatSession{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusBreakdown)

	// Message type breakdown
	var messageTypes []StatusCount
	database.DB.Model(&models.ChatMessage{}).
		Select("sender_type as status, COUNT(*) as count").
		Group("sender_type").
		Scan(&messageTypes)

	var totalSessions int64
	database.DB.Model(&models.ChatSession{}).Count(&totalSessions)

	var totalMessages int64
	database.DB.Model(&models.ChatMessage{}).Count(&totalMessages)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"messages_per_day":    messagesPerDay,
		"sessions_per_day":   sessionsPerDay,
		"platform_breakdown": platformBreakdown,
		"status_breakdown":   statusBreakdown,
		"message_types":      messageTypes,
		"total_sessions":     totalSessions,
		"total_messages":     totalMessages,
	})
}

func (h *AdminHandler) GetSystemInfo(c echo.Context) error {
	var userCount int64
	database.DB.Model(&models.User{}).Count(&userCount)

	var subscriptionCount int64
	database.DB.Model(&models.Subscription{}).Count(&subscriptionCount)

	var chatSessionCount int64
	database.DB.Model(&models.ChatSession{}).Count(&chatSessionCount)

	var chatMessageCount int64
	database.DB.Model(&models.ChatMessage{}).Count(&chatMessageCount)

	var deviceCount int64
	database.DB.Model(&models.WhatsAppDevice{}).Count(&deviceCount)

	var kbCount int64
	database.DB.Model(&models.KnowledgeBase{}).Count(&kbCount)

	var qaCount int64
	database.DB.Model(&models.QAItem{}).Count(&qaCount)

	var tagCount int64
	database.DB.Model(&models.Tag{}).Count(&tagCount)

	uptime := time.Since(h.startTime)

	dbStatus := "connected"
	sqlDB, err := database.DB.DB()
	if err != nil || sqlDB.Ping() != nil {
		dbStatus = "disconnected"
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"db_status":     dbStatus,
		"uptime":        uptime.String(),
		"uptime_seconds": int(uptime.Seconds()),
		"tables": map[string]int64{
			"users":            userCount,
			"subscriptions":    subscriptionCount,
			"chat_sessions":    chatSessionCount,
			"chat_messages":    chatMessageCount,
			"whatsapp_devices": deviceCount,
			"knowledge_bases":  kbCount,
			"qa_items":         qaCount,
			"tags":             tagCount,
		},
	})
}

// GetBotPrompt returns the globally managed AI system prompt.
// Response:
//   - value:   effective prompt value (custom if set, otherwise the env default)
//   - is_custom: true when an admin-managed prompt exists in platform_settings
//   - default: the env-level fallback, for reference / reset
//   - updated_at / updated_by: metadata when a custom prompt exists
func (h *AdminHandler) GetBotPrompt(c echo.Context) error {
	var setting models.PlatformSetting
	err := database.DB.Where("key = ?", gemini.PlatformSystemPromptKey).First(&setting).Error

	resp := map[string]interface{}{
		"default":   h.config.GeminiSystemPrompt,
		"is_custom": false,
		"value":     h.config.GeminiSystemPrompt,
	}

	if err == nil {
		resp["is_custom"] = strings.TrimSpace(setting.Value) != ""
		if setting.Value != "" {
			resp["value"] = setting.Value
		}
		resp["updated_at"] = setting.UpdatedAt
		resp["updated_by"] = setting.UpdatedBy
	}

	return c.JSON(http.StatusOK, resp)
}

// UpdateBotPrompt sets (or clears) the globally managed AI system prompt.
// Passing an empty string clears the custom prompt so the env default is used again.
// On success, the in-memory prompt cache and all users' Gemini context caches
// are invalidated so the new prompt takes effect on the next message.
func (h *AdminHandler) UpdateBotPrompt(c echo.Context) error {
	var req struct {
		Value string `json:"value"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	value := strings.TrimSpace(req.Value)

	var setting models.PlatformSetting
	err := database.DB.Where("key = ?", gemini.PlatformSystemPromptKey).First(&setting).Error
	if err != nil {
		setting = models.PlatformSetting{
			Key:       gemini.PlatformSystemPromptKey,
			Value:     value,
			UpdatedBy: h.config.AdminUsername,
		}
		if err := database.DB.Create(&setting).Error; err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to save prompt")
		}
	} else {
		setting.Value = value
		setting.UpdatedBy = h.config.AdminUsername
		if err := database.DB.Save(&setting).Error; err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to save prompt")
		}
	}

	if gemini.GlobalService != nil {
		gemini.GlobalService.InvalidateSystemPromptCache()
		// Run cache invalidation off the request goroutine: it may delete many
		// Gemini cache resources and we don't want to block the admin UI.
		go gemini.GlobalService.InvalidateAllContextCaches()
	}

	effective := value
	if effective == "" {
		effective = h.config.GeminiSystemPrompt
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"value":      effective,
		"is_custom":  value != "",
		"default":    h.config.GeminiSystemPrompt,
		"updated_at": setting.UpdatedAt,
		"updated_by": setting.UpdatedBy,
	})
}

// GetAIReplyDelay returns the admin-managed AI reply delay range (seconds).
// Response includes the effective min/max, whether a custom value is set, and
// the defaults so the UI can offer a reset.
func (h *AdminHandler) GetAIReplyDelay(c echo.Context) error {
	min, max := gemini.GetAIReplyDelayRange()

	isCustom := false
	var minSetting, maxSetting models.PlatformSetting
	if err := database.DB.Where("key = ?", gemini.PlatformAIReplyDelayMinKey).First(&minSetting).Error; err == nil {
		isCustom = true
	}
	if err := database.DB.Where("key = ?", gemini.PlatformAIReplyDelayMaxKey).First(&maxSetting).Error; err == nil {
		isCustom = true
	}

	resp := map[string]interface{}{
		"min_seconds":         min,
		"max_seconds":         max,
		"default_min_seconds": gemini.DefaultAIReplyDelayMin,
		"default_max_seconds": gemini.DefaultAIReplyDelayMax,
		"is_custom":           isCustom,
	}
	if maxSetting.UpdatedAt.After(minSetting.UpdatedAt) {
		resp["updated_at"] = maxSetting.UpdatedAt
		resp["updated_by"] = maxSetting.UpdatedBy
	} else if !minSetting.UpdatedAt.IsZero() {
		resp["updated_at"] = minSetting.UpdatedAt
		resp["updated_by"] = minSetting.UpdatedBy
	}

	return c.JSON(http.StatusOK, resp)
}

// UpdateAIReplyDelay sets the admin-managed AI reply delay range (seconds).
// Both values must be non-negative integers with min <= max. Passing an upper
// bound of ~600s is enforced to guard against typos that would hang replies.
func (h *AdminHandler) UpdateAIReplyDelay(c echo.Context) error {
	var req struct {
		MinSeconds *int `json:"min_seconds"`
		MaxSeconds *int `json:"max_seconds"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.MinSeconds == nil || req.MaxSeconds == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "min_seconds and max_seconds are required")
	}
	min := *req.MinSeconds
	max := *req.MaxSeconds
	if min < 0 || max < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "delay values must be non-negative")
	}
	if min > max {
		return echo.NewHTTPError(http.StatusBadRequest, "min_seconds must be <= max_seconds")
	}
	// Upper bound guards against accidental multi-hour waits.
	if max > 600 {
		return echo.NewHTTPError(http.StatusBadRequest, "max_seconds must be <= 600")
	}

	if err := upsertPlatformSetting(gemini.PlatformAIReplyDelayMinKey, strconv.Itoa(min), h.config.AdminUsername); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save min delay")
	}
	if err := upsertPlatformSetting(gemini.PlatformAIReplyDelayMaxKey, strconv.Itoa(max), h.config.AdminUsername); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save max delay")
	}

	gemini.InvalidateAIReplyDelayCache()

	return c.JSON(http.StatusOK, map[string]interface{}{
		"min_seconds":         min,
		"max_seconds":         max,
		"default_min_seconds": gemini.DefaultAIReplyDelayMin,
		"default_max_seconds": gemini.DefaultAIReplyDelayMax,
		"is_custom":           true,
	})
}

// ResetAIReplyDelay clears the admin-managed delay range so the defaults are
// used again. Implemented as a dedicated endpoint so the UI can offer a
// single-click "Reset to defaults" action without having to remember the
// exact default values.
func (h *AdminHandler) ResetAIReplyDelay(c echo.Context) error {
	database.DB.Where("key = ?", gemini.PlatformAIReplyDelayMinKey).Delete(&models.PlatformSetting{})
	database.DB.Where("key = ?", gemini.PlatformAIReplyDelayMaxKey).Delete(&models.PlatformSetting{})
	gemini.InvalidateAIReplyDelayCache()

	return c.JSON(http.StatusOK, map[string]interface{}{
		"min_seconds":         gemini.DefaultAIReplyDelayMin,
		"max_seconds":         gemini.DefaultAIReplyDelayMax,
		"default_min_seconds": gemini.DefaultAIReplyDelayMin,
		"default_max_seconds": gemini.DefaultAIReplyDelayMax,
		"is_custom":           false,
	})
}

func upsertPlatformSetting(key, value, updatedBy string) error {
	var setting models.PlatformSetting
	err := database.DB.Where("key = ?", key).First(&setting).Error
	if err != nil {
		setting = models.PlatformSetting{
			Key:       key,
			Value:     value,
			UpdatedBy: updatedBy,
		}
		return database.DB.Create(&setting).Error
	}
	setting.Value = value
	setting.UpdatedBy = updatedBy
	return database.DB.Save(&setting).Error
}

// UpdateUser lets an admin override user fields that normally require a
// Stripe round-trip. Only the fields we explicitly allow here are editable;
// email/provider/password/ids are read-only via this endpoint.
func (h *AdminHandler) UpdateUser(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	// All fields are optional so partial updates are possible.
	var req struct {
		Name               *string `json:"name"`
		SubscriptionPlan   *string `json:"subscription_plan"`
		SubscriptionStatus *string `json:"subscription_status"`
		MaxDevices         *int    `json:"max_devices"`
		TrialEndsAt        *string `json:"trial_ends_at"` // RFC3339 or "" to clear
		EmailVerified      *bool   `json:"email_verified"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Name != nil {
		user.Name = strings.TrimSpace(*req.Name)
	}
	if req.SubscriptionPlan != nil {
		user.SubscriptionPlan = strings.TrimSpace(*req.SubscriptionPlan)
	}
	if req.SubscriptionStatus != nil {
		user.SubscriptionStatus = strings.TrimSpace(*req.SubscriptionStatus)
	}
	if req.MaxDevices != nil {
		if *req.MaxDevices < 0 || *req.MaxDevices > 1000 {
			return echo.NewHTTPError(http.StatusBadRequest, "max_devices out of range")
		}
		user.MaxDevices = *req.MaxDevices
	}
	if req.TrialEndsAt != nil {
		s := strings.TrimSpace(*req.TrialEndsAt)
		if s == "" {
			user.TrialEndsAt = nil
		} else {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "trial_ends_at must be RFC3339")
			}
			user.TrialEndsAt = &t
		}
	}
	if req.EmailVerified != nil {
		user.EmailVerified = *req.EmailVerified
	}

	if err := database.DB.Save(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update user")
	}

	return c.JSON(http.StatusOK, user)
}

// ResetUserPassword sets a new password for the user. If the request body
// contains a non-empty password the admin's chosen value is used; otherwise
// the server generates a 12-char random password and returns it (the admin
// is responsible for sharing it with the user out-of-band).
func (h *AdminHandler) ResetUserPassword(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	var req struct {
		Password string `json:"password"`
	}
	_ = c.Bind(&req)

	password := strings.TrimSpace(req.Password)
	generated := false
	if password == "" {
		p, err := generateRandomPassword(12)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate password")
		}
		password = p
		generated = true
	}
	if len(password) < 8 {
		return echo.NewHTTPError(http.StatusBadRequest, "password must be at least 8 characters")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to hash password")
	}
	user.Password = string(hashed)
	// Ensure the account is usable via email/password after a reset, even if
	// it was originally created via OAuth.
	if user.Provider == "" {
		user.Provider = "email"
	}
	if err := database.DB.Save(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save password")
	}

	resp := map[string]interface{}{
		"message":   "password reset",
		"generated": generated,
	}
	if generated {
		resp["password"] = password
	}
	return c.JSON(http.StatusOK, resp)
}

// SuspendUser blocks a user from logging in and invalidates any existing
// sessions (JWTMiddleware rejects tokens for suspended users).
func (h *AdminHandler) SuspendUser(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.Bind(&req)

	now := time.Now()
	user.Suspended = true
	user.SuspendedAt = &now
	user.SuspendedReason = strings.TrimSpace(req.Reason)
	if err := database.DB.Save(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to suspend user")
	}

	return c.JSON(http.StatusOK, user)
}

// UnsuspendUser restores login access.
func (h *AdminHandler) UnsuspendUser(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	user.Suspended = false
	user.SuspendedAt = nil
	user.SuspendedReason = ""
	if err := database.DB.Save(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to unsuspend user")
	}

	return c.JSON(http.StatusOK, user)
}

// GetUserKnowledge returns the knowledge base config for a specific user.
// If the user has never configured a KB yet, an empty shell is returned so
// the admin UI can still render the editor.
func (h *AdminHandler) GetUserKnowledge(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	var kb models.KnowledgeBase
	if err := database.DB.Where("user_id = ?", user.ID).First(&kb).Error; err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"user_id":            user.ID,
			"auto_reply_enabled": false,
			"system_prompt":      "",
			"synced":             false,
		})
	}

	var qaCount int64
	database.DB.Model(&models.QAItem{}).Where("user_id = ?", user.ID).Count(&qaCount)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":                 kb.ID,
		"user_id":            kb.UserID,
		"auto_reply_enabled": kb.AutoReplyEnabled,
		"system_prompt":      kb.SystemPrompt,
		"last_synced_at":     kb.LastSyncedAt,
		"qa_count":           qaCount,
	})
}

// UpdateUserKnowledge lets the admin edit a user's KB config on their behalf.
func (h *AdminHandler) UpdateUserKnowledge(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	var req struct {
		AutoReplyEnabled *bool   `json:"auto_reply_enabled"`
		SystemPrompt     *string `json:"system_prompt"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	var kb models.KnowledgeBase
	if err := database.DB.Where("user_id = ?", user.ID).First(&kb).Error; err != nil {
		kb = models.KnowledgeBase{UserID: user.ID}
		if err := database.DB.Create(&kb).Error; err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create knowledge base")
		}
	}

	promptChanged := false
	if req.AutoReplyEnabled != nil {
		kb.AutoReplyEnabled = *req.AutoReplyEnabled
	}
	if req.SystemPrompt != nil && *req.SystemPrompt != kb.SystemPrompt {
		kb.SystemPrompt = *req.SystemPrompt
		promptChanged = true
	}

	if err := database.DB.Save(&kb).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save knowledge base")
	}

	// A changed per-user prompt is baked into the cached Gemini context, so drop
	// the cache to make the new prompt take effect on the next message.
	if promptChanged && gemini.GlobalService != nil {
		gemini.GlobalService.InvalidateUserContextCache(user.ID)
	}

	return c.JSON(http.StatusOK, kb)
}

// ListDeviceChats returns chat sessions attached to a given WhatsApp device,
// used by the admin UI to drill down from a user's device to their customer
// conversations.
func (h *AdminHandler) ListDeviceChats(c echo.Context) error {
	deviceID, err := strconv.ParseUint(c.Param("deviceId"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid device id")
	}

	var device models.WhatsAppDevice
	if err := database.DB.First(&device, deviceID).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "device not found")
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	search := strings.TrimSpace(c.QueryParam("search"))

	query := database.DB.Model(&models.ChatSession{}).Where("device_id = ?", device.ID)
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("customer_name ILIKE ? OR customer_phone ILIKE ? OR last_message ILIKE ?", like, like, like)
	}

	var total int64
	query.Count(&total)

	var sessions []models.ChatSession
	query.Order("last_message_at DESC NULLS LAST, created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&sessions)

	totalPages := (int(total) + pageSize - 1) / pageSize

	return c.JSON(http.StatusOK, map[string]interface{}{
		"device":      device,
		"sessions":    sessions,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}

// GetAdminChatMessages returns messages for a session (read-only).
// Unlike the user-facing /chats/:id/messages, this is not scoped to a user
// and is gated by the admin middleware.
func (h *AdminHandler) GetAdminChatMessages(c echo.Context) error {
	sessionID, err := strconv.ParseUint(c.Param("sessionId"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid session id")
	}

	var session models.ChatSession
	if err := database.DB.First(&session, sessionID).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "chat session not found")
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	var total int64
	database.DB.Model(&models.ChatMessage{}).Where("session_id = ?", sessionID).Count(&total)

	messages := make([]models.ChatMessage, 0)
	database.DB.Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&messages)

	totalPages := (int(total) + pageSize - 1) / pageSize

	return c.JSON(http.StatusOK, map[string]interface{}{
		"session":     session,
		"messages":    messages,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}

// generateRandomPassword produces a URL-safe password of roughly the requested
// length. We use base64 and trim because it's fast, cryptographically random,
// and contains no ambiguous characters we need to worry about for copy-paste.
func generateRandomPassword(n int) (string, error) {
	if n < 8 {
		n = 8
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	s := base64.RawURLEncoding.EncodeToString(buf)
	if len(s) > n {
		s = s[:n]
	}
	return s, nil
}
