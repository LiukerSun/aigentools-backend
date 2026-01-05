package task

import "aigentools-backend/internal/models"

type CreateTaskRequest struct {
	Body map[string]interface{} `json:"body" binding:"required"`
	User struct {
		CreatorID   uint   `json:"creatorId" binding:"required"`
		CreatorName string `json:"creatorName" binding:"required"`
	} `json:"user" binding:"required"`
}

type UpdateTaskRequest struct {
	Body map[string]interface{} `json:"body" binding:"required"`
}

type TaskListResponse struct {
	Total int64         `json:"total"`
	Items []models.Task `json:"items"`
}
