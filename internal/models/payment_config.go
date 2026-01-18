package models

import (
	"time"

	"gorm.io/datatypes"
)

// 订单状态常量
const (
	OrderStatusPending   = "pending"
	OrderStatusPaid      = "paid"
	OrderStatusCancelled = "cancelled"
)

// 订单类型常量
const (
	OrderTypePayment = "payment" // 在线支付
	OrderTypeManual  = "manual"  // 管理员手动创建
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

	// 新增字段
	OrderType   string     `gorm:"type:varchar(20);default:'payment';index"` // payment, manual
	Remark      string     `gorm:"type:varchar(500)"`                        // 订单备注
	CompletedAt *time.Time `gorm:"index"`                                    // 完成时间
	CompletedBy uint       `gorm:"index;default:0"`                          // 完成操作者ID（0表示系统）

	CreatedAt time.Time
	UpdatedAt time.Time
}
