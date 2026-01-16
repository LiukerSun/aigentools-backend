package ai_assistant_test

import (
	"aigentools-backend/internal/api/v1/ai_assistant"
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"aigentools-backend/pkg/logger"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func setupTestDB() {
	logger.Log = zap.NewNop()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %v", err))
	}

	db.Migrator().DropTable(&models.Prompt{}, &models.PromptTemplate{})
	err = db.AutoMigrate(&models.Prompt{}, &models.PromptTemplate{})
	if err != nil {
		panic("failed to migrate database")
	}

	database.DB = db
}

func setupTestRedis() *miniredis.Miniredis {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}

	database.RedisClient = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	return mr
}

func TestCreatePrompt(t *testing.T) {
	setupTestDB()
	gin.SetMode(gin.TestMode)

	reqBody := ai_assistant.CreatePromptRequest{
		Code:    "test_prompt",
		Content: "This is a test prompt",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/ai-assistant/prompts", bytes.NewBuffer(body))

	ai_assistant.CreatePrompt(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data models.Prompt `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "test_prompt", resp.Data.Code)
	assert.Equal(t, "This is a test prompt", resp.Data.Content)
}

func TestBatchCreatePrompts(t *testing.T) {
	setupTestDB()
	gin.SetMode(gin.TestMode)

	reqBody := ai_assistant.BatchCreatePromptRequest{
		Prompts: []ai_assistant.CreatePromptRequest{
			{Code: "p1", Content: "c1"},
			{Code: "p2", Content: "c2"},
		},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/ai-assistant/prompts/batch", bytes.NewBuffer(body))

	ai_assistant.BatchCreatePrompts(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify DB
	var count int64
	database.DB.Model(&models.Prompt{}).Count(&count)
	assert.Equal(t, int64(2), count)
}

func TestGetPrompt(t *testing.T) {
	setupTestDB()
	mr := setupTestRedis()
	defer mr.Close()
	gin.SetMode(gin.TestMode)

	// Create prompt in DB
	prompt := models.Prompt{
		Code:    "get_prompt",
		Content: "Content for get",
	}
	database.DB.Create(&prompt)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/ai-assistant/prompts/get_prompt", nil)
	c.Params = gin.Params{{Key: "code", Value: "get_prompt"}}

	ai_assistant.GetPrompt(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data models.Prompt `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "get_prompt", resp.Data.Code)
}

func TestUpdatePrompt(t *testing.T) {
	setupTestDB()
	mr := setupTestRedis()
	defer mr.Close()
	gin.SetMode(gin.TestMode)

	prompt := models.Prompt{
		Code:    "update_prompt",
		Content: "Old content",
	}
	database.DB.Create(&prompt)

	reqBody := ai_assistant.UpdatePromptRequest{
		Content: "New content",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("PUT", "/ai-assistant/prompts/update_prompt", bytes.NewBuffer(body))
	c.Params = gin.Params{{Key: "code", Value: "update_prompt"}}

	ai_assistant.UpdatePrompt(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify DB update
	var updated models.Prompt
	database.DB.First(&updated, "code = ?", "update_prompt")
	assert.Equal(t, "New content", updated.Content)
}

func TestDeletePrompt(t *testing.T) {
	setupTestDB()
	mr := setupTestRedis()
	defer mr.Close()
	gin.SetMode(gin.TestMode)

	prompt := models.Prompt{
		Code:    "delete_prompt",
		Content: "To be deleted",
	}
	database.DB.Create(&prompt)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("DELETE", "/ai-assistant/prompts/delete_prompt", nil)
	c.Params = gin.Params{{Key: "code", Value: "delete_prompt"}}

	ai_assistant.DeletePrompt(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify DB delete
	var count int64
	database.DB.Model(&models.Prompt{}).Where("code = ?", "delete_prompt").Count(&count)
	assert.Equal(t, int64(0), count)
}
