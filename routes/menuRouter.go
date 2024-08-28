package routes

import (
	"restorent-management/controllers"

	"github.com/gin-gonic/gin"
)

func MenuRoutes(router *gin.Engine) {
	menuGroup := router.Group("/menus")
	{
		menuGroup.GET("", controllers.GetMenus())
		menuGroup.GET("/:menu_id", controllers.GetMenuByID())
		menuGroup.POST("/create", controllers.CreateMenu())
		menuGroup.PATCH("/:menu_id", controllers.UpdateMenu())
	}
}
