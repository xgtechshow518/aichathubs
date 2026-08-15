package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	ServerPort string
	ListenHost string

	// Database
	DatabaseURL string

	// JWT
	JWTSecret     string
	JWTExpiryDays int

	// OAuth - Google
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	// OAuth - Facebook
	FacebookClientID     string
	FacebookClientSecret string
	FacebookRedirectURL  string

	// SMTP
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string

	// Stripe
	StripeSecretKey       string
	StripeWebhookSecret   string
	StripePriceStarter    string
	StripePricePro        string
	StripePriceEnterprise string

	// Billing behaviour: when false the app runs fully free (no Stripe,
	// no WhatsApp device limit, no subscription gate).
	BillingEnabled bool

	// Gemini AI
	GeminiAPIKey          string
	GeminiModel           string
	GeminiSystemPrompt    string
	MatchScoreThreshold   float64

	// Frontend URL (for CORS)
	FrontendURL string

	// CORS - additional allowed origins (comma-separated)
	CORSOrigins []string

	// Admin credentials
	AdminUsername string
	AdminPassword string

	// WhatsApp session database path
	WhatsAppDBPath string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	return &Config{
		ServerPort:            getEnv("SERVER_PORT", "8080"),
		ListenHost:            getEnv("LISTEN_HOST", "0.0.0.0"),
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/smart_live_chats?sslmode=disable"),
		JWTSecret:             getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-in-production"),
		JWTExpiryDays:         getEnvInt("JWT_EXPIRY_DAYS", 7),
		GoogleClientID:        getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:    getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:     getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/auth/google/callback"),
		FacebookClientID:      getEnv("FACEBOOK_CLIENT_ID", ""),
		FacebookClientSecret:  getEnv("FACEBOOK_CLIENT_SECRET", ""),
		FacebookRedirectURL:   getEnv("FACEBOOK_REDIRECT_URL", "http://localhost:8080/api/auth/facebook/callback"),
		SMTPHost:              getEnv("SMTP_HOST", ""),
		SMTPPort:              getEnv("SMTP_PORT", "587"),
		SMTPUser:              getEnv("SMTP_USER", ""),
		SMTPPassword:          getEnv("SMTP_PASS", ""),
		SMTPFrom:              getEnv("SMTP_FROM", ""),
		StripeSecretKey:       getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret:   getEnv("STRIPE_WEBHOOK_SECRET", ""),
		StripePriceStarter:    getEnv("STRIPE_PRICE_STARTER", ""),
		StripePricePro:        getEnv("STRIPE_PRICE_PRO", ""),
		StripePriceEnterprise: getEnv("STRIPE_PRICE_ENTERPRISE", ""),
		BillingEnabled:        getEnvBool("BILLING_ENABLED", false),
		GeminiAPIKey:          getEnv("GEMINI_API_KEY", ""),
		GeminiModel:           getEnv("GEMINI_MODEL", "gemini-2.5-flash"),
		GeminiSystemPrompt:    getEnv("GEMINI_SYSTEM_PROMPT", "You are a professional and friendly customer service assistant. Answer customer questions accurately based on the provided knowledge base. Be concise, helpful, and polite. Reply in the same language the customer uses. If you cannot find the answer in the knowledge base, politely let the customer know and suggest they contact a human agent for further assistance. Never make up information."),
		MatchScoreThreshold:   getEnvFloat("MATCH_SCORE_THRESHOLD", 0.3),
		FrontendURL:           getEnv("FRONTEND_URL", "http://localhost:5173"),
		CORSOrigins:           parseOrigins(getEnv("CORS_ORIGINS", "")),
		AdminUsername:         getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:         getEnv("ADMIN_PASSWORD", "admin123456"),
		WhatsAppDBPath:        getEnv("WHATSAPP_DB_PATH", "file:whatsapp_sessions.db?_foreign_keys=on"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

const (
	defaultJWTSecret     = "your-super-secret-jwt-key-change-in-production"
	defaultAdminPassword = "admin123456"
)

// SMTPConfigured reports whether outbound email (verification codes) can be sent.
func (c *Config) SMTPConfigured() bool {
	return c.SMTPHost != "" && c.SMTPUser != "" && c.SMTPPassword != "" && c.SMTPFrom != ""
}

// GoogleOAuthConfigured reports whether "Sign in with Google" is available.
func (c *Config) GoogleOAuthConfigured() bool {
	return c.GoogleClientID != "" && c.GoogleClientSecret != ""
}

// FacebookOAuthConfigured reports whether "Sign in with Facebook" is available.
func (c *Config) FacebookOAuthConfigured() bool {
	return c.FacebookClientID != "" && c.FacebookClientSecret != ""
}

// GeminiConfigured reports whether the AI assistant has an API key.
func (c *Config) GeminiConfigured() bool {
	return c.GeminiAPIKey != ""
}

// StripeConfigured reports whether Stripe billing calls can be made.
func (c *Config) StripeConfigured() bool {
	return c.StripeSecretKey != ""
}

// LogStartupSummary prints which optional features are enabled and warns about
// insecure defaults. Helps self-hosters see at a glance what they still need.
func (c *Config) LogStartupSummary() {
	onOff := func(ok bool) string {
		if ok {
			return "enabled"
		}
		return "disabled"
	}
	log.Println("──────────── AIChatsHub configuration ────────────")
	log.Printf("  AI assistant (Gemini):  %s", onOff(c.GeminiConfigured()))
	log.Printf("  Email verification:     %s", onOff(c.SMTPConfigured()))
	log.Printf("  Google login:           %s", onOff(c.GoogleOAuthConfigured()))
	log.Printf("  Facebook login:         %s", onOff(c.FacebookOAuthConfigured()))
	log.Printf("  Billing (Stripe):       %s", onOff(c.BillingEnabled && c.StripeConfigured()))
	if !c.SMTPConfigured() {
		log.Println("  → No SMTP: new users are auto-verified on signup (no email sent).")
		if c.SMTPHost != "" || c.SMTPUser != "" || c.SMTPPassword != "" || c.SMTPFrom != "" {
			log.Println("  WARNING: SMTP is only PARTIALLY set (need all of SMTP_HOST/USER/PASS/FROM) — email verification is OFF and signups are auto-verified. Check for a missing/typo'd SMTP_* var.")
		}
	}
	if !c.BillingEnabled {
		log.Println("  → Billing off: unlimited WhatsApp devices, no subscription gate.")
	}
	if c.JWTSecret == defaultJWTSecret {
		log.Println("  WARNING: JWT_SECRET is the built-in default — set a strong random JWT_SECRET before exposing this server.")
	}
	if c.AdminPassword == defaultAdminPassword {
		log.Println("  WARNING: ADMIN_PASSWORD is the built-in default — change ADMIN_PASSWORD before exposing this server.")
	}
	log.Println("──────────────────────────────────────────────────")
}

func parseOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}

