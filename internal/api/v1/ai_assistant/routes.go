package ai_assistant

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.RouterGroup) {
	aiGroup := router.Group("/ai-assistant")
	{
		aiGroup.POST("/analyze", AnalyzeImage)

		// Prompt management
		aiGroup.POST("/prompts", CreatePrompt)
		aiGroup.POST("/prompts/batch", BatchCreatePrompts)
		aiGroup.PUT("/prompts/:code", UpdatePrompt)
		aiGroup.DELETE("/prompts/:code", DeletePrompt)
		aiGroup.GET("/prompts/:code", GetPrompt)
		aiGroup.GET("/prompts", ListPrompts)

		// Template management
		aiGroup.POST("/templates", CreateTemplate)
		aiGroup.GET("/templates", ListTemplates)
		aiGroup.GET("/templates/:id", GetTemplate)
		aiGroup.PUT("/templates/:id", UpdateTemplate)
		aiGroup.DELETE("/templates/:id", DeleteTemplate)
	}
}
