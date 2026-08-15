package handlers

import (
	"net/http"
	"strconv"

	"smart-live-chats/internal/config"
	"smart-live-chats/internal/database"
	"smart-live-chats/internal/middleware"
	"smart-live-chats/internal/models"
	"smart-live-chats/internal/whatsapp"

	"github.com/labstack/echo/v4"
)

type WhatsAppHandler struct {
	config *config.Config
}

func NewWhatsAppHandler(cfg *config.Config) *WhatsAppHandler {
	return &WhatsAppHandler{config: cfg}
}

func (h *WhatsAppHandler) Connect(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	// Enforce the per-plan device limit only when billing is enabled. In free
	// self-hosted mode users may connect unlimited WhatsApp devices.
	if h.config != nil && h.config.BillingEnabled {
		var connectedCount int64
		database.DB.Model(&models.WhatsAppDevice{}).
			Where("user_id = ? AND status = ?", user.ID, "connected").
			Count(&connectedCount)

		if int(connectedCount) >= user.MaxDevices {
			return echo.NewHTTPError(http.StatusBadRequest,
				"Device limit reached. You can connect up to "+strconv.Itoa(user.MaxDevices)+" devices. Upgrade your plan for more.")
		}
	}

	if _, err := whatsapp.GlobalManager.Connect(user.ID); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "QR code will be sent via WebSocket",
		"status":  "scanning",
	})
}

func (h *WhatsAppHandler) Disconnect(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	deviceIDStr := c.Param("deviceId")
	if deviceIDStr == "" {
		// Legacy: disconnect all devices for this user
		var devices []models.WhatsAppDevice
		database.DB.Where("user_id = ?", user.ID).Find(&devices)
		for _, d := range devices {
			whatsapp.GlobalManager.LogoutDevice(d.ID)
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "disconnected"})
	}

	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid device ID")
	}

	var device models.WhatsAppDevice
	if err := database.DB.Where("id = ? AND user_id = ?", deviceID, user.ID).First(&device).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "device not found")
	}

	if err := whatsapp.GlobalManager.LogoutDevice(uint(deviceID)); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "disconnected"})
}

func (h *WhatsAppHandler) GetStatus(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var devices []models.WhatsAppDevice
	database.DB.Where("user_id = ?", user.ID).Find(&devices)

	deviceStatuses := make([]map[string]interface{}, 0)
	for _, d := range devices {
		connected := whatsapp.GlobalManager.IsDeviceConnected(d.ID)
		status := "disconnected"
		if connected {
			status = "connected"
		}
		deviceStatuses = append(deviceStatuses, map[string]interface{}{
			"device_id": d.ID,
			"status":    status,
			"phone":     d.Phone,
			"push_name": d.PushName,
			"jid":       d.JID,
		})
	}

	anyConnected := whatsapp.GlobalManager.IsUserConnected(user.ID)
	overallStatus := "disconnected"
	if anyConnected {
		overallStatus = "connected"
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  overallStatus,
		"devices": deviceStatuses,
	})
}

func (h *WhatsAppHandler) GetDevices(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var devices []models.WhatsAppDevice
	database.DB.Where("user_id = ?", user.ID).Find(&devices)

	for i := range devices {
		if devices[i].Status == "connected" && !whatsapp.GlobalManager.IsDeviceConnected(devices[i].ID) {
			devices[i].Status = "disconnected"
			database.DB.Model(&devices[i]).Update("status", "disconnected")
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"devices": devices,
	})
}
