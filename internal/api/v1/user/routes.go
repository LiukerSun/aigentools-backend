package user

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	auth.GET("/user", CurrentUser)
}
