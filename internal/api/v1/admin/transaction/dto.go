package transaction

import (
	"aigentools-backend/internal/models"
	"time"
)

type TransactionListItem struct {
	ID            uint                   `json:"id"`
	CreatedAt     time.Time              `json:"created_at"`
	UserID        uint                   `json:"user_id"`
	Amount        float64                `json:"amount"`
	BalanceBefore float64                `json:"balance_before"`
	BalanceAfter  float64                `json:"balance_after"`
	Reason        string                 `json:"reason"`
	Operator      string                 `json:"operator"`
	Type          models.TransactionType `json:"type"`
	IPAddress     string                 `json:"ip_address"`
	DeviceInfo    string                 `json:"device_info"`
	Hash          string                 `json:"hash"`
}

type TransactionListResponse struct {
	Transactions []TransactionListItem `json:"transactions"`
	Total        int64                 `json:"total"`
	Page         int                   `json:"page"`
	Limit        int                   `json:"limit"`
}
