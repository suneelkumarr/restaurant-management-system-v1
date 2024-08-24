package routes

import (
	"restorent-management/controllers"

	"github.com/gin-gonic/gin"
)

// UserRoutes sets up the user-related routes
func UserRoutes(router *gin.Engine) {
	userGroup := router.Group("/users")
	{
		userGroup.GET("", controllers.GetUsers())
		userGroup.GET("/:user_id", controllers.GetUser())
		userGroup.POST("/signup", controllers.SignUp())
		userGroup.POST("/login", controllers.Login())
		userGroup.PUT("/update/:user_id", controllers.UpdateUser())
	}
}
