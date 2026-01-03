package ai_model

import (
	"aigentools-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup) {
	modelGroup := router.Group("/models")
	modelGroup.Use(middleware.AuthMiddleware())
	{
		modelGroup.GET("", GetModels)
		modelGroup.PATCH("/:id/status", UpdateModelStatus)
		modelGroup.POST("/create", CreateModel)
	}
}
