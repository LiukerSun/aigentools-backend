package order

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.RouterGroup) {
	h := NewHandler()

	orderGroup := r.Group("/orders")
	{
		orderGroup.GET("", h.ListOrders)
		orderGroup.GET("/:id", h.GetOrder)
		orderGroup.POST("", h.CreateOrder)
		orderGroup.POST("/:id/complete", h.CompleteOrder)
		orderGroup.POST("/:id/cancel", h.CancelOrder)
	}
}
