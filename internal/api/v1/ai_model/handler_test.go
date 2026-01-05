package ai_model_test

import (
	"aigentools-backend/internal/api/v1/ai_model"
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
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() {
	// Initialize logger for tests
	logger.Log = zap.NewNop()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	db.Migrator().DropTable(&models.User{}, &models.AIModel{})
	err = db.AutoMigrate(&models.User{}, &models.AIModel{})
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

func TestGetModelNames(t *testing.T) {
	setupTestDB()
	gin.SetMode(gin.TestMode)

	modelsList := []models.AIModel{
		{Name: "Model A", Status: models.AIModelStatusOpen},
		{Name: "Model B", Status: models.AIModelStatusOpen},
	}
	database.DB.Create(&modelsList)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/models/names", nil)

	ai_model.GetModelNames(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Status int                          `json:"status"`
		Data   []ai_model.AIModelSimpleItem `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 200, resp.Status)
	assert.Len(t, resp.Data, 2)

	// Verify content
	names := []string{resp.Data[0].Name, resp.Data[1].Name}
	assert.Contains(t, names, "Model A")
	assert.Contains(t, names, "Model B")

	// Verify structure
	assert.NotEmpty(t, resp.Data[0].ID)
	assert.NotEmpty(t, resp.Data[0].Status)
}

func TestGetModelParameters(t *testing.T) {
	setupTestDB()
	mr := setupTestRedis()
	defer mr.Close()
	gin.SetMode(gin.TestMode)

	params := models.JSON{"key": "value"}
	model := models.AIModel{
		Name:       "Param Model",
		Status:     models.AIModelStatusOpen,
		Parameters: params,
	}
	database.DB.Create(&model)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(model.ID)}}
	c.Request, _ = http.NewRequest("GET", "/models/"+fmt.Sprint(model.ID)+"/parameters", nil)

	ai_model.GetModelParameters(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Status int         `json:"status"`
		Data   models.JSON `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 200, resp.Status)
	assert.Equal(t, "value", resp.Data["key"])
}

func TestGetModels(t *testing.T) {
	setupTestDB()
	gin.SetMode(gin.TestMode)

	// Seed data
	adminUser := models.User{Username: "admin", Role: "admin"}
	normalUser := models.User{Username: "user", Role: "user"}
	database.DB.Create(&adminUser)
	database.DB.Create(&normalUser)

	modelsList := []models.AIModel{
		{Name: "Model 1", Status: models.AIModelStatusOpen},
		{Name: "Model 2", Status: models.AIModelStatusClosed},
		{Name: "Model 3", Status: models.AIModelStatusDraft},
		{Name: "Model 4", Status: models.AIModelStatusOpen},
	}
	database.DB.Create(&modelsList)

	tests := []struct {
		name           string
		user           models.User
		queryParams    string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "Admin sees all models",
			user:           adminUser,
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Status int                          `json:"status"`
					Data   ai_model.AIModelListResponse `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.Equal(t, 200, resp.Status)
				assert.Equal(t, int64(4), resp.Data.Total)
			},
		},
		{
			name:           "User sees only open models",
			user:           normalUser,
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Status int                          `json:"status"`
					Data   ai_model.AIModelListResponse `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.Equal(t, 200, resp.Status)
				assert.Equal(t, int64(2), resp.Data.Total)
				for _, m := range resp.Data.Models {
					assert.Equal(t, models.AIModelStatusOpen, m.Status)
				}
			},
		},
		{
			name:           "User requests closed models -> empty",
			user:           normalUser,
			queryParams:    "?status=closed",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Status int                          `json:"status"`
					Data   ai_model.AIModelListResponse `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.Equal(t, 200, resp.Status)
				assert.Equal(t, int64(0), resp.Data.Total)
			},
		},
		{
			name:           "Admin requests closed models -> sees closed",
			user:           adminUser,
			queryParams:    "?status=closed",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Status int                          `json:"status"`
					Data   ai_model.AIModelListResponse `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.Equal(t, 200, resp.Status)
				assert.Equal(t, int64(1), resp.Data.Total)
				assert.Equal(t, models.AIModelStatusClosed, resp.Data.Models[0].Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req, _ := http.NewRequest("GET", "/models"+tt.queryParams, nil)
			c.Request = req

			// Set user in context (mock middleware)
			c.Set("user", tt.user)

			ai_model.GetModels(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResponse(t, w.Body.Bytes())
		})
	}
}

func TestUpdateModelStatus(t *testing.T) {
	setupTestDB()
	gin.SetMode(gin.TestMode)

	adminUser := models.User{Username: "admin", Role: "admin"}
	normalUser := models.User{Username: "user", Role: "user"}
	database.DB.Create(&adminUser)
	database.DB.Create(&normalUser)

	model := models.AIModel{Name: "Model 1", Status: models.AIModelStatusDraft}
	database.DB.Create(&model)

	tests := []struct {
		name           string
		user           models.User
		modelID        string
		reqBody        interface{}
		expectedStatus int
		checkResult    func(t *testing.T)
	}{
		{
			name:           "Admin updates status to open",
			user:           adminUser,
			modelID:        "1",
			reqBody:        ai_model.UpdateStatusRequest{Status: models.AIModelStatusOpen},
			expectedStatus: http.StatusOK,
			checkResult: func(t *testing.T) {
				var updatedModel models.AIModel
				database.DB.First(&updatedModel, model.ID)
				assert.Equal(t, models.AIModelStatusOpen, updatedModel.Status)
			},
		},
		{
			name:           "User cannot update status",
			user:           normalUser,
			modelID:        "1",
			reqBody:        ai_model.UpdateStatusRequest{Status: models.AIModelStatusClosed},
			expectedStatus: http.StatusForbidden,
			checkResult: func(t *testing.T) {
				// Status should remain Open from previous test
				var updatedModel models.AIModel
				database.DB.First(&updatedModel, model.ID)
				assert.Equal(t, models.AIModelStatusOpen, updatedModel.Status)
			},
		},
		{
			name:           "Invalid status",
			user:           adminUser,
			modelID:        "1",
			reqBody:        map[string]string{"status": "invalid"},
			expectedStatus: http.StatusBadRequest,
			checkResult:    func(t *testing.T) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonBytes, _ := json.Marshal(tt.reqBody)
			req, _ := http.NewRequest("PATCH", "/models/"+tt.modelID+"/status", bytes.NewBuffer(jsonBytes))
			c.Request = req
			c.Params = []gin.Param{{Key: "id", Value: tt.modelID}}

			c.Set("user", tt.user)

			ai_model.UpdateModelStatus(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResult(t)
		})
	}
}

func TestCreateModel(t *testing.T) {
	setupTestDB()
	gin.SetMode(gin.TestMode)

	adminUser := models.User{Username: "admin", Role: "admin"}
	normalUser := models.User{Username: "user", Role: "user"}
	database.DB.Create(&adminUser)
	database.DB.Create(&normalUser)

	tests := []struct {
		name           string
		user           models.User
		payload        interface{}
		expectedStatus int
	}{
		{
			name: "Admin creates model",
			user: adminUser,
			payload: ai_model.CreateModelRequest{
				Name:        "New Model",
				Description: "Description",
				Status:      models.AIModelStatusDraft,
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "User cannot create model",
			user: normalUser,
			payload: ai_model.CreateModelRequest{
				Name:   "User Model",
				Status: models.AIModelStatusDraft,
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "Invalid payload",
			user: adminUser,
			payload: map[string]interface{}{
				"description": "No name",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Mock auth
			c.Set("user", tt.user)

			jsonBytes, _ := json.Marshal(tt.payload)
			c.Request, _ = http.NewRequest("POST", "/models/create", bytes.NewBuffer(jsonBytes))

			ai_model.CreateModel(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestCreateModelURL(t *testing.T) {
	setupTestDB()
	gin.SetMode(gin.TestMode)

	adminUser := models.User{Username: "admin", Role: "admin"}
	database.DB.Create(&adminUser)

	t.Run("Admin creates model with URL", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// Mock auth
		c.Set("user", adminUser)

		payload := ai_model.CreateModelRequest{
			Name:        "Model With URL",
			Description: "Description",
			Status:      models.AIModelStatusDraft,
			URL:         "https://api.example.com/v1/chat",
		}

		jsonBytes, _ := json.Marshal(payload)
		c.Request, _ = http.NewRequest("POST", "/models/create", bytes.NewBuffer(jsonBytes))

		ai_model.CreateModel(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp struct {
			Status int                      `json:"status"`
			Msg    string                   `json:"msg"`
			Data   ai_model.AIModelListItem `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "https://api.example.com/v1/chat", resp.Data.URL)
		assert.Equal(t, "Model With URL", resp.Data.Name)
	})
}

func TestUpdateModelURL(t *testing.T) {
	setupTestDB()
	gin.SetMode(gin.TestMode)

	adminUser := models.User{Username: "admin", Role: "admin"}
	database.DB.Create(&adminUser)

	model := models.AIModel{
		Name:   "Original Model",
		Status: models.AIModelStatusDraft,
		URL:    "http://original.com",
		Parameters: models.JSON{
			"request_header":      []interface{}{},
			"request_body":        []interface{}{},
			"response_parameters": []interface{}{},
		},
	}
	database.DB.Create(&model)

	t.Run("Admin updates model URL", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// Mock auth
		c.Set("user", adminUser)
		c.Params = []gin.Param{{Key: "id", Value: "1"}}

		payload := ai_model.UpdateModelRequest{
			URL: "https://updated.com/api",
		}

		jsonBytes, _ := json.Marshal(payload)
		c.Request, _ = http.NewRequest("PUT", "/models/1", bytes.NewBuffer(jsonBytes))

		ai_model.UpdateModel(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Status int                      `json:"status"`
			Msg    string                   `json:"msg"`
			Data   ai_model.AIModelListItem `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "https://updated.com/api", resp.Data.URL)
		assert.Equal(t, "Original Model", resp.Data.Name) // Name should remain unchanged
	})
}
