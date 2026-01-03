package models

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type TransactionType string

const (
	TransactionTypeSystemAdmin TransactionType = "admin_adjustment"
	TransactionTypeSystemAuto  TransactionType = "system_auto"
	TransactionTypeUserConsume TransactionType = "user_consume"
	TransactionTypeUserRefund  TransactionType = "user_refund"
)

type Transaction struct {
	ID            uint            `gorm:"primarykey"`
	CreatedAt     time.Time       `gorm:"precision:3"` // Millisecond precision
	UserID        uint            `gorm:"index;not null"`
	Amount        float64         `gorm:"type:decimal(20,8);not null"`
	BalanceBefore float64         `gorm:"type:decimal(20,8);not null"`
	BalanceAfter  float64         `gorm:"type:decimal(20,8);not null"`
	Reason        string          `gorm:"type:text"`
	Operator      string          `gorm:"type:varchar(100)"` // Username or 'system'
	OperatorID    uint            `gorm:"index;default:0"`   // 0 for system, otherwise UserID
	Type          TransactionType `gorm:"type:varchar(50);index;default:'system_auto'"`
	IPAddress     string          `gorm:"type:varchar(50)"`
	DeviceInfo    string          `gorm:"type:varchar(255)"`
	Hash          string          `gorm:"type:varchar(64);default:''"` // HMAC SHA256
}

// GenerateHash generates a tamper-proof hash for the transaction
func (t *Transaction) GenerateHash(secret string) string {
	data := fmt.Sprintf("%d|%d|%.8f|%.8f|%.8f|%s|%s|%s|%d",
		t.UserID, t.CreatedAt.UnixNano(), t.Amount, t.BalanceBefore, t.BalanceAfter,
		t.Reason, t.Operator, t.Type, t.OperatorID)

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
