package handlers

import (
	"net/http"

	"smart-live-chats/internal/config"

	"github.com/labstack/echo/v4"
)

// MetaHandler serves public, non-secret server metadata.
type MetaHandler struct {
	config *config.Config
}

func NewMetaHandler(cfg *config.Config) *MetaHandler {
	return &MetaHandler{config: cfg}
}

// PublicConfig exposes feature flags the frontend uses to render conditionally
// (which login buttons to show, whether email verification / billing are on).
// It intentionally returns only booleans — never any keys or secrets.
func (h *MetaHandler) PublicConfig(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]bool{
		"googleAuth":        h.config.GoogleOAuthConfigured(),
		"facebookAuth":      h.config.FacebookOAuthConfigured(),
		"emailVerification": h.config.SMTPConfigured(),
		"billingEnabled":    h.config.BillingEnabled,
		"aiEnabled":         h.config.GeminiConfigured(),
	})
}
