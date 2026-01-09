package ai_assistant

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.RouterGroup) {
	aiGroup := router.Group("/ai-assistant")
	{
		aiGroup.POST("/analyze", AnalyzeImage)
	}
}
