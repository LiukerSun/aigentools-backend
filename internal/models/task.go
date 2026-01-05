package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
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
	gorm.Model
	InputData   datatypes.JSON `gorm:"type:jsonb" json:"input_data"`
	CreatorID   uint           `json:"creator_id"`
	CreatorName string         `json:"creator_name"`
	Status      TaskStatus     `json:"status"`
	ResultURL   string         `json:"result_url"`
	RetryCount  int            `json:"retry_count" gorm:"default:0"`
	MaxRetries  int            `json:"max_retries" gorm:"default:3"`
	ErrorLog    string         `json:"error_log"`
}

// TableName overrides the table name
func (Task) TableName() string {
	return "tasks"
}
