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

	// 1. Check Model and Price
	var price float64
	var modelID uint
	var modelName string

	// Helper to extract ID
	extractID := func(key string) uint {
		if val, ok := inputData[key]; ok {
			switch v := val.(type) {
			case float64:
				return uint(v)
			case int:
				return uint(v)
			case string:
				if id, err := strconv.Atoi(v); err == nil {
					return uint(id)
				}
			}
		}
		return 0
	}

	modelID = extractID("model_id")
	if modelID == 0 {
		modelID = extractID("modelId")
	}

	// Try to find by model_url if model_id is missing
	if modelID == 0 {
		if modelData, ok := inputData["model"].(map[string]interface{}); ok {
			if url, ok := modelData["model_url"].(string); ok && url != "" {
				var am models.AIModel
				if err := database.DB.Where("url = ?", url).First(&am).Error; err == nil {
					modelID = am.ID
				}
			}
		}
	}

	if modelID == 0 {
		return nil, errors.New("model_id is required")
	}

	// 2. Start Transaction
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if modelID > 0 {
		// We need to call GetAIModelByID but inside the transaction to be safe?
		// Actually GetAIModelByID uses database.DB. We can use it, but if we want to be part of transaction
		// we should probably query using tx. But GetAIModelByID is a read operation, it's fine to be outside or independent.
		// However, to ensure consistency (e.g. price doesn't change), maybe we should query inside.
		// But for now, using the service function is fine.
		model, err := GetAIModelByID(modelID)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("invalid model_id: %v", err)
		}
		price = model.Price
		modelName = model.Name
	}

	// 3. Deduct Balance
	if price > 0 {
		_, err := DeductBalanceTx(tx, creatorID, price, fmt.Sprintf("Create task for model: %s", modelName), TransactionMetadata{
			Operator:   "system",
			OperatorID: 0,
			Type:       models.TransactionTypeUserConsume,
		})
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	task := models.Task{
		InputData:   datatypes.JSON(inputJSON),
		CreatorID:   creatorID,
		CreatorName: creatorName,
		Status:      models.TaskStatusPendingAudit,
		MaxRetries:  3,
		Cost:        price,
	}

	if cfg.AutoAudit {
		task.Status = models.TaskStatusPendingExecution
	}

	if err := tx.Create(&task).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Invalidate user cache to ensure balance is updated
	if database.RedisClient != nil {
		cacheKey := fmt.Sprintf("user:%d", creatorID)
		database.RedisClient.Del(database.Ctx, cacheKey)
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

// RetryTask retries a failed or completed task
func RetryTask(id uint, userID uint) (*models.Task, error) {
	var task models.Task
	if err := database.DB.First(&task, id).Error; err != nil {
		return nil, err
	}

	// Verify permission (assuming userID matches creator)
	if task.CreatorID != userID {
		// In a real app we might check admin role too, but for now strict ownership
		return nil, errors.New("unauthorized to retry this task")
	}

	// Check if retryable (Failed or Completed - user wants to re-run?)
	// Requirement says "check if retryable (like failed/timeout)"
	// Assuming Completed tasks can also be re-run if user desires, or restrict to Failed?
	// Let's restrict to Failed or maybe PendingExecution if it got stuck?
	// Common pattern: Allow retry on Failed.
	// User input: "检查任务当前状态是否可重试（如失败/超时状态）"
	if task.Status != models.TaskStatusFailed {
		return nil, errors.New("task is not in a failed state")
	}

	// Reset status and counters
	task.Status = models.TaskStatusPendingExecution
	task.RetryCount = 0
	task.ErrorLog = "" // Clear previous error
	task.ResultURL = ""

	if err := database.DB.Save(&task).Error; err != nil {
		return nil, err
	}

	// Push back to Redis
	if err := database.RedisClient.RPush(database.Ctx, TaskQueueKey, task.ID).Err(); err != nil {
		return &task, fmt.Errorf("task reset but failed to push to redis: %v", err)
	}

	return &task, nil
}

// CancelTask cancels a task
func CancelTask(id uint, userID uint) (*models.Task, error) {
	var task models.Task
	if err := database.DB.First(&task, id).Error; err != nil {
		return nil, err
	}

	// Verify permission
	if task.CreatorID != userID {
		return nil, errors.New("unauthorized to cancel this task")
	}

	// Check if cancellable
	if task.Status == models.TaskStatusCompleted || task.Status == models.TaskStatusFailed || task.Status == models.TaskStatusCancelled {
		return nil, errors.New("task cannot be cancelled in its current state")
	}

	task.Status = models.TaskStatusCancelled
	if err := database.DB.Save(&task).Error; err != nil {
		return nil, err
	}

	return &task, nil
}

// ResumeProcessingTasks finds tasks stuck in processing state and re-queues them or polls their status
func ResumeProcessingTasks() {
	var processingTasks []models.Task
	// Find tasks that are 'Processing' but not completed.
	// We might want to filter by updated_at to avoid picking up just-started tasks,
	// but for startup recovery, we should check all.
	if err := database.DB.Where("status = ?", models.TaskStatusProcessing).Find(&processingTasks).Error; err != nil {
		fmt.Printf("Failed to fetch processing tasks: %v\n", err)
		return
	}

	fmt.Printf("Found %d tasks in Processing state. Attempting to resume...\n", len(processingTasks))

	for _, task := range processingTasks {
		// If task has RemoteTaskID, we can add it to PollingManager (to be implemented)
		// Or push back to Redis for Worker to pick up.
		// If we push back to Redis, the worker will call executeTaskLogic again.
		// executeTaskLogic usually starts a new execution (submits new request).
		// But for Jiekou/Remote tasks, we might want to check status instead of re-submitting if we have RemoteTaskID.
		// Currently executeTaskLogic doesn't support "Resume" mode directly, it re-executes.

		// Strategy:
		// 1. If RemoteTaskID exists, it means we submitted successfully. We should poll.
		// 2. If no RemoteTaskID, maybe it crashed before submission or during submission. Safe to re-try (re-queue).

		if task.RemoteTaskID != "" {
			fmt.Printf("Task %d has RemoteTaskID %s. Adding to polling queue...\n", task.ID, task.RemoteTaskID)
			// TODO: Add to PollingManager
			// For now, we can just push to Redis, but we need executeTaskLogic to handle "already submitted" case?
			// Or we create a specific "PollTask" function?
			// Let's implement PollingManager as requested.
			PollingMgr.Add(task.ID)
		} else {
			fmt.Printf("Task %d has no RemoteTaskID. Re-queuing for execution...\n", task.ID)
			// Reset status to PendingExecution to be picked up by worker normally
			task.Status = models.TaskStatusPendingExecution
			database.DB.Save(&task)
			database.RedisClient.RPush(database.Ctx, TaskQueueKey, task.ID)
		}
	}
}

// StartWorker starts the background worker
func StartWorker() {
	// Start Polling Manager
	go PollingMgr.Start()

	// Resume tasks
	go ResumeProcessingTasks()

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

		// Refund if cost > 0
		if task.Cost > 0 {
			_, refundErr := AdjustBalance(task.CreatorID, task.Cost, fmt.Sprintf("Refund for task %d failure", task.ID), TransactionMetadata{
				Operator: "system",
				Type:     models.TransactionTypeUserRefund,
			})
			if refundErr != nil {
				fmt.Printf("Refund failed for task %d: %v\n", task.ID, refundErr)
				task.ErrorLog += fmt.Sprintf("; Refund failed: %v", refundErr)
			}
		}

		database.DB.Save(task)
	}
}
