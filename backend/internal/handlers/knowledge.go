package handlers

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"smart-live-chats/internal/database"
	"smart-live-chats/internal/gemini"
	"smart-live-chats/internal/middleware"
	"smart-live-chats/internal/models"

	"github.com/labstack/echo/v4"
	"github.com/xuri/excelize/v2"
)

func autoSync(userID uint) {
	if err := gemini.GlobalService.SyncQAItems(userID); err != nil {
		log.Printf("Auto-sync failed for user %d: %v", userID, err)
	}
}

type KnowledgeHandler struct{}

func NewKnowledgeHandler() *KnowledgeHandler {
	return &KnowledgeHandler{}
}

type QAItemRequest struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Category string `json:"category"`
}

// ListQAItems returns all QA items for the current user
func (h *KnowledgeHandler) ListQAItems(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var items []models.QAItem
	database.DB.Where("user_id = ?", user.ID).Order("category ASC, created_at DESC").Find(&items)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"items": items,
		"total": len(items),
	})
}

// CreateQAItem adds a new QA pair
func (h *KnowledgeHandler) CreateQAItem(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var req QAItemRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if req.Question == "" || req.Answer == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "question and answer are required")
	}

	item := models.QAItem{
		UserID:   user.ID,
		Question: req.Question,
		Answer:   req.Answer,
		Category: req.Category,
	}
	if err := database.DB.Create(&item).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create QA item")
	}

	go autoSync(user.ID)
	return c.JSON(http.StatusCreated, item)
}

// UpdateQAItem updates an existing QA pair
func (h *KnowledgeHandler) UpdateQAItem(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid ID")
	}

	var item models.QAItem
	if err := database.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&item).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "QA item not found")
	}

	var req QAItemRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	if req.Question != "" {
		item.Question = req.Question
	}
	if req.Answer != "" {
		item.Answer = req.Answer
	}
	item.Category = req.Category

	database.DB.Save(&item)
	go autoSync(user.ID)
	return c.JSON(http.StatusOK, item)
}

// DeleteQAItem removes a QA pair
func (h *KnowledgeHandler) DeleteQAItem(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid ID")
	}

	result := database.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.QAItem{})
	if result.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "QA item not found")
	}

	go autoSync(user.ID)
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// DeleteAllQAItems removes all QA items for the authenticated user
func (h *KnowledgeHandler) DeleteAllQAItems(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	result := database.DB.Where("user_id = ?", user.ID).Delete(&models.QAItem{})
	go autoSync(user.ID)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "deleted",
		"deleted": result.RowsAffected,
	})
}

// SyncToGemini syncs all QA items to the Gemini File Search Store
func (h *KnowledgeHandler) SyncToGemini(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	if err := gemini.GlobalService.SyncQAItems(user.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "synced"})
}

// DownloadTemplate returns an XLSX template file with the required columns
func (h *KnowledgeHandler) DownloadTemplate(c echo.Context) error {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)

	headers := []string{"category", "question", "answer"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#D9E1F2"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A1", "C1", style)

	example := []string{"Shipping", "How long does delivery take?", "Standard delivery takes 3-5 business days."}
	for i, v := range example {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sheet, cell, v)
	}

	f.SetColWidth(sheet, "A", "A", 18)
	f.SetColWidth(sheet, "B", "B", 40)
	f.SetColWidth(sheet, "C", "C", 50)

	c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=qa_template.xlsx")
	if err := f.Write(c.Response()); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate template")
	}
	return nil
}

// UploadFile handles file upload. CSV/XLSX are parsed into Q&A items; other files go to Gemini directly.
func (h *KnowledgeHandler) UploadFile(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	log.Println("UploadFile handler called")
	file, err := c.FormFile("file")
	if err != nil {
		log.Printf("FormFile error: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "file is required")
	}
	log.Printf("Got file: %s, size: %d", file.Filename, file.Size)

	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to open file")
	}
	defer src.Close()

	tmpFile, err := os.CreateTemp("", "upload_*"+filepath.Ext(file.Filename))
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
	log.Printf("Upload file: %s, ext: %s", file.Filename, ext)

	if ext != ".xlsx" && ext != ".xls" {
		return echo.NewHTTPError(http.StatusBadRequest, "Only .xls and .xlsx files are supported. Please download the template and use the correct format.")
	}

	items, parseErr := parseXLSX(tmpFile.Name(), user.ID)
	if parseErr != nil {
		return echo.NewHTTPError(http.StatusBadRequest, parseErr.Error())
	}

	if len(items) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "No Q&A items found in the file. Make sure your data starts from row 2.")
	}

	for _, item := range items {
		database.DB.Create(&item)
	}

	go autoSync(user.ID)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":   "imported",
		"filename": file.Filename,
		"count":    len(items),
	})
}

