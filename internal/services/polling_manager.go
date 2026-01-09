package services

import (
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// PollingManager handles background polling of tasks
type PollingManager struct {
	mu         sync.RWMutex
	tasks      map[uint]*PollingTask
	processing sync.Map
	addChan    chan uint
	removeChan chan uint
	stopChan   chan struct{}
}

type PollingTask struct {
	ID           uint
	RemoteTaskID string
	RetryCount   int
	LastPoll     time.Time
	Executor     string // "jiekou_api" etc.
}

var PollingMgr *PollingManager

func init() {
	PollingMgr = &PollingManager{
		tasks:      make(map[uint]*PollingTask),
		addChan:    make(chan uint, 100),
		removeChan: make(chan uint, 100),
		stopChan:   make(chan struct{}),
	}
}

// Add adds a task ID to the polling manager
func (pm *PollingManager) Add(taskID uint) {
	pm.addChan <- taskID
}

// Remove removes a task ID from the polling manager
func (pm *PollingManager) Remove(taskID uint) {
	pm.removeChan <- taskID
}

// Start starts the polling loop
func (pm *PollingManager) Start() {
	fmt.Println("PollingManager started...")
	ticker := time.NewTicker(30 * time.Second) // Poll every 30s as requested
	defer ticker.Stop()

	for {
		select {
		case id := <-pm.addChan:
			pm.mu.Lock()
			// Check if already exists
			if _, exists := pm.tasks[id]; !exists {
				// Load task details
				var task models.Task
				if err := database.DB.First(&task, id).Error; err == nil {
					// Identify executor
					executorName := "jiekou_api" // Default or detect
					var input map[string]interface{}
					json.Unmarshal(task.InputData, &input)
					if name, ok := input["executor"].(string); ok && name != "" {
						executorName = name
					}

					pm.tasks[id] = &PollingTask{
						ID:           task.ID,
						RemoteTaskID: task.RemoteTaskID,
						RetryCount:   0,
						LastPoll:     time.Now(),
						Executor:     executorName,
					}
					fmt.Printf("PollingManager: Added task %d\n", id)
				}
			}
			pm.mu.Unlock()

		case id := <-pm.removeChan:
			pm.mu.Lock()
			delete(pm.tasks, id)
			pm.mu.Unlock()
			fmt.Printf("PollingManager: Removed task %d\n", id)

		case <-ticker.C:
			pm.pollAll()

		case <-pm.stopChan:
			return
		}
	}
}

func (pm *PollingManager) pollAll() {
	pm.mu.RLock()
	tasks := make([]*PollingTask, 0, len(pm.tasks))
	for _, t := range pm.tasks {
		tasks = append(tasks, t)
	}
	pm.mu.RUnlock()

	for _, pt := range tasks {
		go pm.pollTask(pt)
	}
}

func (pm *PollingManager) pollTask(pt *PollingTask) {
	fmt.Printf("PollingManager: Polling task %d (Remote: %s)...\n", pt.ID, pt.RemoteTaskID)

	// We need a way to execute the "Poll" logic specifically.
	// Currently executors implement "Execute" which does full submission + polling.
	// We should probably extend TaskExecutor interface or just use JiekouExecutor directly if we know it.
	// For now, let's assume we can reuse JiekouExecutor logic or extract it.
	// Since refactoring interface is bigger, I will instantiate JiekouExecutor and call a new method if possible,
	// or just reuse Execute but logic needs to handle "Already Submitted".

	// Let's modify JiekouExecutor to handle "Poll Only" if RemoteTaskID is present?
	// Or better, creating a dedicated Poll method in services is cleaner but requires code duplication.
	// Let's try to use the registered executor.

	var task models.Task
	if err := database.DB.First(&task, pt.ID).Error; err != nil {
		pm.Remove(pt.ID)
		return
	}

	// Determine executor
	ex := getExecutor(pt.Executor)
	if ex == nil {
		// Fallback
		ex = getExecutor("jiekou_api")
	}

	if ex == nil {
		fmt.Printf("PollingManager: No executor found for task %d\n", pt.ID)
		return
	}

	// We need the executor to support "Resume" or "Poll".
	// If we call Execute(task), standard JiekouExecutor will try to submit again because it doesn't check RemoteTaskID at start.
	// We should modify JiekouExecutor to check task.RemoteTaskID first!

	result, err := ex.Execute(&task)
	if err != nil {
		fmt.Printf("PollingManager: Task %d poll failed: %v\n", pt.ID, err)
		pt.RetryCount++
		if pt.RetryCount > 5 {
			fmt.Printf("PollingManager: Task %d failed too many times, removing.\n", pt.ID)
			// Mark as failed in DB
			task.Status = models.TaskStatusFailed
			task.ErrorLog = fmt.Sprintf("Polling failed after retries: %v", err)

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

			database.DB.Save(&task)
			pm.Remove(pt.ID)
		}
		return
	}

	// Success
	fmt.Printf("PollingManager: Task %d completed.\n", pt.ID)
	// Update DB (Execute usually returns map, but doesn't save final status?
	// Wait, processTask saves status. Executor returns result map.
	// We need to save result here.

	if hookErr := runAfterExecutionHooks(&task, result); hookErr != nil {
		fmt.Printf("PollingManager: Task %d after hooks error: %v\n", pt.ID, hookErr)
	}
	task.Status = models.TaskStatusCompleted
	if ossURL, ok := result["oss_url"].(string); ok && ossURL != "" {
		task.ResultURL = ossURL
	}
	database.DB.Save(&task)
	pm.Remove(pt.ID)
}
