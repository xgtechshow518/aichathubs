package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"smart-live-chats/internal/config"
	"smart-live-chats/internal/database"
	"smart-live-chats/internal/middleware"
	"smart-live-chats/internal/models"

	"github.com/labstack/echo/v4"
	"fmt"

	"github.com/stripe/stripe-go/v82"
	billingSession "github.com/stripe/stripe-go/v82/billingportal/session"
	checkoutSession "github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/customer"
	stripeSub "github.com/stripe/stripe-go/v82/subscription"
	"github.com/stripe/stripe-go/v82/webhook"
)

type BillingHandler struct {
	config *config.Config
}

func NewBillingHandler(cfg *config.Config) *BillingHandler {
	stripe.Key = cfg.StripeSecretKey
	return &BillingHandler{config: cfg}
}

// billingActive reports whether Stripe billing is both switched on and configured.
func (h *BillingHandler) billingActive() bool {
	return h.config.BillingEnabled && h.config.StripeConfigured()
}

func (h *BillingHandler) CreateCheckout(c echo.Context) error {
	if !h.billingActive() {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "billing is not enabled on this server")
	}
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var req struct {
		Quantity int64 `json:"quantity"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if req.Quantity < 1 {
		req.Quantity = 1
	}

	priceID := h.config.StripePriceStarter
	if priceID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "pricing not configured")
	}

	customerID, err := h.ensureStripeCustomer(user)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create customer")
	}

	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(req.Quantity),
			},
		},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			TrialPeriodDays: stripe.Int64(7),
		},
		SuccessURL: stripe.String(h.config.FrontendURL + "/dashboard?payment=success"),
		CancelURL:  stripe.String(h.config.FrontendURL + "/dashboard?payment=cancelled"),
	}

	session, err := checkoutSession.New(params)
	if err != nil {
		log.Printf("Stripe checkout error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create checkout session")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"checkout_url": session.URL,
		"session_id":   session.ID,
	})
}

func (h *BillingHandler) CreatePortal(c echo.Context) error {
	if !h.billingActive() {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "billing is not enabled on this server")
	}
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	if user.StripeCustomerID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "no billing account found")
	}

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(user.StripeCustomerID),
		ReturnURL: stripe.String(h.config.FrontendURL + "/settings"),
	}

	session, err := billingSession.New(params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create portal session")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"portal_url": session.URL,
	})
}

func (h *BillingHandler) UpdateQuantity(c echo.Context) error {
	if !h.billingActive() {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "billing is not enabled on this server")
	}
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var req struct {
		Quantity int64 `json:"quantity"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if req.Quantity < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "quantity must be at least 1")
	}

	if user.StripeCustomerID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "no billing account found")
	}

	// Find the active subscription from Stripe by customer ID
	sub, err := h.findActiveSubscription(user.StripeCustomerID)
	if err != nil {
		log.Printf("Failed to find subscription for user %d: %v", user.ID, err)
		return echo.NewHTTPError(http.StatusBadRequest, "no active subscription found")
	}

	if len(sub.Items.Data) == 0 {
		return echo.NewHTTPError(http.StatusInternalServerError, "no subscription items found")
	}

	itemID := sub.Items.Data[0].ID

	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:       stripe.String(itemID),
				Quantity: stripe.Int64(req.Quantity),
			},
		},
		ProrationBehavior: stripe.String("create_prorations"),
	}

	updated, err := stripeSub.Update(sub.ID, params)
	if err != nil {
		log.Printf("Failed to update subscription quantity: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update quantity")
	}

	newQuantity := int(req.Quantity)
	if len(updated.Items.Data) > 0 {
		newQuantity = int(updated.Items.Data[0].Quantity)
	}

	user.MaxDevices = newQuantity
	user.SubscriptionPlan = "active"
	user.SubscriptionStatus = string(updated.Status)
	database.DB.Save(user)

	log.Printf("Updated subscription quantity for user %d: %d devices", user.ID, newQuantity)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"max_devices": newQuantity,
		"status":      string(updated.Status),
		"message":     "Subscription updated successfully. Proration applied.",
	})
}

func (h *BillingHandler) findActiveSubscription(customerID string) (*stripe.Subscription, error) {
	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerID),
		Status:   stripe.String("active"),
	}
	params.Filters.AddFilter("limit", "", "1")

	i := stripeSub.List(params)
	if i.Next() {
		return i.Subscription(), nil
	}

	// Try trialing
	params.Status = stripe.String("trialing")
	i = stripeSub.List(params)
	if i.Next() {
		return i.Subscription(), nil
	}

	return nil, fmt.Errorf("no active subscription found for customer %s", customerID)
}

func (h *BillingHandler) GetSubscription(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var sub models.Subscription
	hasSub := database.DB.Where("user_id = ?", user.ID).Order("created_at DESC").First(&sub).Error == nil

	trialActive := false
	trialDaysLeft := 0
	if user.TrialEndsAt != nil && time.Now().Before(*user.TrialEndsAt) {
		trialActive = true
		trialDaysLeft = int(time.Until(*user.TrialEndsAt).Hours() / 24)
	}

	result := map[string]interface{}{
		"plan":          user.SubscriptionPlan,
		"status":        user.SubscriptionStatus,
		"max_devices":   user.MaxDevices,
		"trial_active":  trialActive,
		"trial_days_left": trialDaysLeft,
		"trial_ends_at": user.TrialEndsAt,
	}

	if hasSub {
		result["subscription"] = sub
	}

	return c.JSON(http.StatusOK, result)
}

