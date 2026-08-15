package gemini

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"smart-live-chats/internal/broadcast"
	"smart-live-chats/internal/database"
	"smart-live-chats/internal/models"
)

const maxToolHops = 3

// ToolContext carries session-scoped data for tool side effects (image send, leads).
type ToolContext struct {
	UserID    uint
	SessionID uint
	DeviceID  uint
	ChatJID   string
}

type toolDef struct {
	Name        string
	Description string
	Parameters  map[string]any
	Exec        func(ctx ToolContext, args map[string]any) (any, error)
}

func (s *Service) toolRegistry() []toolDef {
	return []toolDef{
		searchProductsTool(),
		getProductBySKUTool(),
		getActiveDiscountsTool(),
		sendProductImageTool(s),
		captureLeadTool(),
	}
}

func (s *Service) toolDeclarations() []map[string]any {
	var decls []map[string]any
	for _, t := range s.toolRegistry() {
		decls = append(decls, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"parameters":  t.Parameters,
		})
	}
	return decls
}

func searchProductsTool() toolDef {
	return toolDef{
		Name:        "search_products",
		Description: "Search the product catalog by name, description, tags, category, price, or stock. Returns up to 5 matches. Each result may include a product_url (product page) and checkout_url (buy link) you can share with the customer when they want to view or purchase.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query":      map[string]any{"type": "string", "description": "Search keywords"},
				"category":   map[string]any{"type": "string", "description": "Optional category filter"},
				"max_price":  map[string]any{"type": "number", "description": "Optional maximum price"},
				"in_stock":   map[string]any{"type": "boolean", "description": "If true, only products with stock > 0"},
			},
			"required": []string{"query"},
		},
		Exec: execSearchProducts,
	}
}

func execSearchProducts(ctx ToolContext, args map[string]any) (any, error) {
	query := strings.TrimSpace(fmt.Sprint(args["query"]))
	if query == "" {
		return map[string]any{"results": []any{}, "suggestion": "ask the customer to clarify"}, nil
	}

	category := strings.TrimSpace(fmt.Sprint(args["category"]))
	inStock, _ := args["in_stock"].(bool)
	maxPrice := 0.0
	if v, ok := args["max_price"].(float64); ok {
		maxPrice = v
	}

	var products []models.Product
	like := "%" + strings.ToLower(query) + "%"
	q := database.DB.Where("user_id = ? AND active = ? AND deleted_at IS NULL", ctx.UserID, true).
		Where("lower(name) LIKE ? OR lower(description) LIKE ? OR lower(tags) LIKE ? OR lower(sku) LIKE ?",
			like, like, like, like)
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if inStock {
		q = q.Where("stock > 0")
	}
	if maxPrice > 0 {
		q = q.Where("price <= ?", maxPrice)
	}
	q.Order("updated_at DESC").Limit(5).Find(&products)

	results := make([]map[string]any, 0, len(products))
	for _, p := range products {
		short := p.Description
		if len(short) > 120 {
			short = short[:120] + "…"
		}
		entry := map[string]any{
			"sku": p.SKU, "name": p.Name, "price": p.Price, "currency": p.Currency,
			"short_desc": short, "category": p.Category, "stock": p.Stock,
		}
		if p.ProductURL != "" {
			entry["product_url"] = p.ProductURL
		}
		if p.CheckoutURL != "" {
			entry["checkout_url"] = p.CheckoutURL
		}
		if d := bestDiscountForProduct(ctx.UserID, &p); d != nil {
			entry["active_discount"] = d
		}
		results = append(results, entry)
	}
	if len(results) == 0 {
		return map[string]any{"results": []any{}, "suggestion": "ask the customer to clarify"}, nil
	}
	return map[string]any{"results": results}, nil
}

func getProductBySKUTool() toolDef {
	return toolDef{
		Name:        "get_product_by_sku",
		Description: "Get full details for one product by SKU including primary image, active discount, product_url (product page link) and checkout_url (purchase link). Share product_url when the customer wants more info, and checkout_url when they're ready to buy.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"sku": map[string]any{"type": "string"},
			},
			"required": []string{"sku"},
		},
		Exec: execGetProductBySKU,
	}
}

