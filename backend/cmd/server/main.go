package main

import (
	"log"
	"net/http"

	"smart-live-chats/internal/config"
	"smart-live-chats/internal/database"
	"smart-live-chats/internal/gemini"
	"smart-live-chats/internal/handlers"
	"smart-live-chats/internal/middleware"
	"smart-live-chats/internal/whatsapp"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

func main() {
	// Load configuration
	cfg := config.Load()
	cfg.LogStartupSummary()

	// Connect to database
	if err := database.Connect(cfg); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(echoMiddleware.Logger())
	e.Use(echoMiddleware.Recover())
	allowedOrigins := []string{cfg.FrontendURL}
	allowedOrigins = append(allowedOrigins, cfg.CORSOrigins...)
	e.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowCredentials: true,
	}))

	// Initialize WhatsApp manager
	waManager, err := whatsapp.NewManager(cfg.WhatsAppDBPath)
	if err != nil {
		log.Fatalf("Failed to initialize WhatsApp manager: %v", err)
	}
	waManager.ReconnectAll()

	// Initialize Gemini service
	gemini.NewService(cfg)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(cfg)
	chatHandler := handlers.NewChatHandler()
	wsHandler := handlers.NewWebSocketHandler(cfg)
	waHandler := handlers.NewWhatsAppHandler(cfg)
	kbHandler := handlers.NewKnowledgeHandler()
	productsHandler := handlers.NewProductsHandler()
	billingHandler := handlers.NewBillingHandler(cfg)
	tagHandler := handlers.NewTagHandler()
	adminHandler := handlers.NewAdminHandler(cfg)
	metaHandler := handlers.NewMetaHandler(cfg)

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// API routes
	api := e.Group("/api")

	// Public server metadata (feature flags for the frontend)
	api.GET("/config", metaHandler.PublicConfig)

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/verify-email", authHandler.VerifyEmail)
	auth.POST("/resend-code", authHandler.ResendCode)
	auth.POST("/login", authHandler.EmailLogin)
	auth.GET("/google", authHandler.GoogleLogin)
	auth.POST("/google/callback", authHandler.GoogleCallback)
	auth.GET("/facebook", authHandler.FacebookLogin)
	auth.POST("/facebook/callback", authHandler.FacebookCallback)

	// Protected auth routes
	authProtected := auth.Group("")
	authProtected.Use(middleware.JWTMiddleware(cfg))
	authProtected.GET("/me", authHandler.GetMe)

	// Chat routes (protected + subscription required)
	chats := api.Group("/chats")
	chats.Use(middleware.JWTMiddleware(cfg))
	chats.Use(middleware.SubscriptionMiddleware(cfg))
	chats.GET("", chatHandler.ListChats)
	chats.GET("/stats", chatHandler.GetChatStats)
	chats.GET("/reports", chatHandler.GetReportData)
	chats.GET("/blacklisted", chatHandler.GetBlacklisted)
	chats.GET("/:id", chatHandler.GetChat)
	chats.GET("/:id/messages", chatHandler.GetChatMessages)
	chats.POST("/:id/messages", chatHandler.SendMessage)
	chats.POST("/:id/read", chatHandler.MarkAsRead)
	chats.POST("/:id/upload", chatHandler.UploadImage)
	chats.POST("/:id/accept", chatHandler.AcceptConversation)
	chats.POST("/:id/blacklist", chatHandler.BlacklistCustomer)
	chats.POST("/:id/unblacklist", chatHandler.UnblacklistCustomer)
	chats.GET("/:id/tags", tagHandler.GetSessionTags)
	chats.POST("/:id/tags", tagHandler.AddTagToSession)
	chats.DELETE("/:id/tags/:tagId", tagHandler.RemoveTagFromSession)

	// Tags routes (protected + subscription required)
	tags := api.Group("/tags")
	tags.Use(middleware.JWTMiddleware(cfg))
	tags.Use(middleware.SubscriptionMiddleware(cfg))
	tags.GET("", tagHandler.ListTags)
	tags.POST("", tagHandler.CreateTag)
	tags.DELETE("/:id", tagHandler.DeleteTag)

	// WhatsApp routes (protected + subscription required)
	wa := api.Group("/whatsapp")
	wa.Use(middleware.JWTMiddleware(cfg))
	wa.Use(middleware.SubscriptionMiddleware(cfg))
	wa.POST("/connect", waHandler.Connect)
	wa.POST("/disconnect", waHandler.Disconnect)
	wa.POST("/disconnect/:deviceId", waHandler.Disconnect)
	wa.GET("/status", waHandler.GetStatus)
	wa.GET("/devices", waHandler.GetDevices)

	// Knowledge Base / Q&A routes (protected + subscription required)
	kb := api.Group("/knowledge")
	kb.Use(middleware.JWTMiddleware(cfg))
	kb.Use(middleware.SubscriptionMiddleware(cfg))
	kb.GET("", kbHandler.GetKnowledgeBase)
	kb.PUT("", kbHandler.UpdateKnowledgeBase)
	kb.GET("/qa", kbHandler.ListQAItems)
	kb.POST("/qa", kbHandler.CreateQAItem)
	kb.PUT("/qa/:id", kbHandler.UpdateQAItem)
	kb.DELETE("/qa", kbHandler.DeleteAllQAItems)
	kb.DELETE("/qa/:id", kbHandler.DeleteQAItem)
	kb.POST("/sync", kbHandler.SyncToGemini)
	kb.POST("/upload", kbHandler.UploadFile)
	kb.GET("/template", kbHandler.DownloadTemplate)
	kb.POST("/test", kbHandler.TestQuery)
	kb.POST("/test/chat", kbHandler.TestChat)

	// Products catalog (protected + subscription required)
	products := api.Group("/products")
	products.Use(middleware.JWTMiddleware(cfg))
	products.Use(middleware.SubscriptionMiddleware(cfg))
	products.GET("", productsHandler.ListProducts)
	products.POST("", productsHandler.CreateProduct)
	products.GET("/template", productsHandler.DownloadProductTemplate)
	products.POST("/import", productsHandler.ImportProducts)
	products.GET("/discounts", productsHandler.ListDiscounts)
	products.POST("/discounts", productsHandler.CreateDiscount)
	products.PUT("/discounts/:id", productsHandler.UpdateDiscount)
	products.DELETE("/discounts/:id", productsHandler.DeleteDiscount)
	products.PUT("/:id", productsHandler.UpdateProduct)
	products.DELETE("/:id", productsHandler.DeleteProduct)
	products.POST("/:id/images", productsHandler.AddProductImage)
	products.DELETE("/:id/images/:imageId", productsHandler.DeleteProductImage)

	// Leads (protected + subscription required)
	leads := api.Group("/leads")
	leads.Use(middleware.JWTMiddleware(cfg))
	leads.Use(middleware.SubscriptionMiddleware(cfg))
	leads.GET("", productsHandler.ListLeads)

	// Billing routes
	billing := api.Group("/billing")
	billing.Use(middleware.JWTMiddleware(cfg))
	billing.POST("/checkout", billingHandler.CreateCheckout)
	billing.POST("/portal", billingHandler.CreatePortal)
	billing.PUT("/quantity", billingHandler.UpdateQuantity)
	billing.GET("/subscription", billingHandler.GetSubscription)
	billing.POST("/sync", billingHandler.SyncSubscription)

	// Admin routes
	admin := api.Group("/admin")
	admin.POST("/login", adminHandler.Login)
	adminProtected := admin.Group("")
	adminProtected.Use(middleware.AdminMiddleware(cfg))
	adminProtected.GET("/dashboard", adminHandler.GetDashboard)
	adminProtected.GET("/users", adminHandler.ListUsers)
	adminProtected.GET("/users/:id", adminHandler.GetUser)
	adminProtected.PATCH("/users/:id", adminHandler.UpdateUser)
	adminProtected.DELETE("/users/:id", adminHandler.DeleteUser)
	adminProtected.POST("/users/:id/reset-password", adminHandler.ResetUserPassword)
	adminProtected.POST("/users/:id/suspend", adminHandler.SuspendUser)
	adminProtected.POST("/users/:id/unsuspend", adminHandler.UnsuspendUser)
	adminProtected.GET("/users/:id/knowledge", adminHandler.GetUserKnowledge)
	adminProtected.PUT("/users/:id/knowledge", adminHandler.UpdateUserKnowledge)
	adminProtected.GET("/devices", adminHandler.ListDevices)
	adminProtected.GET("/devices/:deviceId/chats", adminHandler.ListDeviceChats)
	adminProtected.GET("/chats/:sessionId/messages", adminHandler.GetAdminChatMessages)
	adminProtected.GET("/subscriptions", adminHandler.ListSubscriptions)
	adminProtected.GET("/chats/analytics", adminHandler.GetChatAnalytics)
	adminProtected.GET("/system", adminHandler.GetSystemInfo)
	adminProtected.GET("/bot-prompt", adminHandler.GetBotPrompt)
	adminProtected.PUT("/bot-prompt", adminHandler.UpdateBotPrompt)
	adminProtected.GET("/ai-reply-delay", adminHandler.GetAIReplyDelay)
	adminProtected.PUT("/ai-reply-delay", adminHandler.UpdateAIReplyDelay)
	adminProtected.POST("/ai-reply-delay/reset", adminHandler.ResetAIReplyDelay)

	// Stripe webhook (no auth - verified by signature)
	api.POST("/stripe/webhook", billingHandler.HandleWebhook)

	// WebSocket (with optional auth via query param)
	e.GET("/ws", wsHandler.HandleWebSocket)

	// Start server
	listenAddr := cfg.ListenHost + ":" + cfg.ServerPort
	log.Printf("Server starting on %s", listenAddr)
	if err := e.Start(listenAddr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}

