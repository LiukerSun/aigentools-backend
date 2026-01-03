package transaction

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/transactions", ListTransactions)
	router.GET("/transactions/export", ExportTransactions)
}
