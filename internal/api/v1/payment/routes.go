package payment

import (
	"aigentools-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	h := NewHandler()

	// Public notify route
	// /api/v1/payment/notify/:uuid
	r.Any("/payment/notify/:uuid", h.Notify)

	// Protected payment routes
	auth := r.Group("/payment")
	auth.Use(middleware.AuthMiddleware())
	{
		auth.GET("/methods", h.GetPaymentMethods)
		auth.POST("/create", h.CreatePayment)
	}
}