func parseCSV(filePath string, userID uint) ([]models.QAItem, error) {
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
		return nil, fmt.Errorf("invalid CSV format: %w", err)
	}

	if len(records) == 0 {
		return nil, nil
	}

	// Detect header row
	startRow := 0
	qCol, aCol, cCol := detectColumns(records[0])
	if qCol >= 0 {
		startRow = 1
	} else {
		// No header: assume col 0 = question, col 1 = answer, col 2 = category
		qCol, aCol, cCol = 0, 1, 2
	}

	var items []models.QAItem
	for i := startRow; i < len(records); i++ {
		row := records[i]
		if qCol >= len(row) || aCol >= len(row) {
			continue
		}
		q := strings.TrimSpace(row[qCol])
		a := strings.TrimSpace(row[aCol])
		if q == "" || a == "" {
			continue
		}
		category := ""
		if cCol >= 0 && cCol < len(row) {
			category = strings.TrimSpace(row[cCol])
		}
		items = append(items, models.QAItem{
			UserID:   userID,
			Question: q,
			Answer:   a,
			Category: category,
		})
	}

	return items, nil
}

func parseXLSX(filePath string, userID uint) ([]models.QAItem, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("invalid Excel file: %w", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, fmt.Errorf("no sheets found")
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, err
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("Invalid file format. The file must have a header row with columns: category, question, answer")
	}

	header := rows[0]
	if len(header) < 3 {
		return nil, fmt.Errorf("Invalid file format. Expected 3 columns (category, question, answer) but found %d. Please download and use the template.", len(header))
	}

	cCol, qCol, aCol := -1, -1, -1
	for i, h := range header {
		lower := strings.ToLower(strings.TrimSpace(h))
		switch lower {
		case "category":
			cCol = i
		case "question":
			qCol = i
		case "answer":
			aCol = i
		}
	}

	if qCol < 0 || aCol < 0 || cCol < 0 {
		found := make([]string, len(header))
		for i, h := range header {
			found[i] = strings.TrimSpace(h)
		}
		return nil, fmt.Errorf("Invalid column headers. Expected: category, question, answer. Found: %s. Please download and use the template.", strings.Join(found, ", "))
	}

	var items []models.QAItem
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if qCol >= len(row) || aCol >= len(row) {
			continue
		}
		q := strings.TrimSpace(row[qCol])
		a := strings.TrimSpace(row[aCol])
		if q == "" || a == "" {
			continue
		}
		category := ""
		if cCol >= 0 && cCol < len(row) {
			category = strings.TrimSpace(row[cCol])
		}
		items = append(items, models.QAItem{
			UserID:   userID,
			Question: q,
			Answer:   a,
			Category: category,
		})
	}

	return items, nil
}

// detectColumns finds question/answer/category columns from a header row
func detectColumns(header []string) (qCol, aCol, cCol int) {
	qCol, aCol, cCol = -1, -1, -1
	for i, h := range header {
		lower := strings.ToLower(strings.TrimSpace(h))
		switch {
		case strings.Contains(lower, "question") || lower == "q":
			qCol = i
		case strings.Contains(lower, "answer") || lower == "a":
			aCol = i
		case strings.Contains(lower, "category") || strings.Contains(lower, "tag") || strings.Contains(lower, "topic"):
			cCol = i
		}
	}
	if qCol >= 0 && aCol >= 0 {
		return qCol, aCol, cCol
	}
	return -1, -1, -1
}

// GetKnowledgeBase returns the user's knowledge base config
func (h *KnowledgeHandler) GetKnowledgeBase(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var kb models.KnowledgeBase
	if err := database.DB.Where("user_id = ?", user.ID).First(&kb).Error; err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"auto_reply_enabled": false,
			"system_prompt":      "",
			"synced":             false,
		})
	}

	return c.JSON(http.StatusOK, kb)
}

