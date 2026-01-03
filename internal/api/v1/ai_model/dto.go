package ai_model

import (
	"aigentools-backend/internal/models"
	"time"
)

type AIModelListItem struct {
	ID          uint                 `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Status      models.AIModelStatus `json:"status"`
	Parameters  models.JSON          `json:"parameters"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

type AIModelListResponse struct {
	Models []AIModelListItem `json:"models"`
	Total  int64             `json:"total"`
	Page   int               `json:"page"`
	Limit  int               `json:"limit"`
}

type UpdateStatusRequest struct {
	Status models.AIModelStatus `json:"status" binding:"required,oneof=open closed draft"`
}

type UpdateModelRequest struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Status      models.AIModelStatus `json:"status" binding:"omitempty,oneof=open closed draft"`
	Parameters  models.JSON          `json:"parameters"`
}

type CreateModelRequest struct {
	Name        string               `json:"name" binding:"required"`
	Description string               `json:"description"`
	Status      models.AIModelStatus `json:"status" binding:"required,oneof=open closed draft"`
	Parameters  models.JSON          `json:"parameters"`
}
