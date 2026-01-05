package services

import (
	"aigentools-backend/config"
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
)

const TaskQueueKey = "task_queue"

// CreateTask creates a new task and optionally pushes it to the queue
func CreateTask(inputData map[string]interface{}, creatorID uint, creatorName string) (*models.Task, error) {
	cfg, _ := config.LoadConfig()

	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		return nil, err
	}

	task := models.Task{
		InputData:   datatypes.JSON(inputJSON),
		CreatorID:   creatorID,
		CreatorName: creatorName,
		Status:      models.TaskStatusPendingAudit,
		MaxRetries:  3,
	}

	if cfg.AutoAudit {
		task.Status = models.TaskStatusPendingExecution
	}

	if err := database.DB.Create(&task).Error; err != nil {
		return nil, err
	}

	if cfg.AutoAudit {
		// Push to Redis
		err := database.RedisClient.RPush(database.Ctx, TaskQueueKey, task.ID).Err()
		if err != nil {
			// If redis fails, we might want to rollback or mark task as failed?
			// For now, just return error, but task is created.
			// Ideally we should transactionally do this or have a sweeper.
			// But for this simple implementation:
			return &task, fmt.Errorf("task created but failed to push to redis: %v", err)
		}
	}

	return &task, nil
}

// ApproveTask approves a task and pushes it to the queue
func ApproveTask(id uint) (*models.Task, error) {
	var task models.Task
	if err := database.DB.First(&task, id).Error; err != nil {
		return nil, err
	}

	if task.Status != models.TaskStatusPendingAudit {
		return nil, errors.New("task is not pending audit")
	}

	task.Status = models.TaskStatusPendingExecution
	if err := database.DB.Save(&task).Error; err != nil {
		return nil, err
	}

	if err := database.RedisClient.RPush(database.Ctx, TaskQueueKey, task.ID).Err(); err != nil {
		return &task, fmt.Errorf("task approved but failed to push to redis: %v", err)
	}

	return &task, nil
}

// UpdateTask updates the input data of a task
func UpdateTask(id uint, inputData map[string]interface{}) (*models.Task, error) {
	var task models.Task
	if err := database.DB.First(&task, id).Error; err != nil {
		return nil, err
	}

	if task.Status >= models.TaskStatusProcessing {
		return nil, errors.New("cannot update task in processing or later state")
	}

	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		return nil, err
	}

	task.InputData = datatypes.JSON(inputJSON)
	if err := database.DB.Save(&task).Error; err != nil {
		return nil, err
	}

	return &task, nil
}

// GetTasks retrieves tasks with pagination and filtering
func GetTasks(page, pageSize int, creatorID uint, status *models.TaskStatus) ([]models.Task, int64, error) {
	var tasks []models.Task
	var total int64

	db := database.DB.Model(&models.Task{})

	if creatorID != 0 {
		db = db.Where("creator_id = ?", creatorID)
	}

	if status != nil {
		db = db.Where("status = ?", *status)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Order("created_at desc").Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// GetTaskByID retrieves a single task by ID
func GetTaskByID(id uint) (*models.Task, error) {
	var task models.Task
	if err := database.DB.First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// StartWorker starts the background worker
func StartWorker() {
	fmt.Println("Worker started...")
	for {
		// BLPop blocks until an item is available
		result, err := database.RedisClient.BLPop(context.Background(), 0*time.Second, TaskQueueKey).Result()
		if err != nil {
			fmt.Printf("Redis BLPop error: %v\n", err)
			time.Sleep(1 * time.Second) // Prevent tight loop on error
			continue
		}

		// result[0] is the key, result[1] is the value
		taskIDStr := result[1]
		taskID, err := strconv.Atoi(taskIDStr)
		if err != nil {
			fmt.Printf("Invalid task ID: %s\n", taskIDStr)
			continue
		}

		go processTask(uint(taskID))
	}
}

func processTask(taskID uint) {
	var task models.Task
	if err := database.DB.First(&task, taskID).Error; err != nil {
		fmt.Printf("Task %d not found: %v\n", taskID, err)
		return
	}

	// Update status to Processing
	task.Status = models.TaskStatusProcessing
	database.DB.Save(&task)

	fmt.Printf("Processing task %d...\n", taskID)

	result, err := executeTaskLogic(&task)

	if err != nil {
		fmt.Printf("Task %d failed: %v\n", taskID, err)
		handleFailure(&task, err)
	} else {
		if hookErr := runAfterExecutionHooks(&task, result); hookErr != nil {
			fmt.Printf("Task %d after hooks error: %v\n", taskID, hookErr)
		}
		fmt.Printf("Task %d completed\n", taskID)
		task.Status = models.TaskStatusCompleted

		// Use OSS URL if available, otherwise fallback to simulated
		if ossURL, ok := result["oss_url"].(string); ok && ossURL != "" {
			task.ResultURL = ossURL
		} else if resultURL, ok := result["result_url"].(string); ok && resultURL != "" {
			task.ResultURL = resultURL
		} else {
			task.ResultURL = fmt.Sprintf("http://oss.example.com/result/%d", taskID)
		}

		database.DB.Save(&task)
	}
}

func executeTaskLogic(task *models.Task) (map[string]interface{}, error) {
	time.Sleep(5 * time.Second)

	// Simulate failure if input contains "fail"
	var input map[string]interface{}
	json.Unmarshal(task.InputData, &input)

	if name, ok := input["executor"].(string); ok && name != "" {
		if ex := getExecutor(name); ex != nil {
			return ex.Execute(task)
		}
	}

	// Auto-detect Jiekou/Model task structure
	if _, ok := input["model"]; ok {
		if ex := getExecutor("jiekou_api"); ex != nil {
			return ex.Execute(task)
		}
	}

	if prompt, ok := input["prompt"].(string); ok && strings.Contains(prompt, "fail") {
		return nil, errors.New("simulated failure")
	}

	return map[string]interface{}{"status": "ok"}, nil
}

func handleFailure(task *models.Task, err error) {
	task.ErrorLog = err.Error()

	if task.RetryCount < task.MaxRetries {
		task.RetryCount++
		task.Status = models.TaskStatusPendingExecution
		fmt.Printf("Retrying task %d (attempt %d/%d)...\n", task.ID, task.RetryCount, task.MaxRetries)

		database.DB.Save(task)

		// Push back to Redis
		database.RedisClient.RPush(database.Ctx, TaskQueueKey, task.ID)
	} else {
		task.Status = models.TaskStatusFailed
		fmt.Printf("Task %d failed permanently after %d retries\n", task.ID, task.MaxRetries)
		database.DB.Save(task)
	}
}
