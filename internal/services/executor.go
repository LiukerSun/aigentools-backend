package services

import (
	"aigentools-backend/internal/models"
	"sync"
)

type TaskExecutor interface {
	Execute(*models.Task) (map[string]interface{}, error)
}

var executorMu sync.RWMutex
var executorRegistry = make(map[string]TaskExecutor)

func RegisterExecutor(name string, ex TaskExecutor) {
	executorMu.Lock()
	executorRegistry[name] = ex
	executorMu.Unlock()
}

func getExecutor(name string) TaskExecutor {
	executorMu.RLock()
	ex := executorRegistry[name]
	executorMu.RUnlock()
	return ex
}

type AfterExecutionHook func(*models.Task, map[string]interface{}) error

var hookMu sync.RWMutex
var afterHooks []AfterExecutionHook

func RegisterAfterExecutionHook(h AfterExecutionHook) {
	hookMu.Lock()
	afterHooks = append(afterHooks, h)
	hookMu.Unlock()
}

func runAfterExecutionHooks(task *models.Task, result map[string]interface{}) error {
	hookMu.RLock()
	hooks := make([]AfterExecutionHook, len(afterHooks))
	copy(hooks, afterHooks)
	hookMu.RUnlock()
	for _, h := range hooks {
		if err := h(task, result); err != nil {
			return err
		}
	}
	return nil
}
