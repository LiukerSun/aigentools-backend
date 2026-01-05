package services

import (
	"aigentools-backend/internal/models"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

func TestAfterExecutionHookIsCalled(t *testing.T) {
	ch := make(chan uint, 1)
	RegisterAfterExecutionHook(func(task *models.Task, result map[string]interface{}) error {
		ch <- task.ID
		return nil
	})
	input := map[string]interface{}{
		"prompt": "hello",
	}
	raw, _ := json.Marshal(input)
	task := &models.Task{
		InputData: datatypes.JSON(raw),
	}
	result, err := executeTaskLogic(task)
	assert.NoError(t, err)
	err = runAfterExecutionHooks(task, result)
	assert.NoError(t, err)
	select {
	case gotID := <-ch:
		assert.Equal(t, task.ID, gotID)
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for after execution hook")
	}
}
