package models

import (
	"time"

	"gorm.io/datatypes"
)

// TaskStatus defines the status of a task
type TaskStatus int

const (
	TaskStatusPendingAudit     TaskStatus = 1
	TaskStatusPendingExecution TaskStatus = 2
	TaskStatusProcessing       TaskStatus = 3
	TaskStatusCompleted        TaskStatus = 4
	TaskStatusFailed           TaskStatus = 5
)

// Task represents a task in the system
type Task struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    *time.Time     `gorm:"index" json:"deleted_at,omitempty"`
	InputData    datatypes.JSON `gorm:"type:jsonb" json:"input_data" swaggertype:"object"`
	CreatorID    uint           `json:"creator_id"`
	CreatorName  string         `json:"creator_name"`
	Status       TaskStatus     `json:"status"`
	ResultURL    string         `json:"result_url"`
	RetryCount   int            `json:"retry_count" gorm:"default:0"`
	MaxRetries   int            `json:"max_retries" gorm:"default:3"`
	ErrorLog     string         `json:"error_log"`
	RemoteTaskID string         `json:"remote_task_id"`
}

// TableName overrides the table name
func (Task) TableName() string {
	return "tasks"
}
