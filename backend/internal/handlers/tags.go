package handlers

import (
	"net/http"
	"strconv"

	"smart-live-chats/internal/database"
	"smart-live-chats/internal/middleware"
	"smart-live-chats/internal/models"

	"github.com/labstack/echo/v4"
)

type TagHandler struct{}

func NewTagHandler() *TagHandler {
	return &TagHandler{}
}

func (h *TagHandler) ListTags(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var tags []models.Tag
	database.DB.Where("user_id = ?", user.ID).Order("name ASC").Find(&tags)
	return c.JSON(http.StatusOK, tags)
}

func (h *TagHandler) CreateTag(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var req struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := c.Bind(&req); err != nil || req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}
	if req.Color == "" {
		req.Color = "blue"
	}

	tag := models.Tag{UserID: user.ID, Name: req.Name, Color: req.Color}
	if err := database.DB.Create(&tag).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create tag")
	}

	return c.JSON(http.StatusCreated, tag)
}

func (h *TagHandler) DeleteTag(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid ID")
	}

	database.DB.Where("tag_id = ?", id).Delete(&models.SessionTag{})
	database.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.Tag{})

	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *TagHandler) GetSessionTags(c echo.Context) error {
	sessionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid session ID")
	}

	var sessionTags []models.SessionTag
	database.DB.Where("session_id = ?", sessionID).Preload("Tag").Find(&sessionTags)

	tags := make([]models.Tag, 0)
	for _, st := range sessionTags {
		if st.Tag != nil {
			tags = append(tags, *st.Tag)
		}
	}

	return c.JSON(http.StatusOK, tags)
}

func (h *TagHandler) AddTagToSession(c echo.Context) error {
	sessionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid session ID")
	}

	var req struct {
		TagID uint `json:"tag_id"`
	}
	if err := c.Bind(&req); err != nil || req.TagID == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "tag_id is required")
	}

	var existing models.SessionTag
	if database.DB.Where("session_id = ? AND tag_id = ?", sessionID, req.TagID).First(&existing).Error == nil {
		return c.JSON(http.StatusOK, map[string]string{"status": "already tagged"})
	}

	st := models.SessionTag{SessionID: uint(sessionID), TagID: req.TagID}
	database.DB.Create(&st)

	return c.JSON(http.StatusCreated, map[string]string{"status": "tagged"})
}

func (h *TagHandler) RemoveTagFromSession(c echo.Context) error {
	sessionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid session ID")
	}

	tagID, err := strconv.ParseUint(c.Param("tagId"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid tag ID")
	}

	database.DB.Where("session_id = ? AND tag_id = ?", sessionID, tagID).Delete(&models.SessionTag{})
	return c.JSON(http.StatusOK, map[string]string{"status": "removed"})
}
