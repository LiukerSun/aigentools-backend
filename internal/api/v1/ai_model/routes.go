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
		modelGroup.GET("/names", GetModelNames)
		modelGroup.GET("/:id/parameters", GetModelParameters)
		modelGroup.PATCH("/:id/status", UpdateModelStatus)
		modelGroup.PUT("/:id", UpdateModel)
		modelGroup.POST("/create", CreateModel)
	}
}
