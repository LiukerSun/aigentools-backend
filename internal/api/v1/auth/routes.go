package auth

import (
	"aigentools-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	auth.POST("/register", Register)
	auth.POST("/login", Login)
	auth.POST("/logout", middleware.AuthMiddleware(), Logout)
}
