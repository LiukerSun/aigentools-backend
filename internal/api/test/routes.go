package test

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/task", CreateTestTaskHandler)
}
