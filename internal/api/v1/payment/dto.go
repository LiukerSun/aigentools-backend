package payment

type CreatePaymentRequest struct {
	Amount            float64 `json:"amount" binding:"required,gt=0"`
	PaymentMethodUUID string  `json:"payment_method_uuid" binding:"required"`
	ReturnURL         string  `json:"return_url" binding:"required"`
}

type CreatePaymentResponse struct {
	JumpURL string `json:"jump_url"`
	OrderID string `json:"order_id"`
}

type PaymentMethodResponse struct {
	UUID string `json:"uuid"`
	Type string `json:"type"` // e.g., "epay"
	Name string `json:"name"` // For now, we might just use the type or a placeholder
}
