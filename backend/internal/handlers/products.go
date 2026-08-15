package handlers

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"smart-live-chats/internal/database"
	"smart-live-chats/internal/gemini"
	"smart-live-chats/internal/middleware"
	"smart-live-chats/internal/models"

	"github.com/labstack/echo/v4"
	"github.com/xuri/excelize/v2"
)

type ProductsHandler struct{}

func NewProductsHandler() *ProductsHandler {
	return &ProductsHandler{}
}

func invalidateProductCache(userID uint) {
	if gemini.GlobalService != nil {
		gemini.GlobalService.InvalidateUserContextCache(userID)
	}
}

// ListProducts returns paginated products for the current user.
func (h *ProductsHandler) ListProducts(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	search := strings.TrimSpace(c.QueryParam("search"))
	category := strings.TrimSpace(c.QueryParam("category"))
	activeOnly := c.QueryParam("active") == "true"

	query := database.DB.Model(&models.Product{}).Where("user_id = ?", user.ID)
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name ILIKE ? OR sku ILIKE ? OR description ILIKE ?", like, like, like)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if activeOnly {
		query = query.Where("active = ?", true)
	}

	var products []models.Product
	query.Preload("Images").Order("updated_at DESC").Find(&products)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"products": products,
		"total":    len(products),
	})
}

type productRequest struct {
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Currency    string  `json:"currency"`
	Stock       int     `json:"stock"`
	Category    string  `json:"category"`
	Tags        string  `json:"tags"`
	ProductURL  *string `json:"product_url"`
	CheckoutURL *string `json:"checkout_url"`
	Active      *bool   `json:"active"`
}

// CreateProduct adds a catalog item.
func (h *ProductsHandler) CreateProduct(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var req productRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	req.SKU = strings.TrimSpace(req.SKU)
	req.Name = strings.TrimSpace(req.Name)
	if req.SKU == "" || req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "sku and name are required")
	}
	if req.Price < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "price must be non-negative")
	}
	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = "USD"
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}

	p := models.Product{
		UserID:      user.ID,
		SKU:         req.SKU,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Currency:    currency,
		Stock:       req.Stock,
		Category:    req.Category,
		Tags:        req.Tags,
		Active:      active,
	}
	if req.ProductURL != nil {
		p.ProductURL = strings.TrimSpace(*req.ProductURL)
	}
	if req.CheckoutURL != nil {
		p.CheckoutURL = strings.TrimSpace(*req.CheckoutURL)
	}
	if err := database.DB.Create(&p).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "UNIQUE") {
			return echo.NewHTTPError(http.StatusConflict, "SKU already exists")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create product")
	}
	invalidateProductCache(user.ID)
	return c.JSON(http.StatusCreated, p)
}

// UpdateProduct updates a product by id.
func (h *ProductsHandler) UpdateProduct(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var p models.Product
	if err := database.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&p).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "product not found")
	}

	var req productRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if req.Name != "" {
		p.Name = strings.TrimSpace(req.Name)
	}
	if req.Description != "" || c.Request().ContentLength > 0 {
		p.Description = req.Description
	}
	if req.Price > 0 || req.Price == 0 {
		p.Price = req.Price
	}
	if req.Currency != "" {
		p.Currency = req.Currency
	}
	p.Stock = req.Stock
	if req.Category != "" {
		p.Category = req.Category
	}
	if req.Tags != "" {
		p.Tags = req.Tags
	}
	if req.ProductURL != nil {
		p.ProductURL = strings.TrimSpace(*req.ProductURL)
	}
	if req.CheckoutURL != nil {
		p.CheckoutURL = strings.TrimSpace(*req.CheckoutURL)
	}
	if req.Active != nil {
		p.Active = *req.Active
	}

	if err := database.DB.Save(&p).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update product")
	}
	invalidateProductCache(user.ID)
	database.DB.Preload("Images").First(&p, p.ID)
	return c.JSON(http.StatusOK, p)
}

