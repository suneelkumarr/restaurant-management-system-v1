package routes

import (
	"restorent-management/controllers"

	"github.com/gin-gonic/gin"
)

func OrderRoutes(router *gin.Engine) {
	orderGroup := router.Group("/orders")
	{
		orderGroup.GET("", controllers.GetOrders())
		orderGroup.GET("/:order_id", controllers.GetOrderByID())
		orderGroup.POST("/orders", controllers.CreateOrder())
		orderGroup.PATCH("/:order_id", controllers.UpdateOrder())
	}
}
