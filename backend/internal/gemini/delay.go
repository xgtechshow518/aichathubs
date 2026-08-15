package gemini

import (
	"math/rand"
	"strconv"
	"sync"
	"time"

	"smart-live-chats/internal/database"
	"smart-live-chats/internal/models"
)

// Keys used in the platform_settings table for the admin-configurable AI
// reply delay range. Values are stored as seconds (integer strings).
const (
	PlatformAIReplyDelayMinKey = "ai_reply_delay_min_seconds"
	PlatformAIReplyDelayMaxKey = "ai_reply_delay_max_seconds"
)

// Default delay range when no admin override is set. Kept in sync with the
// product spec (2–30s) so fresh installs behave sensibly without any config.
const (
	DefaultAIReplyDelayMin = 2
	DefaultAIReplyDelayMax = 30
)

// delayCacheTTL mirrors systemPromptCacheTTL: short enough that admin edits
// land quickly, long enough to avoid a DB hit on every auto-reply.
const delayCacheTTL = 30 * time.Second

var (
	delayMu       sync.RWMutex
	delayCacheMin int
	delayCacheMax int
	delayCacheExp time.Time
)

// GetAIReplyDelayRange returns the currently configured (min, max) delay in
// seconds, falling back to the defaults when the admin has not configured a
// value or the stored value is invalid. Values are clamped so min <= max and
// both are non-negative.
func GetAIReplyDelayRange() (int, int) {
	delayMu.RLock()
	if !delayCacheExp.IsZero() && time.Now().Before(delayCacheExp) {
		min, max := delayCacheMin, delayCacheMax
		delayMu.RUnlock()
		return min, max
	}
	delayMu.RUnlock()

	delayMu.Lock()
	defer delayMu.Unlock()
	if !delayCacheExp.IsZero() && time.Now().Before(delayCacheExp) {
		return delayCacheMin, delayCacheMax
	}

	min := DefaultAIReplyDelayMin
	max := DefaultAIReplyDelayMax

	if v, ok := readIntSetting(PlatformAIReplyDelayMinKey); ok {
		min = v
	}
	if v, ok := readIntSetting(PlatformAIReplyDelayMaxKey); ok {
		max = v
	}

	if min < 0 {
		min = 0
	}
	if max < min {
		max = min
	}

	delayCacheMin = min
	delayCacheMax = max
	delayCacheExp = time.Now().Add(delayCacheTTL)
	return min, max
}

// InvalidateAIReplyDelayCache forces the next GetAIReplyDelayRange call to
// re-read from the database. Call this right after an admin updates the range.
func InvalidateAIReplyDelayCache() {
	delayMu.Lock()
	delayCacheExp = time.Time{}
	delayMu.Unlock()
}

// PickAIReplyDelay returns a random delay duration within the configured
// range. A dedicated helper keeps call sites readable and ensures the same
// clamping / randomization logic everywhere.
func PickAIReplyDelay() time.Duration {
	min, max := GetAIReplyDelayRange()
	if max <= min {
		return time.Duration(min) * time.Second
	}
	// rand.Intn(n) returns [0, n); +1 to make max inclusive.
	return time.Duration(min+rand.Intn(max-min+1)) * time.Second
}

func readIntSetting(key string) (int, bool) {
	var setting models.PlatformSetting
	if err := database.DB.Where("key = ?", key).First(&setting).Error; err != nil {
		return 0, false
	}
	v, err := strconv.Atoi(setting.Value)
	if err != nil {
		return 0, false
	}
	return v, true
}
