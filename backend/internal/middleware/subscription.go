package middleware

import (
	"net/http"
	"time"

	"smart-live-chats/internal/config"

	"github.com/labstack/echo/v4"
)

// SubscriptionMiddleware checks if the user has an active subscription or valid trial.
// When billing is disabled (free self-hosted mode) it lets every authenticated
// user through.
func SubscriptionMiddleware(cfg *config.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := GetUserFromContext(c)
			if user == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
			}

			// Free mode: no billing, no gate.
			if cfg == nil || !cfg.BillingEnabled {
				return next(c)
			}

			// Allow if local trial is still valid
			if user.TrialEndsAt != nil && time.Now().Before(*user.TrialEndsAt) {
				return next(c)
			}

			// Allow if Stripe subscription is active
			if user.SubscriptionStatus == "active" {
				return next(c)
			}

			// Allow if Stripe-managed trial is active (trialing status + valid end date)
			if user.SubscriptionStatus == "trialing" && user.TrialEndsAt != nil && time.Now().Before(*user.TrialEndsAt) {
				return next(c)
			}

			return echo.NewHTTPError(http.StatusPaymentRequired, "Your trial has expired. Please subscribe to continue using the service.")
		}
	}
}
