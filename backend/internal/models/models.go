package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	Email              string         `gorm:"uniqueIndex;not null;size:255" json:"email"`
	Name               string         `gorm:"size:255" json:"name"`
	AvatarURL          string         `gorm:"type:text" json:"avatar_url"`
	Password           string         `gorm:"size:255" json:"-"`
	Provider           string         `gorm:"size:50" json:"provider"`
	ProviderID         string         `gorm:"size:255" json:"provider_id"`
	EmailVerified      bool           `gorm:"default:false" json:"email_verified"`
	VerifyCode         string         `gorm:"size:6" json:"-"`
	VerifyExpiry       *time.Time     `json:"-"`
	StripeCustomerID   string         `gorm:"size:255" json:"-"`
	SubscriptionPlan   string         `gorm:"size:50;default:'trial'" json:"subscription_plan"`
	SubscriptionStatus string         `gorm:"size:50;default:'trialing'" json:"subscription_status"`
	TrialEndsAt        *time.Time     `json:"trial_ends_at"`
	MaxDevices         int            `gorm:"default:1" json:"max_devices"`
	Suspended          bool           `gorm:"default:false;index" json:"suspended"`
	SuspendedAt        *time.Time     `json:"suspended_at"`
	SuspendedReason    string         `gorm:"type:text" json:"suspended_reason"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

// Subscription tracks Stripe subscription details
type Subscription struct {
	ID                   uint           `gorm:"primaryKey" json:"id"`
	UserID               uint           `gorm:"not null;index" json:"user_id"`
	User                 *User          `gorm:"foreignKey:UserID" json:"-"`
	StripeSubscriptionID string         `gorm:"size:255;uniqueIndex" json:"stripe_subscription_id"`
	StripePriceID        string         `gorm:"size:255" json:"stripe_price_id"`
	Plan                 string         `gorm:"size:50" json:"plan"`
	Status               string         `gorm:"size:50" json:"status"`
	CurrentPeriodStart   *time.Time     `json:"current_period_start"`
	CurrentPeriodEnd     *time.Time     `json:"current_period_end"`
	CancelAtPeriodEnd    bool           `gorm:"default:false" json:"cancel_at_period_end"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Subscription) TableName() string {
	return "subscriptions"
}

// ChatSession represents a chat conversation
type ChatSession struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	CustomerName   string         `gorm:"size:255" json:"customer_name"`
	CustomerAvatar string         `gorm:"type:text" json:"customer_avatar"`
	CustomerPhone  string         `gorm:"size:50" json:"customer_phone"`
	Platform       string         `gorm:"size:50" json:"platform"` // 'whatsapp', 'facebook', 'telegram', etc.
	UnreadCount    int            `gorm:"default:0" json:"unread_count"`
	LastMessage    string         `gorm:"type:text" json:"last_message"`
	LastMessageAt  *time.Time     `json:"last_message_at"`
	LastSenderType string         `gorm:"size:20" json:"last_sender_type"`
	Status         string         `gorm:"size:20;default:'active'" json:"status"` // 'active', 'closed'
	DeviceID       *uint          `json:"device_id"`
	AssignedToID   *uint          `json:"assigned_to_id"`
	AssignedTo     *User          `gorm:"foreignKey:AssignedToID" json:"assigned_to,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	Messages       []ChatMessage  `gorm:"foreignKey:SessionID" json:"messages,omitempty"`
}

// ChatMessage represents a single message in a chat session
type ChatMessage struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	SessionID   uint           `gorm:"not null;index" json:"session_id"`
	Session     *ChatSession   `gorm:"foreignKey:SessionID" json:"-"`
	SenderType  string         `gorm:"size:20;not null" json:"sender_type"` // 'customer' or 'agent'
	SenderID    *uint          `json:"sender_id"`                           // User ID if sender is agent
	Content     string         `gorm:"type:text" json:"content"`
	MessageType string         `gorm:"size:20;default:'text'" json:"message_type"` // 'text', 'image', 'file'
	MediaURL    string         `gorm:"type:text" json:"media_url,omitempty"`
	IsRead      bool           `gorm:"default:false" json:"is_read"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// WhatsAppDevice represents a connected WhatsApp device
type WhatsAppDevice struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UserID      uint           `gorm:"not null;index" json:"user_id"`
	User        *User          `gorm:"foreignKey:UserID" json:"-"`
	JID         string         `gorm:"size:255" json:"jid"`
	PushName    string         `gorm:"size:255" json:"push_name"`
	Phone       string         `gorm:"size:50" json:"phone"`
	Status      string         `gorm:"size:20;default:'disconnected'" json:"status"`
	ConnectedAt *time.Time     `json:"connected_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// KnowledgeBase stores user's Gemini File Search Store config
type KnowledgeBase struct {
	ID                  uint           `gorm:"primaryKey" json:"id"`
	UserID              uint           `gorm:"uniqueIndex;not null" json:"user_id"`
	User                *User          `gorm:"foreignKey:UserID" json:"-"`
	GeminiStoreID       string         `gorm:"size:255" json:"gemini_store_id"`
	GeminiStoreName     string         `gorm:"size:255" json:"gemini_store_name"`
	CacheName           string         `gorm:"size:255" json:"-"`
	CacheExpiresAt      *time.Time     `json:"-"`
	AutoReplyEnabled    bool           `gorm:"default:false" json:"auto_reply_enabled"`
	SystemPrompt        string         `gorm:"type:text" json:"system_prompt"`
	LastSyncedAt        *time.Time     `json:"last_synced_at"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}

