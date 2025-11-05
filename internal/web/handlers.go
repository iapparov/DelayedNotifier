package web

import (
	"delayedNotifier/internal/app"
	wbgin "github.com/wb-go/wbf/ginext"
	wbzlog "github.com/wb-go/wbf/zlog"
	"net/http"
)

type NotifyHandler struct {
	repo  StorageProvider
	cache CacheProvider
}

type StorageProvider interface {
	SaveNotification(notification *app.Notification) error
	GetNotification(id string) (*app.Notification, error)
	DeleteNotification(id string) error
}

type CacheProvider interface {
	SaveNotification(notification *app.Notification) error
	GetNotification(id string) (*app.Notification, error)
	DeleteNotification(id string) error
}

func NewNotifyHandler(repo StorageProvider, cache CacheProvider) *NotifyHandler {
	return &NotifyHandler{repo: repo, cache: cache}
}

// ErrorResponse представляет стандартную ошибку API
type ErrorResponse struct {
	Error string `json:"error" example:"invalid input data"`
}

// Create Notification godoc
// @Summary      Create Notification
// @Description  Создает новое уведомление (Email, Telegram) и сохраняет его в БД и Redis
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Param        notification  body  app.NotificationRequest  true  "Notification to create"
// @Success      201  {object}  app.Notification  "Created notification"
// @Failure      400  {object}  ErrorResponse  "Invalid input data"
// @Failure      503  {object}  ErrorResponse  "Service unavailable (DB or cache)"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Router       /notify [post]
func (h *NotifyHandler) CreateNotification(ctx *wbgin.Context) {
	var req app.NotificationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, wbgin.H{"error": err.Error()})
		return
	}

	notif, err := app.NewNotification(req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, wbgin.H{"error": err.Error()})
		return
	}
	err = h.repo.SaveNotification(notif)
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, wbgin.H{"error": err.Error()})
		return
	}
	err = h.cache.SaveNotification(notif)
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, wbgin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, notif)
}

// Get Notification godoc
// @Summary      Get Notification
// @Description  Получает уведомление по ID (из кэша или базы данных)
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Param        id   path   string  true  "Notification ID"
// @Success      200  {object}  app.Notification  "Notification object"
// @Failure      400  {object}  ErrorResponse  "Invalid notification ID"
// @Failure      404  {object}  ErrorResponse  "Notification not found"
// @Failure      503  {object}  ErrorResponse  "Service unavailable"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Router       /notify/{id} [get]
func (h *NotifyHandler) GetNotification(ctx *wbgin.Context) {
	id := ctx.Param("id")

	if !app.IsValidUUID(id) {
		ctx.JSON(http.StatusBadRequest, wbgin.H{"error": "id is invalid"})
		return
	}

	notification, err := h.cache.GetNotification(id)
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, wbgin.H{"error": err.Error()})
		return
	}
	if notification == nil {
		notification, err := h.repo.GetNotification(id)
		if err != nil {
			ctx.JSON(http.StatusServiceUnavailable, wbgin.H{"error": err.Error()})
			return
		}
		if notification == nil {
			ctx.JSON(http.StatusNotFound, wbgin.H{"error": "id not found"})
			return
		}
		if err := h.cache.SaveNotification(notification); err != nil {
			wbzlog.Logger.Error().
				Err(err).
				Str("id", notification.ID.String()).
				Msg("Failed to save notification to cache")
		}
	}
	ctx.JSON(http.StatusOK, notification.Status)
}

// Delete Notification godoc
// @Summary      Delete Notification
// @Description  Удаляет уведомление по ID из кэша и базы данных
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Param        id   path   string  true  "Notification ID"
// @Success      204  {string}  string  "Notification deleted successfully"
// @Failure      400  {object}  ErrorResponse  "Invalid notification ID"
// @Failure      503  {object}  ErrorResponse  "Service unavailable"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Router       /notify/{id} [delete]
func (h *NotifyHandler) DeleteNotification(ctx *wbgin.Context) {
	id := ctx.Param("id")
	if !app.IsValidUUID(id) {
		ctx.JSON(http.StatusBadRequest, wbgin.H{"error": "id is invalid"})
		return
	}
	err := h.cache.DeleteNotification(id)
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, wbgin.H{"error": err.Error()})
	}

	err = h.repo.DeleteNotification(id)
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, wbgin.H{"error": err.Error()})
	}

	ctx.Status(http.StatusNoContent)
}
