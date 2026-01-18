package order

import "time"

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	UserID uint    `json:"user_id" binding:"required"`
	Amount float64 `json:"amount" binding:"required,gt=0"`
	Remark string  `json:"remark"`
}

// OrderListItem 订单列表项
type OrderListItem struct {
	ID          string     `json:"id"`
	UserID      uint       `json:"user_id"`
	Username    string     `json:"username,omitempty"`
	Amount      float64    `json:"amount"`
	Status      string     `json:"status"`
	OrderType   string     `json:"order_type"`
	PaymentUUID string     `json:"payment_uuid,omitempty"`
	ExternalID  string     `json:"external_id,omitempty"`
	Remark      string     `json:"remark,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CompletedBy uint       `json:"completed_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// OrderListResponse 订单列表响应
type OrderListResponse struct {
	Orders []OrderListItem `json:"orders"`
	Total  int64           `json:"total"`
	Page   int             `json:"page"`
	Limit  int             `json:"limit"`
}

// OrderDetailResponse 订单详情响应
type OrderDetailResponse struct {
	OrderListItem
	User *UserBrief `json:"user,omitempty"`
}

// UserBrief 用户简要信息
type UserBrief struct {
	ID       uint    `json:"id"`
	Username string  `json:"username"`
	Balance  float64 `json:"balance"`
}
