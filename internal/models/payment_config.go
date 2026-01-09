package models

import (
	"time"

	"gorm.io/datatypes"
)

type PaymentConfig struct {
	ID            uint           `gorm:"primarykey"`
	UUID          string         `gorm:"uniqueIndex;type:varchar(36);not null"`
	Name          string         `gorm:"type:varchar(100);not null;default:'Payment Method'"` // Display name
	PaymentMethod string         `gorm:"type:varchar(50);not null"`                           // e.g., "epay"
	Config        datatypes.JSON `gorm:"type:json;not null"`
	Enable        bool           `gorm:"default:true"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type PaymentOrderRecord struct {
	ID          string  `gorm:"primarykey;type:varchar(32)"` // Order ID
	UserID      uint    `gorm:"index;not null"`
	Amount      float64 `gorm:"type:decimal(20,2);not null"`
	Status      string  `gorm:"type:varchar(20);default:'pending'"` // pending, paid, cancelled
	PaymentUUID string  `gorm:"type:varchar(36);index"`             // Which payment config was used
	ExternalID  string  `gorm:"type:varchar(64);index"`             // Transaction ID from payment gateway
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