// DeleteProduct soft-deletes a product.
func (h *ProductsHandler) DeleteProduct(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	result := database.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.Product{})
	if result.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "product not found")
	}
	invalidateProductCache(user.ID)
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// AddProductImage attaches an image URL to a product.
func (h *ProductsHandler) AddProductImage(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	productID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid product id")
	}

	var p models.Product
	if err := database.DB.Where("id = ? AND user_id = ?", productID, user.ID).First(&p).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "product not found")
	}

	var req struct {
		URL       string `json:"url"`
		IsPrimary bool   `json:"is_primary"`
		SortOrder int    `json:"sort_order"`
	}
	if err := c.Bind(&req); err != nil || strings.TrimSpace(req.URL) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "url is required")
	}

	if req.IsPrimary {
		database.DB.Model(&models.ProductImage{}).Where("product_id = ?", p.ID).Update("is_primary", false)
	}

	img := models.ProductImage{
		ProductID: p.ID,
		URL:       strings.TrimSpace(req.URL),
		IsPrimary: req.IsPrimary,
		SortOrder: req.SortOrder,
	}
	if err := database.DB.Create(&img).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to add image")
	}
	invalidateProductCache(user.ID)
	return c.JSON(http.StatusCreated, img)
}

// DeleteProductImage removes an image from a product.
func (h *ProductsHandler) DeleteProductImage(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	productID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	imageID, err := strconv.ParseUint(c.Param("imageId"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid image id")
	}

	var p models.Product
	if err := database.DB.Where("id = ? AND user_id = ?", productID, user.ID).First(&p).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "product not found")
	}

	result := database.DB.Where("id = ? AND product_id = ?", imageID, p.ID).Delete(&models.ProductImage{})
	if result.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "image not found")
	}
	invalidateProductCache(user.ID)
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

type discountRequest struct {
	ProductID *uint   `json:"product_id"`
	Category  string  `json:"category"`
	Type      string  `json:"type"`
	Value     float64 `json:"value"`
	Code      string  `json:"code"`
	StartsAt  *string `json:"starts_at"`
	EndsAt    *string `json:"ends_at"`
	Active    *bool   `json:"active"`
}

// ListDiscounts returns discounts for the user.
func (h *ProductsHandler) ListDiscounts(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var discounts []models.Discount
	database.DB.Where("user_id = ?", user.ID).Order("created_at DESC").Find(&discounts)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"discounts": discounts,
		"total":     len(discounts),
	})
}

// CreateDiscount adds a discount rule.
func (h *ProductsHandler) CreateDiscount(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var req discountRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	dtype := strings.TrimSpace(req.Type)
	if dtype != "percent" && dtype != "fixed" && dtype != "bogo" {
		return echo.NewHTTPError(http.StatusBadRequest, "type must be percent, fixed, or bogo")
	}

	d := models.Discount{
		UserID:    user.ID,
		ProductID: req.ProductID,
		Category:  req.Category,
		Type:      dtype,
		Value:     req.Value,
		Code:      strings.TrimSpace(req.Code),
		Active:    true,
	}
	if req.Active != nil {
		d.Active = *req.Active
	}
	if req.StartsAt != nil && *req.StartsAt != "" {
		t, err := time.Parse(time.RFC3339, *req.StartsAt)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "starts_at must be RFC3339")
		}
		d.StartsAt = &t
	}
	if req.EndsAt != nil && *req.EndsAt != "" {
		t, err := time.Parse(time.RFC3339, *req.EndsAt)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "ends_at must be RFC3339")
		}
		d.EndsAt = &t
	}

	if err := database.DB.Create(&d).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create discount")
	}
	invalidateProductCache(user.ID)
	return c.JSON(http.StatusCreated, d)
}

// UpdateDiscount updates a discount.
func (h *ProductsHandler) UpdateDiscount(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var d models.Discount
	if err := database.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&d).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "discount not found")
	}

	var req discountRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if req.Type != "" {
		d.Type = req.Type
	}
	if req.Value > 0 || req.Value == 0 {
		d.Value = req.Value
	}
	if req.Category != "" {
		d.Category = req.Category
	}
	d.ProductID = req.ProductID
	if req.Code != "" {
		d.Code = req.Code
	}
	if req.Active != nil {
		d.Active = *req.Active
	}

	database.DB.Save(&d)
	invalidateProductCache(user.ID)
	return c.JSON(http.StatusOK, d)
}

// DeleteDiscount removes a discount.
func (h *ProductsHandler) DeleteDiscount(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	result := database.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.Discount{})
	if result.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "discount not found")
	}
	invalidateProductCache(user.ID)
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// DownloadProductTemplate returns CSV/XLSX template for product import.
func (h *ProductsHandler) DownloadProductTemplate(c echo.Context) error {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	headers := []string{"sku", "name", "description", "price", "currency", "stock", "category", "tags", "product_url", "checkout_url"}
	for i, hname := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, hname)
	}
	example := []string{"SKU-001", "Sample Widget", "A great widget", "29.99", "USD", "100", "Gadgets", "widget,bestseller", "https://shop.example.com/p/sku-001", "https://shop.example.com/checkout?sku=SKU-001"}
	for i, v := range example {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sheet, cell, v)
	}
	c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=products_template.xlsx")
	return f.Write(c.Response())
}

