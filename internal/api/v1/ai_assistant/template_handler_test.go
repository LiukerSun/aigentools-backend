package ai_assistant_test

import (
	"aigentools-backend/internal/api/v1/ai_assistant"
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCreateTemplate(t *testing.T) {
	setupTestDB()
	mr := setupTestRedis()
	defer mr.Close()
	gin.SetMode(gin.TestMode)

	user := models.User{ID: 1, Role: "user"}

	reqBody := ai_assistant.CreateTemplateRequest{
		Name:     "My Template",
		Content:  "Hello {{name}}",
		IsPublic: false,
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/ai-assistant/templates", bytes.NewBuffer(body))
	c.Set("user", user)

	ai_assistant.CreateTemplate(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data models.PromptTemplate `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "My Template", resp.Data.Name)
	assert.Equal(t, uint(1), resp.Data.UserID)
	assert.Equal(t, models.PromptTemplateTypePrivate, resp.Data.Type)
}

func TestCreatePublicTemplate_Forbidden(t *testing.T) {
	setupTestDB()
	mr := setupTestRedis()
	defer mr.Close()
	gin.SetMode(gin.TestMode)

	user := models.User{ID: 1, Role: "user"}

	reqBody := ai_assistant.CreateTemplateRequest{
		Name:     "Public Template",
		Content:  "Content",
		IsPublic: true,
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/ai-assistant/templates", bytes.NewBuffer(body))
	c.Set("user", user)

	ai_assistant.CreateTemplate(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCreatePublicTemplate_Admin_Success(t *testing.T) {
	setupTestDB()
	mr := setupTestRedis()
	defer mr.Close()
	gin.SetMode(gin.TestMode)

	admin := models.User{ID: 99, Role: "admin"}

	reqBody := ai_assistant.CreateTemplateRequest{
		Name:     "Public Template",
		Content:  "Content",
		IsPublic: true,
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/ai-assistant/templates", bytes.NewBuffer(body))
	c.Set("user", admin)

	ai_assistant.CreateTemplate(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data models.PromptTemplate `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, models.PromptTemplateTypePublic, resp.Data.Type)
}

func TestListTemplates(t *testing.T) {
	setupTestDB()
	mr := setupTestRedis()
	defer mr.Close()
	gin.SetMode(gin.TestMode)

	// Create some templates
	// Public (UserID 0 or Admin)
	db := database.DB
	db.Create(&models.PromptTemplate{Name: "Public 1", Content: "Content", Type: models.PromptTemplateTypePublic, UserID: 0})

	// Private for User 1
	db.Create(&models.PromptTemplate{Name: "Private 1", Content: "Content", Type: models.PromptTemplateTypePrivate, UserID: 1})

	// Private for User 2
	db.Create(&models.PromptTemplate{Name: "Private 2", Content: "Content", Type: models.PromptTemplateTypePrivate, UserID: 2})

	// Test Listing for User 1
	user1 := models.User{ID: 1, Role: "user"}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/ai-assistant/templates", nil)
	c.Set("user", user1)

	ai_assistant.ListTemplates(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data ai_assistant.TemplateListResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Should see Public 1 and Private 1. Should NOT see Private 2.
	// Total 2
	assert.Equal(t, int64(2), resp.Data.Total)
	names := []string{resp.Data.Items[0].Name, resp.Data.Items[1].Name}
	assert.Contains(t, names, "Public 1")
	assert.Contains(t, names, "Private 1")
	assert.NotContains(t, names, "Private 2")
}

func TestUpdateTemplate(t *testing.T) {
	setupTestDB()
	mr := setupTestRedis()
	defer mr.Close()
	gin.SetMode(gin.TestMode)

	db := database.DB
	db.Create(&models.PromptTemplate{ID: 1, Name: "Original", Content: "Content", Type: models.PromptTemplateTypePrivate, UserID: 1})

	user := models.User{ID: 1, Role: "user"}
	reqBody := ai_assistant.UpdateTemplateRequest{
		Name: "Updated",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("PUT", "/ai-assistant/templates/1", bytes.NewBuffer(body))
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("user", user)

	ai_assistant.UpdateTemplate(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data models.PromptTemplate `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Updated", resp.Data.Name)
}

func TestUpdateTemplate_PublicStatus(t *testing.T) {
	setupTestDB()
	mr := setupTestRedis()
	defer mr.Close()
	gin.SetMode(gin.TestMode)

	// User created template
	db := database.DB
	db.Create(&models.PromptTemplate{ID: 1, Name: "My Private", Content: "Content", Type: models.PromptTemplateTypePrivate, UserID: 1})

	user := models.User{ID: 1, Role: "user"}
	admin := models.User{ID: 99, Role: "admin"}

	// 1. User tries to set public -> Forbidden
	isPublic := true
	reqBody := ai_assistant.UpdateTemplateRequest{
		IsPublic: &isPublic,
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("PUT", "/ai-assistant/templates/1", bytes.NewBuffer(body))
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("user", user)

	ai_assistant.UpdateTemplate(c)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// 2. Admin (who also owns it for test simplicity, or we allow admin to update any?)
	// Update logic currently checks ownership strict.
	// Let's create a template owned by admin for this test case.
	db.Create(&models.PromptTemplate{ID: 2, Name: "Admin Private", Content: "Content", Type: models.PromptTemplateTypePrivate, UserID: 99})

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request, _ = http.NewRequest("PUT", "/ai-assistant/templates/2", bytes.NewBuffer(body))
	c2.Params = gin.Params{{Key: "id", Value: "2"}}
	c2.Set("user", admin)

	ai_assistant.UpdateTemplate(c2)
	assert.Equal(t, http.StatusOK, w2.Code)

	var resp struct {
		Data models.PromptTemplate `json:"data"`
	}
	json.Unmarshal(w2.Body.Bytes(), &resp)
	assert.Equal(t, models.PromptTemplateTypePublic, resp.Data.Type)
}

func TestDeleteTemplate(t *testing.T) {
	setupTestDB()
	mr := setupTestRedis()
	defer mr.Close()
	gin.SetMode(gin.TestMode)

	db := database.DB
	db.Create(&models.PromptTemplate{ID: 1, Name: "To Delete", Type: models.PromptTemplateTypePrivate, UserID: 1})

	user := models.User{ID: 1, Role: "user"}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("DELETE", "/ai-assistant/templates/1", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("user", user)

	ai_assistant.DeleteTemplate(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify deleted
	var count int64
	db.Model(&models.PromptTemplate{}).Count(&count)
	assert.Equal(t, int64(0), count)
}
