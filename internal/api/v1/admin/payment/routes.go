package payment

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.RouterGroup) {
	h := NewHandler()

	paymentGroup := r.Group("/payment")
	{
		paymentGroup.GET("/config", h.ListPaymentConfigs)
		paymentGroup.POST("/config", h.CreatePaymentConfig)
		paymentGroup.PUT("/config/:id", h.UpdatePaymentConfig)
		paymentGroup.DELETE("/config/:id", h.DeletePaymentConfig)
	}
}