func execGetProductBySKU(ctx ToolContext, args map[string]any) (any, error) {
	sku := strings.TrimSpace(fmt.Sprint(args["sku"]))
	if sku == "" {
		return nil, fmt.Errorf("sku is required")
	}
	var p models.Product
	if err := database.DB.Preload("Images").Where("user_id = ? AND sku = ? AND deleted_at IS NULL", ctx.UserID, sku).First(&p).Error; err != nil {
		return map[string]any{"found": false}, nil
	}
	primaryURL := ""
	for _, img := range p.Images {
		if img.IsPrimary {
			primaryURL = img.URL
			break
		}
	}
	if primaryURL == "" && len(p.Images) > 0 {
		primaryURL = p.Images[0].URL
	}
	out := map[string]any{
		"found":         true,
		"sku":           p.SKU,
		"name":          p.Name,
		"description":   p.Description,
		"price":         p.Price,
		"currency":      p.Currency,
		"stock":         p.Stock,
		"category":      p.Category,
		"tags":          p.Tags,
		"primary_image": primaryURL,
		"product_url":   p.ProductURL,
		"checkout_url":  p.CheckoutURL,
	}
	if d := bestDiscountForProduct(ctx.UserID, &p); d != nil {
		out["active_discount"] = d
		if adj, ok := d["adjusted_price"].(float64); ok {
			out["effective_price"] = adj
		}
	}
	// Track last viewed SKU on customer profile when possible
	if ctx.SessionID > 0 {
		var session models.ChatSession
		if database.DB.First(&session, ctx.SessionID).Error == nil && session.CustomerPhone != "" {
			var profile models.CustomerProfile
			if database.DB.Where("user_id = ? AND customer_phone = ?", ctx.UserID, session.CustomerPhone).First(&profile).Error == nil {
				profile.LastViewedSKU = sku
				database.DB.Save(&profile)
			}
		}
	}
	return out, nil
}

func getActiveDiscountsTool() toolDef {
	return toolDef{
		Name:        "get_active_discounts",
		Description: "List all currently active discounts for this store.",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Exec: execGetActiveDiscounts,
	}
}

func execGetActiveDiscounts(ctx ToolContext, _ map[string]any) (any, error) {
	now := time.Now()
	var discounts []models.Discount
	database.DB.Where("user_id = ? AND active = ? AND deleted_at IS NULL", ctx.UserID, true).Find(&discounts)

	out := make([]map[string]any, 0)
	for _, d := range discounts {
		if d.StartsAt != nil && now.Before(*d.StartsAt) {
			continue
		}
		if d.EndsAt != nil && now.After(*d.EndsAt) {
			continue
		}
		applies := "store-wide"
		if d.ProductID != nil {
			applies = fmt.Sprintf("product_id:%d", *d.ProductID)
		} else if d.Category != "" {
			applies = "category:" + d.Category
		}
		entry := map[string]any{
			"type": d.Type, "value": d.Value, "applies_to": applies,
		}
		if d.Code != "" {
			entry["code"] = d.Code
		}
		if d.EndsAt != nil {
			entry["ends_at"] = d.EndsAt.Format(time.RFC3339)
		}
		out = append(out, entry)
	}
	return map[string]any{"discounts": out}, nil
}

func sendProductImageTool(s *Service) toolDef {
	return toolDef{
		Name:        "send_product_image",
		Description: "Send the primary product image to the customer on WhatsApp. Call when the customer wants to see a product.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"sku": map[string]any{"type": "string"},
			},
			"required": []string{"sku"},
		},
		Exec: func(ctx ToolContext, args map[string]any) (any, error) {
			return s.execSendProductImage(ctx, args)
		},
	}
}

func (s *Service) execSendProductImage(ctx ToolContext, args map[string]any) (any, error) {
	sku := strings.TrimSpace(fmt.Sprint(args["sku"]))
	if sku == "" {
		return map[string]any{"sent": false, "error": "sku required"}, nil
	}

	var p models.Product
	if err := database.DB.Preload("Images").Where("user_id = ? AND sku = ?", ctx.UserID, sku).First(&p).Error; err != nil {
		return map[string]any{"sent": false, "error": "product not found"}, nil
	}

	imageURL := ""
	for _, img := range p.Images {
		if img.IsPrimary {
			imageURL = img.URL
			break
		}
	}
	if imageURL == "" && len(p.Images) > 0 {
		imageURL = p.Images[0].URL
	}
	if imageURL == "" {
		return map[string]any{"sent": false, "error": "no image for this product"}, nil
	}

	if ctx.DeviceID == 0 || ctx.ChatJID == "" || SendProductImageFunc == nil {
		return map[string]any{"sent": false, "error": "whatsapp not available"}, nil
	}

	caption := p.Name
	if err := SendProductImageFunc(ctx.DeviceID, ctx.ChatJID, imageURL, caption); err != nil {
		log.Printf("send_product_image failed: %v", err)
		return map[string]any{"sent": false, "error": err.Error()}, nil
	}

	if ctx.SessionID > 0 {
		now := time.Now()
		msg := models.ChatMessage{
			SessionID:   ctx.SessionID,
			SenderType:  "bot",
			Content:     caption,
			MessageType: "image",
			MediaURL:    imageURL,
		}
		database.DB.Create(&msg)
		database.DB.Model(&models.ChatSession{}).Where("id = ?", ctx.SessionID).
			Updates(map[string]interface{}{"last_message": "[Image]", "last_message_at": now, "last_sender_type": "bot"})
		broadcast.SessionMessage(ctx.SessionID, &msg)
		broadcast.ToUser(ctx.UserID, &broadcast.Message{
			Type:      "message",
			SessionID: ctx.SessionID,
			Payload:   &msg,
		})
	}

	return map[string]any{"sent": true, "url": imageURL}, nil
}