// QAItem represents a single Q&A pair in the knowledge base
type QAItem struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	User      *User          `gorm:"foreignKey:UserID" json:"-"`
	Question  string         `gorm:"type:text;not null" json:"question"`
	Answer    string         `gorm:"type:text;not null" json:"answer"`
	Category  string         `gorm:"size:100" json:"category"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// PlatformSetting is a generic key-value store for global platform configuration
// that can be edited at runtime by the platform admin (no server restart needed).
// Typical keys: "gemini_system_prompt".
type PlatformSetting struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"size:100;uniqueIndex;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	UpdatedBy string    `gorm:"size:100" json:"updated_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (PlatformSetting) TableName() string { return "platform_settings" }

// Product is a normalized catalog item for AI sales tooling and the single source of truth for the catalog.
type Product struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UserID      uint           `gorm:"not null;uniqueIndex:idx_user_sku" json:"user_id"`
	SKU         string         `gorm:"size:100;not null;uniqueIndex:idx_user_sku" json:"sku"`
	Name        string         `gorm:"size:255;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Price       float64        `gorm:"type:numeric(12,2);not null" json:"price"`
	Currency    string         `gorm:"size:8;default:'USD'" json:"currency"`
	Stock       int            `gorm:"default:0" json:"stock"`
	Category    string         `gorm:"size:100;index" json:"category"`
	Tags        string         `gorm:"type:text" json:"tags"`
	ProductURL  string         `gorm:"type:text" json:"product_url"`
	CheckoutURL string         `gorm:"type:text" json:"checkout_url"`
	Active      bool           `gorm:"default:true;index" json:"active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Images      []ProductImage `gorm:"foreignKey:ProductID" json:"images,omitempty"`
}

// ProductImage is a URL attached to a product (one may be marked primary).
type ProductImage struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ProductID uint   `gorm:"not null;index" json:"product_id"`
	URL       string `gorm:"type:text;not null" json:"url"`
	IsPrimary bool   `gorm:"default:false" json:"is_primary"`
	SortOrder int    `gorm:"default:0" json:"sort_order"`
}

// Discount applies to a product, category, or store-wide.
type Discount struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	ProductID *uint          `gorm:"index" json:"product_id"`
	Category  string         `gorm:"size:100;index" json:"category"`
	Type      string         `gorm:"size:20" json:"type"` // percent | fixed | bogo
	Value     float64        `json:"value"`
	Code      string         `gorm:"size:50;index" json:"code,omitempty"`
	StartsAt  *time.Time     `json:"starts_at"`
	EndsAt    *time.Time     `json:"ends_at"`
	Active    bool           `gorm:"default:true;index" json:"active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// CustomerProfile stores cross-session memory for a WhatsApp customer.
type CustomerProfile struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"not null;uniqueIndex:idx_user_phone,priority:1" json:"user_id"`
	CustomerPhone string    `gorm:"size:50;not null;uniqueIndex:idx_user_phone,priority:2" json:"customer_phone"`
	PreferredLang string    `gorm:"size:10" json:"preferred_lang"`
	Interests     string    `gorm:"type:text" json:"interests"` // JSON array
	LastViewedSKU string    `gorm:"size:100" json:"last_viewed_sku"`
	LeadStage     string    `gorm:"size:20;default:'cold'" json:"lead_stage"`
	Notes         string    `gorm:"type:text" json:"notes"`
	LastSummaryAt *time.Time `json:"last_summary_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Lead is captured when the AI bot records purchase intent.
type Lead struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	SessionID uint      `gorm:"not null;index:idx_lead_session_sku,unique" json:"session_id"`
	SKU       string    `gorm:"size:100;not null;index:idx_lead_session_sku,unique" json:"sku"`
	Quantity  int       `gorm:"default:1" json:"quantity"`
	Notes     string    `gorm:"type:text" json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Product) TableName() string        { return "products" }
func (ProductImage) TableName() string   { return "product_images" }
func (Discount) TableName() string      { return "discounts" }
func (CustomerProfile) TableName() string { return "customer_profiles" }
func (Lead) TableName() string         { return "leads" }

// Tag represents a label that can be applied to chat sessions
type Tag struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	Name      string         `gorm:"size:100;not null" json:"name"`
	Color     string         `gorm:"size:20;default:'blue'" json:"color"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// SessionTag is the join table between ChatSession and Tag
type SessionTag struct {
	ID        uint `gorm:"primaryKey" json:"id"`
	SessionID uint `gorm:"not null;index" json:"session_id"`
	TagID     uint `gorm:"not null;index" json:"tag_id"`
	Tag       *Tag `gorm:"foreignKey:TagID" json:"tag,omitempty"`
}

func (Tag) TableName() string        { return "tags" }
func (SessionTag) TableName() string { return "session_tags" }

func (KnowledgeBase) TableName() string {
	return "knowledge_bases"
}

func (QAItem) TableName() string {
	return "qa_items"
}

// TableName specifies the table name for WhatsAppDevice
func (WhatsAppDevice) TableName() string {
	return "whatsapp_devices"
}

// TableName specifies the table name for User
func (User) TableName() string {
	return "users"
}

// TableName specifies the table name for ChatSession
func (ChatSession) TableName() string {
	return "chat_sessions"
}

// TableName specifies the table name for ChatMessage
func (ChatMessage) TableName() string {
	return "chat_messages"
}

