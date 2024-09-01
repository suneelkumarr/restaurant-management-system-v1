package routes

import (
	"restorent-management/controllers"

	"github.com/gin-gonic/gin"
)

func TableRoutes(router *gin.Engine) {
	tableGroup := router.Group("/tables")
	{
		tableGroup.GET("", controllers.GetTables())
		tableGroup.GET("/:table_id", controllers.GetTableByID())
		tableGroup.POST("/create", controllers.CreateTable())
		tableGroup.PATCH("/:table_id", controllers.UpdateTable())
	}
}
