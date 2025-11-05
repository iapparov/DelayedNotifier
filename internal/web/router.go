package web

import (
	wbgin "github.com/wb-go/wbf/ginext"
	httpSwagger "github.com/swaggo/http-swagger"
    _ "delayedNotifier/docs"
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
