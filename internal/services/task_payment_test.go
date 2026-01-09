package services

import (
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPaymentTestDB() {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	db.Migrator().DropTable(&models.User{}, &models.AIModel{}, &models.Task{}, &models.Transaction{})
	db.AutoMigrate(&models.User{}, &models.AIModel{}, &models.Task{}, &models.Transaction{})

	database.DB = db
}

func setupPaymentTestRedis() *miniredis.Miniredis {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}

	database.RedisClient = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	return mr
}

func TestCreateTask_Payment(t *testing.T) {
	setupPaymentTestDB()
	mr := setupPaymentTestRedis()
	defer mr.Close()

	// Seed Model
	model := models.AIModel{
		Name:   "Test Model",
		Price:  10.0,
		Status: models.AIModelStatusOpen,
	}
	database.DB.Create(&model)

	// Seed User
	user := models.User{
		Username:    "payer",
		Balance:     100.0,
		CreditLimit: 0.0,
		Version:     1,
		IsActive:    true,
	}
	database.DB.Create(&user)

	// Case 1: Sufficient Balance
	inputData := map[string]interface{}{
		"model_id": float64(model.ID), // Simulate JSON number
		"prompt":   "test",
	}

	task, err := CreateTask(inputData, user.ID, user.Username)
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, 10.0, task.Cost)

	// Verify Deduction
	var updatedUser models.User
	database.DB.First(&updatedUser, user.ID)
	assert.Equal(t, 90.0, updatedUser.Balance)

	// Verify Transaction
	var trans models.Transaction
	database.DB.Last(&trans)
	assert.Equal(t, -10.0, trans.Amount)
	assert.Equal(t, user.ID, trans.UserID)

	// Case 2: Insufficient Balance but Sufficient Credit
	// Update user balance to 5
	database.DB.Model(&updatedUser).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"balance":      5.0,
		"credit_limit": 50.0,
		"version":      updatedUser.Version + 1,
	})
	// Refresh user
	database.DB.First(&updatedUser, user.ID)

	task2, err := CreateTask(inputData, user.ID, user.Username)
	assert.NoError(t, err)
	assert.NotNil(t, task2)

	database.DB.First(&updatedUser, user.ID)
	assert.Equal(t, -5.0, updatedUser.Balance) // 5 - 10 = -5

	// Case 3: Insufficient Funds
	// Update user balance to -45 (Credit used up mostly), Limit 50. Available = -45 + 50 = 5. Price 10.
	database.DB.Model(&updatedUser).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"balance": -45.0,
		"version": updatedUser.Version + 1,
	})

	task3, err := CreateTask(inputData, user.ID, user.Username)
	assert.Error(t, err)
	assert.Nil(t, task3)
	// ErrInsufficientBalance is not exported or we need to check string
	assert.Contains(t, err.Error(), "insufficient balance")

	database.DB.First(&updatedUser, user.ID)
	assert.Equal(t, -45.0, updatedUser.Balance) // Unchanged

	// Case 4: Missing Model ID
	inputDataMissingID := map[string]interface{}{
		"prompt": "test",
	}
	task4, err := CreateTask(inputDataMissingID, user.ID, user.Username)
	assert.Error(t, err)
	assert.Nil(t, task4)
	assert.Contains(t, err.Error(), "model_id is required")

	// Case 5: Missing Model ID but Valid Model URL
	// Set valid URL for the model
	model.URL = "http://example.com/model"
	database.DB.Save(&model)

	// Reset user balance to positive
	database.DB.Model(&updatedUser).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"balance": 100.0,
		"version": updatedUser.Version + 1,
	})

	inputDataURL := map[string]interface{}{
		"prompt": "test",
		"model": map[string]interface{}{
			"model_url": "http://example.com/model",
		},
	}
	task5, err := CreateTask(inputDataURL, user.ID, user.Username)
	assert.NoError(t, err)
	assert.NotNil(t, task5)
	assert.Equal(t, 10.0, task5.Cost)

	// Verify Deduction
	database.DB.First(&updatedUser, user.ID)
	assert.Equal(t, 90.0, updatedUser.Balance)
	assert.Equal(t, 30.0, updatedUser.TotalConsumed)
}

func TestTask_FailureRefund(t *testing.T) {
	setupPaymentTestDB()
	mr := setupPaymentTestRedis()
	defer mr.Close()

	// Seed Model
	model := models.AIModel{
		Name:   "Refund Test Model",
		Price:  10.0,
		Status: models.AIModelStatusOpen,
	}
	database.DB.Create(&model)

	// Seed User
	user := models.User{
		Username: "refund_user",
		Balance:  100.0,
		Version:  1,
		IsActive: true,
	}
	database.DB.Create(&user)

	// 1. Create Task
	inputData := map[string]interface{}{
		"model_id": float64(model.ID),
		"prompt":   "test",
	}

	task, err := CreateTask(inputData, user.ID, user.Username)
	assert.NoError(t, err)

	// 2. Verify Deduction
	var updatedUser models.User
	database.DB.First(&updatedUser, user.ID)
	assert.Equal(t, 90.0, updatedUser.Balance)

	// 3. Simulate Failure
	// We need to import errors to use errors.New
	// But errors is standard lib.
	// Since we are in same package, we can call handleFailure.
	// But we need to make sure we set max retries.
	task.RetryCount = task.MaxRetries
	handleFailure(task, errors.New("simulated fatal error"))

	// 4. Verify Refund
	database.DB.First(&updatedUser, user.ID)
	assert.Equal(t, 100.0, updatedUser.Balance)

	// Verify Refund Transaction
	var trans models.Transaction
	database.DB.Where("type = ?", models.TransactionTypeUserRefund).Last(&trans)
	assert.Equal(t, 10.0, trans.Amount)
	assert.Equal(t, user.ID, trans.UserID)
	assert.Contains(t, trans.Reason, "Refund")
}