// SyncSubscription pulls the latest subscription from Stripe and updates the local DB.
// This is the fallback for when webhooks can't reach the server (e.g., localhost dev).
func (h *BillingHandler) SyncSubscription(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	if user.StripeCustomerID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "no billing account found")
	}

	sub, err := h.findActiveSubscription(user.StripeCustomerID)
	if err != nil {
		log.Printf("No active subscription found for user %d: %v", user.ID, err)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"synced":      false,
			"max_devices": user.MaxDevices,
			"message":     "no active subscription found",
		})
	}

	h.handleSubscriptionUpdate(sub)

	// Reload user from DB to return fresh data
	database.DB.First(user, user.ID)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"synced":      true,
		"max_devices": user.MaxDevices,
		"plan":        user.SubscriptionPlan,
		"status":      user.SubscriptionStatus,
	})
}

func (h *BillingHandler) HandleWebhook(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to read body")
	}

	event, err := webhook.ConstructEvent(body, c.Request().Header.Get("Stripe-Signature"), h.config.StripeWebhookSecret)
	if err != nil {
		log.Printf("Stripe webhook signature error: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid signature")
	}

	switch event.Type {
	case "customer.subscription.created", "customer.subscription.updated":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			log.Printf("Failed to parse subscription: %v", err)
			break
		}
		h.handleSubscriptionUpdate(&sub)

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			log.Printf("Failed to parse subscription: %v", err)
			break
		}
		h.handleSubscriptionCancelled(&sub)

	case "checkout.session.completed":
		log.Println("Checkout completed")
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *BillingHandler) handleSubscriptionUpdate(sub *stripe.Subscription) {
	var user models.User
	if err := database.DB.Where("stripe_customer_id = ?", sub.Customer.ID).First(&user).Error; err != nil {
		log.Printf("User not found for Stripe customer %s", sub.Customer.ID)
		return
	}

	// Get quantity from subscription items (= number of devices purchased)
	maxDevices := 1
	if len(sub.Items.Data) > 0 {
		maxDevices = int(sub.Items.Data[0].Quantity)
	}

	plan := "active"

	user.SubscriptionPlan = plan
	user.SubscriptionStatus = string(sub.Status)
	user.MaxDevices = maxDevices
	database.DB.Save(&user)

	periodStart := time.Unix(sub.StartDate, 0)
	periodEnd := time.Unix(sub.BillingCycleAnchor, 0)

	var dbSub models.Subscription
	result := database.DB.Where("stripe_subscription_id = ?", sub.ID).First(&dbSub)
	if result.Error != nil {
		dbSub = models.Subscription{
			UserID:               user.ID,
			StripeSubscriptionID: sub.ID,
			Plan:                 plan,
			Status:               string(sub.Status),
			CurrentPeriodStart:   &periodStart,
			CurrentPeriodEnd:     &periodEnd,
			CancelAtPeriodEnd:    sub.CancelAtPeriodEnd,
		}
		if len(sub.Items.Data) > 0 {
			dbSub.StripePriceID = sub.Items.Data[0].Price.ID
		}
		database.DB.Create(&dbSub)
	} else {
		dbSub.Plan = plan
		dbSub.Status = string(sub.Status)
		dbSub.CurrentPeriodStart = &periodStart
		dbSub.CurrentPeriodEnd = &periodEnd
		dbSub.CancelAtPeriodEnd = sub.CancelAtPeriodEnd
		database.DB.Save(&dbSub)
	}

	log.Printf("Subscription updated for user %d: plan=%s status=%s", user.ID, plan, sub.Status)
}

func (h *BillingHandler) handleSubscriptionCancelled(sub *stripe.Subscription) {
	var user models.User
	if err := database.DB.Where("stripe_customer_id = ?", sub.Customer.ID).First(&user).Error; err != nil {
		return
	}

	user.SubscriptionPlan = "cancelled"
	user.SubscriptionStatus = "cancelled"
	user.MaxDevices = 0
	database.DB.Save(&user)

	database.DB.Model(&models.Subscription{}).Where("stripe_subscription_id = ?", sub.ID).
		Update("status", "cancelled")

	log.Printf("Subscription cancelled for user %d", user.ID)
}

func (h *BillingHandler) ensureStripeCustomer(user *models.User) (string, error) {
	if user.StripeCustomerID != "" {
		return user.StripeCustomerID, nil
	}

	params := &stripe.CustomerParams{
		Email: stripe.String(user.Email),
		Name:  stripe.String(user.Name),
		Metadata: map[string]string{
			"user_id": string(rune(user.ID + '0')),
		},
	}

	cust, err := customer.New(params)
	if err != nil {
		return "", err
	}

	user.StripeCustomerID = cust.ID
	database.DB.Save(user)

	return cust.ID, nil
}

// SetTrialForNewUser sets the 7-day trial for newly registered users
func SetTrialForNewUser(user *models.User) {
	trialEnd := time.Now().Add(7 * 24 * time.Hour)
	user.TrialEndsAt = &trialEnd
	user.SubscriptionPlan = "trial"
	user.SubscriptionStatus = "trialing"
	user.MaxDevices = 1
	database.DB.Save(user)
}
