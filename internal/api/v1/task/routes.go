package task

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup) {
	tasks := router.Group("/tasks")
	{
		tasks.POST("", SubmitTask)
		tasks.GET("", ListTasks)
		tasks.GET("/:id", GetTaskDetail)
		tasks.POST("/:id/retry", RetryTask)
		tasks.PATCH("/:id/approve", ApproveTask)
		tasks.PUT("/:id", UpdateTask)
	}
}
