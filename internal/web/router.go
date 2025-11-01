package web

import (
	wbgin "github.com/wb-go/wbf/ginext"
)


func RegisterRoutes(engine *wbgin.Engine, handler *NotifyHandler) {
	api := engine.Group("")
	{
		api.POST("/notify", handler.CreateNotification)
		api.GET("/notify/:id", handler.GetNotification)
		api.DELETE("/notify/:id", handler.DeleteNotification)
	}
}		

/* 

func main() {
	cfg := config.Load()

	engine := ginext.New(cfg.Mode)
	engine.Use(ginext.Logger(), ginext.Recovery())

	// инициализация бизнес-логики (сервис уведомлений)
	service := app.NewNotificationService(cfg)

	// создаём хендлер и регистрируем маршруты
	handler := httptransport.NewHandler(service)
	httptransport.RegisterRoutes(engine, handler)

	engine.Run(":" + cfg.HTTPPort)
}

*/