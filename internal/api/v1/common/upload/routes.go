package upload

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/common/upload")
	{
		group.GET("/token", GetOSSToken)
	}
}
