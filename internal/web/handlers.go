package web

import(
	"DelayedNotifier/internal/app"
	wbgin "github.com/wb-go/wbf/ginext"
	"net/http"
)

type NotifyHandler struct {
	repo StorageProvider
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

func NewNotifyHandler(repo StorageProvider, cache CacheProvider) *NotifyHandler{
	return &NotifyHandler{repo: repo, cache: cache}
}

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

func (h* NotifyHandler) GetNotification(ctx *wbgin.Context) {
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
			ctx.JSON(http.StatusServiceUnavailable, wbgin.H{"error": "id not found"})
			return
		}
		h.cache.SaveNotification(notification)
	}
	ctx.JSON(http.StatusOK, notification.Status)
}


func (h* NotifyHandler) DeleteNotification(ctx *wbgin.Context) {
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
	
	ctx.JSON(http.StatusOK, nil)
}