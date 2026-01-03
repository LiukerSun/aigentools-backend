package user

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/users", ListUsers)
	router.PATCH("/users/:id", UpdateUser)
	router.POST("/users/:id/balance", AdjustBalance)
	router.PUT("/users/:id/balance", DeductBalance)
}
