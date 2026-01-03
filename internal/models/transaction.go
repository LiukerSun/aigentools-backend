package models

import "time"

type Transaction struct {
	ID          uint      `gorm:"primarykey"`
	CreatedAt   time.Time
	UserID      uint      `gorm:"index;not null"`
	Amount      float64   `gorm:"type:decimal(20,8);not null"`
	BalanceBefore float64 `gorm:"type:decimal(20,8);not null"`
	BalanceAfter  float64 `gorm:"type:decimal(20,8);not null"`
	Reason      string    `gorm:"type:text"`
	Operator    string    `gorm:"type:varchar(100)"`
}