// UpdateKnowledgeBase updates the knowledge base settings
func (h *KnowledgeHandler) UpdateKnowledgeBase(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var req struct {
		AutoReplyEnabled *bool  `json:"auto_reply_enabled"`
		SystemPrompt     string `json:"system_prompt"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	var kb models.KnowledgeBase
	if err := database.DB.Where("user_id = ?", user.ID).First(&kb).Error; err != nil {
		kb = models.KnowledgeBase{UserID: user.ID}
		if err := database.DB.Create(&kb).Error; err != nil {
			log.Printf("UpdateKnowledgeBase: failed to create KB for user %d: %v", user.ID, err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create knowledge base")
		}
	}

	// Build a map of columns to update so we only touch fields the caller
	// explicitly sent. Using Updates(map) also avoids any GORM zero-value
	// quirks with Save() for booleans like auto_reply_enabled=false.
	updates := map[string]interface{}{}
	if req.AutoReplyEnabled != nil {
		updates["auto_reply_enabled"] = *req.AutoReplyEnabled
	}
	if req.SystemPrompt != "" {
		updates["system_prompt"] = req.SystemPrompt
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&kb).Updates(updates).Error; err != nil {
			log.Printf("UpdateKnowledgeBase: failed to persist updates for user %d: %v", user.ID, err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to update knowledge base")
		}
		// Refresh the local struct so the response reflects what's in DB.
		database.DB.Where("user_id = ?", user.ID).First(&kb)
		if req.AutoReplyEnabled != nil {
			log.Printf("UpdateKnowledgeBase: user %d auto_reply_enabled -> %v (kb id=%d)", user.ID, kb.AutoReplyEnabled, kb.ID)
		}
		// A changed system prompt is baked into the cached Gemini context, so
		// drop the cache to make the new prompt take effect on the next message.
		if req.SystemPrompt != "" && gemini.GlobalService != nil {
			gemini.GlobalService.InvalidateUserContextCache(user.ID)
		}
	}

	return c.JSON(http.StatusOK, kb)
}

// TestQuery tests a question against the knowledge base
func (h *KnowledgeHandler) TestQuery(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var req struct {
		Question string `json:"question"`
	}
	if err := c.Bind(&req); err != nil || req.Question == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "question is required")
	}

	answer, err := gemini.GlobalService.GetAnswer(user.ID, req.Question)
	if err != nil {
		log.Printf("TestQuery: AI error for user %d: %v", user.ID, err)
		return echo.NewHTTPError(http.StatusServiceUnavailable, gemini.FriendlyErrorMessage(err))
	}

	return c.JSON(http.StatusOK, map[string]string{
		"question": req.Question,
		"answer":   answer,
	})
}

// testChatTurn mirrors a single message in the test sandbox conversation.
type testChatTurn struct {
	Role    string `json:"role"`    // "user" (customer) or "bot"/"model"/"agent"
	Content string `json:"content"`
}

// TestChat runs a multi-turn conversation against the user's bot exactly like a
// real WhatsApp customer would experience it (same AI pipeline, product tools,
// discounts, and multi-turn context). The conversation is ephemeral: the client
// holds the transcript and replays it on each turn, so nothing is written to the
// real chat sessions / leads / reports.
func (h *KnowledgeHandler) TestChat(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var req struct {
		Message string         `json:"message"`
		History []testChatTurn `json:"history"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if strings.TrimSpace(req.Message) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "message is required")
	}

	// Rebuild the conversation as Gemini-style turns, then append the new
	// customer message as the latest user turn.
	history := make([]gemini.ChatTurn, 0, len(req.History)+1)
	for _, t := range req.History {
		content := strings.TrimSpace(t.Content)
		if content == "" {
			continue
		}
		role := "user"
		if t.Role == "bot" || t.Role == "model" || t.Role == "agent" {
			role = "model"
		}
		history = append(history, gemini.ChatTurn{Role: role, Content: content})
	}
	history = append(history, gemini.ChatTurn{Role: "user", Content: req.Message})

	// Ephemeral tool context: no device (so no real image is sent) and no
	// persisted session (so no real lead/profile is written).
	gemini.GlobalService.SetToolContext(gemini.ToolContext{UserID: user.ID})

	answer, err := gemini.GlobalService.GetAnswerForConversation(user.ID, req.Message, history, 0)
	if err != nil {
		log.Printf("TestChat: AI error for user %d: %v", user.ID, err)
		return echo.NewHTTPError(http.StatusServiceUnavailable, gemini.FriendlyErrorMessage(err))
	}

	return c.JSON(http.StatusOK, map[string]string{
		"reply": answer,
	})
}
