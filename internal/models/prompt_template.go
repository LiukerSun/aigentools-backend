package models

import "time"

type PromptTemplateType string

const (
	PromptTemplateTypePublic  PromptTemplateType = "public"
	PromptTemplateTypePrivate PromptTemplateType = "private"
)

// PromptTemplate represents a user or system prompt template
type PromptTemplate struct {
	ID          uint               `gorm:"primarykey" json:"id"`
	Name        string             `gorm:"index;not null" json:"name"`
	Description string             `json:"description"`
	Content     string             `gorm:"type:text;not null" json:"content"`
	Type        PromptTemplateType `gorm:"index;not null;default:'private'" json:"type"`
	UserID      uint               `gorm:"index" json:"user_id"` // 0 for system/public templates
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}