// ImportProducts parses CSV or XLSX and bulk-creates products.
func (h *ProductsHandler) ImportProducts(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "file is required")
	}

	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to open file")
	}
	defer src.Close()

	tmpFile, err := os.CreateTemp("", "products_*"+filepath.Ext(file.Filename))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create temp file")
	}
	defer os.Remove(tmpFile.Name())
	if _, err := io.Copy(tmpFile, src); err != nil {
		tmpFile.Close()
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save file")
	}
	tmpFile.Close()

	ext := strings.ToLower(filepath.Ext(file.Filename))
	var products []models.Product
	switch ext {
	case ".csv":
		products, err = parseProductCSV(tmpFile.Name(), user.ID)
	case ".xlsx", ".xls":
		products, err = parseProductXLSX(tmpFile.Name(), user.ID)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "only .csv, .xls, .xlsx supported")
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if len(products) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "no products found in file")
	}

	created := 0
	for _, p := range products {
		var existing models.Product
		if database.DB.Where("user_id = ? AND sku = ?", user.ID, p.SKU).First(&existing).Error == nil {
			existing.Name = p.Name
			existing.Description = p.Description
			existing.Price = p.Price
			existing.Currency = p.Currency
			existing.Stock = p.Stock
			existing.Category = p.Category
			existing.Tags = p.Tags
			existing.ProductURL = p.ProductURL
			existing.CheckoutURL = p.CheckoutURL
			existing.Active = p.Active
			database.DB.Save(&existing)
		} else {
			if err := database.DB.Create(&p).Error; err == nil {
				created++
			}
		}
	}
	invalidateProductCache(user.ID)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "imported",
		"created": created,
		"total":   len(products),
	})
}

func parseProductCSV(filePath string, userID uint) ([]models.Product, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("invalid CSV: %w", err)
	}
	return parseProductRecords(records, userID)
}

func parseProductXLSX(filePath string, userID uint) ([]models.Product, error) {
	xl, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer xl.Close()
	sheet := xl.GetSheetName(0)
	rows, err := xl.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	return parseProductRecords(rows, userID)
}

func parseProductRecords(records [][]string, userID uint) ([]models.Product, error) {
	if len(records) == 0 {
		return nil, nil
	}
	header := records[0]
	col := map[string]int{}
	for i, h := range header {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}
	start := 0
	if _, ok := col["sku"]; ok {
		start = 1
	} else {
		col = map[string]int{"sku": 0, "name": 1, "description": 2, "price": 3, "currency": 4, "stock": 5, "category": 6, "tags": 7, "product_url": 8, "checkout_url": 9}
	}

	var out []models.Product
	for i := start; i < len(records); i++ {
		row := records[i]
		get := func(key string) string {
			idx, ok := col[key]
			if !ok || idx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}
		sku := get("sku")
		name := get("name")
		if sku == "" || name == "" {
			continue
		}
		price, _ := strconv.ParseFloat(get("price"), 64)
		stock, _ := strconv.Atoi(get("stock"))
		currency := get("currency")
		if currency == "" {
			currency = "USD"
		}
		out = append(out, models.Product{
			UserID:      userID,
			SKU:         sku,
			Name:        name,
			Description: get("description"),
			Price:       price,
			Currency:    currency,
			Stock:       stock,
			Category:    get("category"),
			Tags:        get("tags"),
			ProductURL:  get("product_url"),
			CheckoutURL: get("checkout_url"),
			Active:      true,
		})
	}
	return out, nil
}

// ListLeads returns captured leads for the current user.
func (h *ProductsHandler) ListLeads(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var leads []models.Lead
	database.DB.Where("user_id = ?", user.ID).Order("created_at DESC").Limit(200).Find(&leads)

	type leadRow struct {
		models.Lead
		CustomerName  string `json:"customer_name"`
		CustomerPhone string `json:"customer_phone"`
	}
	rows := make([]leadRow, 0, len(leads))
	for _, l := range leads {
		row := leadRow{Lead: l}
		var session models.ChatSession
		if database.DB.First(&session, l.SessionID).Error == nil {
			row.CustomerName = session.CustomerName
			row.CustomerPhone = session.CustomerPhone
		}
		rows = append(rows, row)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"leads": rows,
		"total": len(rows),
	})
}
