package models

import "time"

type User struct {
	ID            uint `gorm:"primarykey"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Username      string     `gorm:"uniqueIndex;not null"`
	Password      string     `gorm:"not null"`
	Role          string     `gorm:"not null;default:'user'"`
	Version       int        `gorm:"default:1"`
	IsActive      bool       `gorm:"default:true"`
	ActivatedAt   *time.Time `json:"activated_at,omitempty"`
	DeactivatedAt *time.Time `json:"deactivated_at,omitempty"`
	Balance       float64    `gorm:"default:0;type:decimal(20,8)"`
	CreditLimit   float64    `gorm:"default:0;type:decimal(20,8)"`
	TotalConsumed float64    `gorm:"default:0;type:decimal(20,8)"`
}
