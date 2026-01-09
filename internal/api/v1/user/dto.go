package user

import "time"

// UserResponse defines the response structure for user information.
type UserResponse struct {
	ID            uint        `json:"id"`
	Username      string      `json:"username"`
	Role          string      `json:"role"`
	IsActive      bool        `json:"is_active"`
	ActivatedAt   *time.Time  `json:"activated_at,omitempty"`
	DeactivatedAt *time.Time  `json:"deactivated_at,omitempty"`
	CreditLimit   float64     `json:"creditLimit"`
	TotalConsumed float64     `json:"total_consumed"`
	Credit        *CreditInfo `json:"credit,omitempty"`
	Token         string      `json:"token,omitempty"`
}

// CreditInfo defines the structure for credit details
type CreditInfo struct {
	Total           float64 `json:"total"`
	Used            float64 `json:"used"`
	Available       float64 `json:"available"`
	UsagePercentage float64 `json:"usagePercentage"`
}
