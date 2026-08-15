package gemini

import (
	"fmt"
	"strings"

	"smart-live-chats/internal/database"
	"smart-live-chats/internal/models"
)

const maxCatalogSummaryProducts = 25

// RenderProductCatalogSummary builds a compact text catalog from the products
// table. It serves as a count-only hint in the system prompt; the AI reaches
// product specifics via tools (search_products, get_product_by_sku).
func RenderProductCatalogSummary(userID uint, compact bool) string {
	var count int64
	database.DB.Model(&models.Product{}).
		Where("user_id = ? AND active = ? AND deleted_at IS NULL", userID, true).
		Count(&count)
	if count == 0 {
		return ""
	}

	if compact {
		var categories []string
		database.DB.Model(&models.Product{}).
			Select("DISTINCT category").
			Where("user_id = ? AND active = ? AND category <> '' AND deleted_at IS NULL", userID, true).
			Scan(&categories)
		catCount := len(categories)
		if catCount == 0 {
			catCount = 1
		}
		return fmt.Sprintf("Catalog: %d active products across %d categories. Use search_products to look up specifics.", count, catCount)
	}

	var products []models.Product
	database.DB.Where("user_id = ? AND active = ? AND deleted_at IS NULL", userID, true).
		Order("category ASC, name ASC").
		Limit(maxCatalogSummaryProducts).
		Find(&products)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Product catalog (%d items", count))
	if count > int64(len(products)) {
		b.WriteString(fmt.Sprintf(", showing first %d", len(products)))
	}
	b.WriteString(")\n\n")
	for _, p := range products {
		b.WriteString(fmt.Sprintf("- [%s] %s — %.2f %s", p.SKU, p.Name, p.Price, p.Currency))
		if p.Category != "" {
			b.WriteString(fmt.Sprintf(" (%s)", p.Category))
		}
		if p.Stock > 0 {
			b.WriteString(fmt.Sprintf(", stock: %d", p.Stock))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// renderTopFAQ extracts up to 10 Q&A pairs for the system prompt.
func renderTopFAQ(userID uint, limit int) string {
	if limit <= 0 {
		limit = 10
	}
	var items []models.QAItem
	database.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&items)
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Brand FAQ (top questions)\n\n")
	for _, item := range items {
		b.WriteString(fmt.Sprintf("Q: %s\nA: %s\n\n", item.Question, item.Answer))
	}
	return b.String()
}
