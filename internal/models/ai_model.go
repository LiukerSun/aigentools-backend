package models

import "time"

type AIModelStatus string

const (
	AIModelStatusOpen   AIModelStatus = "open"
	AIModelStatusClosed AIModelStatus = "closed"
	AIModelStatusDraft  AIModelStatus = "draft"
)

type AIModel struct {
	ID          uint          `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Name        string        `gorm:"index;not null" json:"name"`
	Description string        `json:"description"`
	Status      AIModelStatus `gorm:"index;not null;default:'draft'" json:"status"`
	Parameters  JSON          `gorm:"type:jsonb;not null;default:'{}'" json:"parameters"`
}
