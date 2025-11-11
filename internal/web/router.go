package web

import (
	_ "delayedNotifier/docs"
	httpSwagger "github.com/swaggo/http-swagger"
	wbgin "github.com/wb-go/wbf/ginext"
)

func RegisterRoutes(engine *wbgin.Engine, handler *NotifyHandler) {
	api := engine.Group("")
	{
		api.POST("/notify", handler.CreateNotification)
		api.GET("/notify/:id", handler.GetNotification)
		api.DELETE("/notify/:id", handler.DeleteNotification)
		api.GET("/swagger/*any", func(c *wbgin.Context) {
			httpSwagger.WrapHandler(c.Writer, c.Request)
		})
	}
}
