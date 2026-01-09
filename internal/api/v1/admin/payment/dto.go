package payment

type CreatePaymentConfigRequest struct {
	Name          string                 `json:"name" binding:"required"`
	PaymentMethod string                 `json:"payment_method" binding:"required"` // e.g. "epay"
	Config        map[string]interface{} `json:"config" binding:"required"`
	Enable        bool                   `json:"enable"`
}

type UpdatePaymentConfigRequest struct {
	Name   string                 `json:"name"`
	Config map[string]interface{} `json:"config"`
	Enable *bool                  `json:"enable"` // Pointer to allow false
}

type PaymentConfigResponse struct {
	ID            uint                   `json:"id"`
	UUID          string                 `json:"uuid"`
	Name          string                 `json:"name"`
	PaymentMethod string                 `json:"payment_method"`
	Config        map[string]interface{} `json:"config"`
	Enable        bool                   `json:"enable"`
	CreatedAt     string                 `json:"created_at"`
	UpdatedAt     string                 `json:"updated_at"`
}
