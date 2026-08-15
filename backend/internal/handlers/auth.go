package handlers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"smart-live-chats/internal/config"
	"smart-live-chats/internal/database"
	"smart-live-chats/internal/email"
	"smart-live-chats/internal/middleware"
	"smart-live-chats/internal/models"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/google"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	config         *config.Config
	googleConfig   *oauth2.Config
	facebookConfig *oauth2.Config
	emailService   *email.Service
}

// GoogleUserInfo represents the user info from Google
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// FacebookUserInfo represents the user info from Facebook
type FacebookUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	} `json:"picture"`
}

// TokenRequest represents a token exchange request from frontend
type TokenRequest struct {
	Code        string `json:"code"`
	RedirectURI string `json:"redirect_uri"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// LoginRequest represents an email/password login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// VerifyEmailRequest represents a verification code submission
type VerifyEmailRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// ResendCodeRequest represents a request to resend verification code
type ResendCodeRequest struct {
	Email string `json:"email"`
}

// AuthResponse represents the response after successful authentication
type AuthResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(cfg *config.Config) *AuthHandler {
	googleConfig := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes:       []string{"email", "profile"},
		Endpoint:     google.Endpoint,
	}

	facebookConfig := &oauth2.Config{
		ClientID:     cfg.FacebookClientID,
		ClientSecret: cfg.FacebookClientSecret,
		RedirectURL:  cfg.FacebookRedirectURL,
		Scopes:       []string{"email", "public_profile"},
		Endpoint:     facebook.Endpoint,
	}

	return &AuthHandler{
		config:         cfg,
		googleConfig:   googleConfig,
		facebookConfig: facebookConfig,
		emailService:   email.NewService(cfg),
	}
}

// GoogleLogin initiates Google OAuth flow
func (h *AuthHandler) GoogleLogin(c echo.Context) error {
	if !h.config.GoogleOAuthConfigured() {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Google login is not configured on this server")
	}
	url := h.googleConfig.AuthCodeURL("state", oauth2.AccessTypeOffline)
	return c.JSON(http.StatusOK, map[string]string{"url": url})
}

// GoogleCallback handles the Google OAuth callback
func (h *AuthHandler) GoogleCallback(c echo.Context) error {
	if !h.config.GoogleOAuthConfigured() {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Google login is not configured on this server")
	}
	var req TokenRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// If redirect_uri is provided, use it
	googleConfig := *h.googleConfig
	if req.RedirectURI != "" {
		googleConfig.RedirectURL = req.RedirectURI
	}

	// Exchange code for token
	token, err := googleConfig.Exchange(context.Background(), req.Code)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("failed to exchange code: %v", err))
	}

	// Get user info
	client := googleConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user info")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to read user info")
	}

	var googleUser GoogleUserInfo
	if err := json.Unmarshal(body, &googleUser); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to parse user info")
	}

	// Find or create user
	user, err := h.findOrCreateUser(googleUser.Email, googleUser.Name, googleUser.Picture, "google", googleUser.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
	}

	if user.Suspended {
		return echo.NewHTTPError(http.StatusForbidden, "this account has been suspended")
	}

	// Generate JWT token
	jwtToken, err := middleware.GenerateToken(user, h.config)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	return c.JSON(http.StatusOK, AuthResponse{
		Token: jwtToken,
		User:  user,
	})
}

// FacebookLogin initiates Facebook OAuth flow
func (h *AuthHandler) FacebookLogin(c echo.Context) error {
	if !h.config.FacebookOAuthConfigured() {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Facebook login is not configured on this server")
	}
	url := h.facebookConfig.AuthCodeURL("state")
	return c.JSON(http.StatusOK, map[string]string{"url": url})
}

// FacebookCallback handles the Facebook OAuth callback
func (h *AuthHandler) FacebookCallback(c echo.Context) error {
	if !h.config.FacebookOAuthConfigured() {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Facebook login is not configured on this server")
	}
	var req TokenRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// If redirect_uri is provided, use it
	facebookConfig := *h.facebookConfig
	if req.RedirectURI != "" {
		facebookConfig.RedirectURL = req.RedirectURI
	}

	// Exchange code for token
	token, err := facebookConfig.Exchange(context.Background(), req.Code)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("failed to exchange code: %v", err))
	}

	// Get user info
	client := facebookConfig.Client(context.Background(), token)
	resp, err := client.Get("https://graph.facebook.com/me?fields=id,name,email,picture")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user info")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to read user info")
	}

	var fbUser FacebookUserInfo
	if err := json.Unmarshal(body, &fbUser); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to parse user info")
	}

	// Find or create user
	user, err := h.findOrCreateUser(fbUser.Email, fbUser.Name, fbUser.Picture.Data.URL, "facebook", fbUser.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
	}

	if user.Suspended {
		return echo.NewHTTPError(http.StatusForbidden, "this account has been suspended")
	}

	// Generate JWT token
	jwtToken, err := middleware.GenerateToken(user, h.config)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	return c.JSON(http.StatusOK, AuthResponse{
		Token: jwtToken,
		User:  user,
	})
}

// GetMe returns the current authenticated user
func (h *AuthHandler) GetMe(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not found")
	}
	return c.JSON(http.StatusOK, user)
}

// Register creates a new user with email/password and sends a verification code
func (h *AuthHandler) Register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Name = strings.TrimSpace(req.Name)

	if req.Email == "" || req.Password == "" || req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email, password and name are required")
	}
	if len(req.Password) < 8 {
		return echo.NewHTTPError(http.StatusBadRequest, "password must be at least 8 characters")
	}

	// When no SMTP is configured (e.g. a fresh self-hosted deploy), we can't
	// email a verification code — so accounts are auto-verified on signup and
	// the user is logged in immediately.
	autoVerify := !h.config.SMTPConfigured()

	var existing models.User
	if result := database.DB.Where("email = ?", req.Email).First(&existing); result.Error == nil {
		// An account already exists. When auto-verify is on there is no emailed
		// code to prove the requester owns this email, so we must NOT overwrite
		// its password or verify it from an unauthenticated request — doing so
		// would let anyone take over an existing unverified account. Treat
		// verified and unverified the same in that mode.
		if existing.EmailVerified || autoVerify {
			return echo.NewHTTPError(http.StatusConflict, "email already registered")
		}

		// SMTP configured: ownership is proven by the emailed code, so it is
		// safe to update the password and re-send verification for an
		// unverified account.
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to process password")
		}
		code := generateVerifyCode()
		expiry := time.Now().Add(10 * time.Minute)
		existing.Password = string(hashed)
		existing.Name = req.Name
		existing.Provider = "email"
		existing.VerifyCode = code
		existing.VerifyExpiry = &expiry
		if err := database.DB.Save(&existing).Error; err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to update account")
		}

		if err := h.emailService.SendVerificationCode(req.Email, code); err != nil {
			fmt.Printf("SMTP error: %v\n", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to send verification email")
		}
		return c.JSON(http.StatusOK, map[string]string{"message": "verification code sent to your email"})
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to process password")
	}

	user := models.User{
		Email:    req.Email,
		Name:     req.Name,
		Password: string(hashed),
		Provider: "email",
	}

	if autoVerify {
		user.EmailVerified = true
	} else {
		code := generateVerifyCode()
		expiry := time.Now().Add(10 * time.Minute)
		user.VerifyCode = code
		user.VerifyExpiry = &expiry
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
	}

	SetTrialForNewUser(&user)

	if autoVerify {
		return h.authSuccess(c, &user, http.StatusCreated)
	}

	if err := h.emailService.SendVerificationCode(req.Email, user.VerifyCode); err != nil {
		fmt.Printf("SMTP error: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to send verification email")
	}

	return c.JSON(http.StatusCreated, map[string]string{"message": "verification code sent to your email"})
}

// authSuccess issues a JWT for the user and returns the standard auth response.
func (h *AuthHandler) authSuccess(c echo.Context, user *models.User, status int) error {
	jwtToken, err := middleware.GenerateToken(user, h.config)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}
	return c.JSON(status, AuthResponse{Token: jwtToken, User: user})
}

// VerifyEmail verifies the user's email with the code
func (h *AuthHandler) VerifyEmail(c echo.Context) error {
	var req VerifyEmailRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	if user.EmailVerified {
		return echo.NewHTTPError(http.StatusBadRequest, "email already verified")
	}

	if user.VerifyCode != req.Code {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid verification code")
	}

	if user.VerifyExpiry != nil && time.Now().After(*user.VerifyExpiry) {
		return echo.NewHTTPError(http.StatusBadRequest, "verification code has expired")
	}

	if user.Suspended {
		return echo.NewHTTPError(http.StatusForbidden, "this account has been suspended")
	}

	user.EmailVerified = true
	user.VerifyCode = ""
	user.VerifyExpiry = nil
	database.DB.Save(&user)

	jwtToken, err := middleware.GenerateToken(&user, h.config)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	return c.JSON(http.StatusOK, AuthResponse{
		Token: jwtToken,
		User:  &user,
	})
}

// ResendCode resends the verification code
func (h *AuthHandler) ResendCode(c echo.Context) error {
	var req ResendCodeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	if user.EmailVerified {
		return echo.NewHTTPError(http.StatusBadRequest, "email already verified")
	}

	code := generateVerifyCode()
	expiry := time.Now().Add(10 * time.Minute)
	user.VerifyCode = code
	user.VerifyExpiry = &expiry
	database.DB.Save(&user)

	if err := h.emailService.SendVerificationCode(req.Email, code); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to send verification email")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "verification code resent"})
}

// EmailLogin authenticates a user with email and password
func (h *AuthHandler) EmailLogin(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid email or password")
	}

	if user.Password == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "please use social login for this account")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid email or password")
	}

	if user.Suspended {
		return echo.NewHTTPError(http.StatusForbidden, "this account has been suspended")
	}

	if !user.EmailVerified {
		return echo.NewHTTPError(http.StatusForbidden, "please verify your email first")
	}

	jwtToken, err := middleware.GenerateToken(&user, h.config)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	return c.JSON(http.StatusOK, AuthResponse{
		Token: jwtToken,
		User:  &user,
	})
}

func generateVerifyCode() string {
	code := ""
	for i := 0; i < 6; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		code += fmt.Sprintf("%d", n.Int64())
	}
	return code
}

// findOrCreateUser finds an existing user or creates a new one
func (h *AuthHandler) findOrCreateUser(email, name, avatarURL, provider, providerID string) (*models.User, error) {
	var user models.User

	// Try to find existing user by provider and provider_id
	result := database.DB.Where("provider = ? AND provider_id = ?", provider, providerID).First(&user)
	if result.Error == nil {
		// Update user info
		user.Name = name
		user.AvatarURL = avatarURL
		database.DB.Save(&user)
		return &user, nil
	}

	// Try to find by email
	result = database.DB.Where("email = ?", email).First(&user)
	if result.Error == nil {
		// Update user info
		user.Provider = provider
		user.ProviderID = providerID
		user.Name = name
		user.AvatarURL = avatarURL
		database.DB.Save(&user)
		return &user, nil
	}

	// Create new user
	user = models.User{
		Email:      email,
		Name:       name,
		AvatarURL:  avatarURL,
		Provider:   provider,
		ProviderID: providerID,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return nil, err
	}

	SetTrialForNewUser(&user)

	return &user, nil
}

