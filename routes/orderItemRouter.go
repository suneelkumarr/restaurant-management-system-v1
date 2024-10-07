package routes

import (
	"restorent-management/controllers"

	"github.com/gin-gonic/gin"
)

func OrderItemRoutes(router *gin.Engine) {
	orderItemGroup := router.Group("/orderItems")
	{
		orderItemGroup.GET("", controllers.GetOrderItems())
		orderItemGroup.GET("/:ordeitem_id", controllers.GetOrderItemsByID())
		orderItemGroup.POST("/create", controllers.CreateOrderItems())
		orderItemGroup.PATCH("/:ordeitem_id", controllers.UpdateOrderItems())
	}
}
