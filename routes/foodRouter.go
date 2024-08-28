package routes

import (
	"restorent-management/controllers"

	"github.com/gin-gonic/gin"
)

func FoodRoutes(router *gin.Engine) {
	foodGroup := router.Group("/foods")
	{
		foodGroup.GET("", controllers.GetFoods())
		foodGroup.GET("/:food_id", controllers.GetFoodByID())
		foodGroup.POST("/create", controllers.CreateFood())
		// foodGroup.PATCH("/:food_id", controllers.UpdateFood())
	}
}
