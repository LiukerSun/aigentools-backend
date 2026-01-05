package services

import (
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRetryTestDB() {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.Migrator().DropTable(&models.Task{})
	db.AutoMigrate(&models.Task{})
	database.DB = db
}

func setupRetryTestRedis() *miniredis.Miniredis {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	database.RedisClient = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	return mr
}

func TestRetryTask(t *testing.T) {
	setupRetryTestDB()
	mr := setupRetryTestRedis()
	defer mr.Close()

	// 1. Setup Data
	creatorID := uint(10)
	task := models.Task{
		CreatorID:  creatorID,
		Status:     models.TaskStatusFailed,
		RetryCount: 2,
		MaxRetries: 3,
		ErrorLog:   "Some error",
		ResultURL:  "http://old.url",
	}
	database.DB.Create(&task)

	// 2. Test Unauthorized
	_, err := RetryTask(task.ID, 999) // Wrong user
	assert.Error(t, err)
	assert.Equal(t, "unauthorized to retry this task", err.Error())

	// 3. Test Invalid Status
	task.Status = models.TaskStatusCompleted
	database.DB.Save(&task)
	_, err = RetryTask(task.ID, creatorID)
	assert.Error(t, err)
	assert.Equal(t, "task is not in a failed state", err.Error())

	// 4. Test Success Retry
	task.Status = models.TaskStatusFailed
	database.DB.Save(&task)

	retriedTask, err := RetryTask(task.ID, creatorID)
	assert.NoError(t, err)
	assert.Equal(t, models.TaskStatusPendingExecution, retriedTask.Status)
	assert.Equal(t, 0, retriedTask.RetryCount)
	assert.Empty(t, retriedTask.ErrorLog)
	assert.Empty(t, retriedTask.ResultURL)

	// 5. Verify Redis
	val, err := database.RedisClient.RPop(database.Ctx, TaskQueueKey).Result()
	assert.NoError(t, err)
	assert.Equal(t, "1", val) // Task ID 1
}
