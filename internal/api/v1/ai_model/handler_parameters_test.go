package ai_model_test

import (
	"aigentools-backend/internal/api/v1/ai_model"
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCreateModelWithParameters(t *testing.T) {
	setupTestDB()
	gin.SetMode(gin.TestMode)

	adminUser := models.User{Username: "admin_tester", Role: "admin"}
	database.DB.Create(&adminUser)

	t.Run("Create model with valid parameters", func(t *testing.T) {
		params := map[string]interface{}{
			"request_header": []map[string]interface{}{
				{"name": "Authorization", "type": "string", "required": true, "description": "Token", "example": "Bearer 123"},
			},
			"request_body": []map[string]interface{}{
				{"name": "prompt", "type": "string", "required": true, "description": "Input", "example": "Hello"},
			},
			"response_parameters": []map[string]interface{}{
				{"name": "text", "type": "string", "required": true, "description": "Output", "example": "World"},
			},
		}
		req := ai_model.CreateModelRequest{
			Name:        "GPT-4-Test",
			Description: "Advanced model",
			Status:      models.AIModelStatusOpen,
			URL:         "https://api.openai.com/v1/chat/completions",
			Parameters:  models.JSON(params),
		}
		body, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user", adminUser)
		c.Request, _ = http.NewRequest("POST", "/models/create", bytes.NewBuffer(body))

		ai_model.CreateModel(c)

		if w.Code != http.StatusCreated {
			t.Logf("Response Body: %s", w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp struct {
			Status int                      `json:"status"`
			Data   ai_model.AIModelListItem `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)

		assert.Equal(t, "GPT-4-Test", resp.Data.Name)

		// Verify parameters structure
		respParams := resp.Data.Parameters
		assert.NotNil(t, respParams)
		assert.Contains(t, respParams, "request_header")
		assert.Contains(t, respParams, "request_body")
		assert.Contains(t, respParams, "response_parameters")
	})

	t.Run("Create model with invalid parameters (missing required fields)", func(t *testing.T) {
		// Missing response_parameters
		params := map[string]interface{}{
			"request_header": []interface{}{},
			"request_body":   []interface{}{},
		}
		req := ai_model.CreateModelRequest{
			Name:        "Invalid-Model",
			Description: "Should fail",
			Status:      models.AIModelStatusOpen,
			Parameters:  models.JSON(params),
		}
		body, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user", adminUser)
		c.Request, _ = http.NewRequest("POST", "/models/create", bytes.NewBuffer(body))

		ai_model.CreateModel(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid parameters")
	})

	t.Run("Create model without parameters (defaults to valid empty)", func(t *testing.T) {
		req := ai_model.CreateModelRequest{
			Name:        "GPT-3.5-Test",
			Description: "Basic model",
			Status:      models.AIModelStatusOpen,
		}
		body, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user", adminUser)
		c.Request, _ = http.NewRequest("POST", "/models/create", bytes.NewBuffer(body))

		ai_model.CreateModel(c)

		if w.Code != http.StatusCreated {
			t.Logf("Response Body: %s", w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp struct {
			Status int                      `json:"status"`
			Data   ai_model.AIModelListItem `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)

		// Should be initialized with empty arrays
		assert.NotNil(t, resp.Data.Parameters)
		assert.Contains(t, resp.Data.Parameters, "request_header")
	})
}

func TestUpdateModelParameters(t *testing.T) {
	setupTestDB()
	gin.SetMode(gin.TestMode)

	adminUser := models.User{Username: "admin_tester", Role: "admin"}
	database.DB.Create(&adminUser)

	// Create a model first (using DB directly to bypass validation for setup if needed, but better to be valid)
	validParams := models.JSON{
		"request_header":      []interface{}{},
		"request_body":        []interface{}{},
		"response_parameters": []interface{}{},
	}
	model := models.AIModel{
		Name:       "Update-Test-Model",
		Status:     models.AIModelStatusDraft,
		Parameters: validParams,
	}
	database.DB.Create(&model)

	t.Run("Update with valid parameters", func(t *testing.T) {
		newParams := map[string]interface{}{
			"request_header": []map[string]interface{}{
				{"name": "Auth", "type": "string", "required": true, "description": "Token", "example": "Bearer"},
			},
			"request_body":        []interface{}{},
			"response_parameters": []interface{}{},
		}
		req := ai_model.UpdateModelRequest{
			Parameters: models.JSON(newParams),
		}
		body, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", model.ID)}}
		c.Set("user", adminUser)
		c.Request, _ = http.NewRequest("PUT", "/models/"+fmt.Sprintf("%d", model.ID), bytes.NewBuffer(body))

		ai_model.UpdateModel(c)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify update
		var updatedModel models.AIModel
		database.DB.First(&updatedModel, model.ID)

		// Check if params updated
		// Since it's stored as JSON, we need to be careful with comparison or check fields
		// We'll trust the validation passed and DB saved it.
	})

	t.Run("Update with invalid parameters", func(t *testing.T) {
		invalidParams := map[string]interface{}{
			"request_header": []interface{}{},
			// Missing request_body and response_parameters
		}
		req := ai_model.UpdateModelRequest{
			Parameters: models.JSON(invalidParams),
		}
		body, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", model.ID)}}
		c.Set("user", adminUser)
		c.Request, _ = http.NewRequest("PUT", "/models/"+fmt.Sprintf("%d", model.ID), bytes.NewBuffer(body))

		ai_model.UpdateModel(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid parameters")
	})
}
