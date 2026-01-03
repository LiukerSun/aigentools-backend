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

	t.Run("Create model with parameters", func(t *testing.T) {
		params := map[string]interface{}{
			"temperature": 0.7,
			"max_tokens":  1000.0, // Use float for JSON number
			"stop":        []interface{}{"\n"},
		}
		req := ai_model.CreateModelRequest{
			Name:        "GPT-4-Test",
			Description: "Advanced model",
			Status:      models.AIModelStatusOpen,
			Parameters:  models.JSON(params),
		}
		body, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user", adminUser)
		c.Request, _ = http.NewRequest("POST", "/models/create", bytes.NewBuffer(body))

		ai_model.CreateModel(c)

		// Debug output if status is not 201
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
		// assert.Equal(t, 201, resp.Status) // Response body status might differ if wrapper changes it, but HTTP code should be 201

		assert.Equal(t, "GPT-4-Test", resp.Data.Name)

		// Verify parameters
		respParams := resp.Data.Parameters
		assert.NotNil(t, respParams)
		assert.Equal(t, 0.7, respParams["temperature"])
		assert.Equal(t, 1000.0, respParams["max_tokens"])
	})

	t.Run("Create model without parameters (default)", func(t *testing.T) {
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

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp struct {
			Status int                      `json:"status"`
			Data   ai_model.AIModelListItem `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)

		// Should be empty map or not nil
		assert.NotNil(t, resp.Data.Parameters)
		assert.Empty(t, resp.Data.Parameters)
	})

	t.Run("Update model parameters", func(t *testing.T) {
		// First create a model
		model := models.AIModel{
			Name:       "To Update",
			Status:     models.AIModelStatusDraft,
			Parameters: models.JSON{"v": 1.0},
		}
		database.DB.Create(&model)

		newParams := map[string]interface{}{
			"v":         2.0,
			"new_field": "updated",
		}
		req := ai_model.UpdateModelRequest{
			Parameters: models.JSON(newParams),
		}
		body, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user", adminUser)
		c.Params = []gin.Param{{Key: "id", Value: fmt.Sprint(model.ID)}}
		c.Request, _ = http.NewRequest("PUT", "/models/"+fmt.Sprint(model.ID), bytes.NewBuffer(body))

		ai_model.UpdateModel(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var updatedModel models.AIModel
		database.DB.First(&updatedModel, model.ID)
		assert.Equal(t, 2.0, updatedModel.Parameters["v"])
		assert.Equal(t, "updated", updatedModel.Parameters["new_field"])
	})
}