func captureLeadTool() toolDef {
	return toolDef{
		Name:        "capture_lead",
		Description: "Record customer purchase intent. Call when the customer wants to buy or asks for checkout.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"sku":      map[string]any{"type": "string"},
				"quantity": map[string]any{"type": "integer"},
				"notes":    map[string]any{"type": "string"},
			},
			"required": []string{"sku"},
		},
		Exec: execCaptureLead,
	}
}

func execCaptureLead(ctx ToolContext, args map[string]any) (any, error) {
	sku := strings.TrimSpace(fmt.Sprint(args["sku"]))
	if sku == "" {
		return nil, fmt.Errorf("sku is required")
	}
	qty := 1
	if v, ok := args["quantity"].(float64); ok && int(v) > 0 {
		qty = int(v)
	}
	notes := strings.TrimSpace(fmt.Sprint(args["notes"]))

	// Ephemeral test conversations (Test Bot sandbox) have no persisted session.
	// Acknowledge the intent to the model so the reply flow stays natural, but do
	// not write a real lead or mutate customer profiles.
	if ctx.SessionID == 0 {
		return map[string]any{"lead_id": 0, "status": "recorded", "note": "test sandbox: not persisted"}, nil
	}

	lead := models.Lead{
		UserID:    ctx.UserID,
		SessionID: ctx.SessionID,
		SKU:       sku,
		Quantity:  qty,
		Notes:     notes,
	}
	var existing models.Lead
	if err := database.DB.Where("user_id = ? AND session_id = ? AND sku = ?", ctx.UserID, ctx.SessionID, sku).First(&existing).Error; err == nil {
		existing.Quantity = qty
		existing.Notes = notes
		database.DB.Save(&existing)
		lead = existing
	} else {
		database.DB.Create(&lead)
	}

	if ctx.SessionID > 0 {
		var session models.ChatSession
		if database.DB.First(&session, ctx.SessionID).Error == nil && session.CustomerPhone != "" {
			var profile models.CustomerProfile
			result := database.DB.Where("user_id = ? AND customer_phone = ?", ctx.UserID, session.CustomerPhone).First(&profile)
			if result.Error != nil {
				profile = models.CustomerProfile{
					UserID:        ctx.UserID,
					CustomerPhone: session.CustomerPhone,
					LeadStage:     "hot",
				}
				database.DB.Create(&profile)
			} else {
				profile.LeadStage = "hot"
				profile.LastViewedSKU = sku
				database.DB.Save(&profile)
			}
		}
	}

	broadcast.ToUser(ctx.UserID, &broadcast.Message{
		Type: "lead_captured",
		Payload: map[string]interface{}{
			"lead_id":    lead.ID,
			"session_id": ctx.SessionID,
			"sku":        sku,
			"quantity":   qty,
			"notes":      notes,
		},
	})

	return map[string]any{"lead_id": lead.ID, "status": "recorded"}, nil
}

func bestDiscountForProduct(userID uint, p *models.Product) map[string]any {
	now := time.Now()
	var discounts []models.Discount
	database.DB.Where("user_id = ? AND active = ? AND deleted_at IS NULL", userID, true).Find(&discounts)

	var best *models.Discount
	for i := range discounts {
		d := &discounts[i]
		if d.StartsAt != nil && now.Before(*d.StartsAt) {
			continue
		}
		if d.EndsAt != nil && now.After(*d.EndsAt) {
			continue
		}
		if d.ProductID != nil && *d.ProductID == p.ID {
			best = d
			break
		}
		if d.Category != "" && strings.EqualFold(d.Category, p.Category) && best == nil {
			best = d
		}
		if d.ProductID == nil && d.Category == "" && best == nil {
			best = d
		}
	}
	if best == nil {
		return nil
	}
	adj := p.Price
	switch best.Type {
	case "percent":
		adj = p.Price * (1 - best.Value/100)
	case "fixed":
		adj = p.Price - best.Value
		if adj < 0 {
			adj = 0
		}
	}
	out := map[string]any{
		"type": best.Type, "value": best.Value, "adjusted_price": adj,
	}
	if best.Code != "" {
		out["code"] = best.Code
	}
	return out
}

func argsToMap(raw json.RawMessage) map[string]any {
	var m map[string]any
	if len(raw) == 0 {
		return map[string]any{}
	}
	_ = json.Unmarshal(raw, &m)
	return m
}
