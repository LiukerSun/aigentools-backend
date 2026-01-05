package services

import (
	"aigentools-backend/internal/models"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestJiekouExecutor_Execute(t *testing.T) {
	// 1. Mock Jiekou API
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/create":
			// Verify creation request
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "123", body["prompt"])
			
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 200,
				"data": map[string]string{
					"id": "jk-task-888",
				},
			})

		case "/query/jk-task-888":
			// Verify polling
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 200,
				"data": map[string]interface{}{
					"status": "success",
					"url": "http://" + r.Host + "/result.mp4",
				},
			})

		case "/result.mp4":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("fake video content"))

		default:
			t.Logf("Unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// 2. Prepare Task Input
	input := map[string]interface{}{
		"data": map[string]interface{}{
			"prompt": "123",
			"image": "https://example.com/img.png",
		},
		"model": map[string]interface{}{
			"model_url": mockServer.URL + "/create",
			"model_name": "Test Model",
			// Override polling template for test since we don't have real jiekou.ai
			"query_url_template": mockServer.URL + "/query/%s",
		},
	}
	inputBytes, _ := json.Marshal(input)

	task := &models.Task{
		Model: gorm.Model{ID: 100},
		InputData: datatypes.JSON(inputBytes),
	}

	// 3. Mock Uploader
	mockUploader := func(localPath string, objectKey string) (string, error) {
		// Verify objectKey matches "tasks/jk-task-888.mp4"
		assert.Equal(t, "tasks/jk-task-888.mp4", objectKey)
		return "https://oss.aliyun.com/" + objectKey, nil
	}

	executor := JiekouExecutor{
		Uploader: mockUploader,
	}

	// 4. Execute
	result, err := executor.Execute(task)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "jk-task-888", result["remote_task_id"])
	assert.Equal(t, "https://oss.aliyun.com/tasks/jk-task-888.mp4", result["oss_url"])
}
