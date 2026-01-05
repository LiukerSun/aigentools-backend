package services

import (
	"aigentools-backend/internal/models"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

func TestRemoteAPITaskExecutor_Execute(t *testing.T) {
	// 1. Mock External API
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/submit":
			// Verify submission
			assert.Equal(t, "POST", r.Method)
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "test_prompt", body["prompt"])

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"task_id": "remote-123",
			})

		case "/status/remote-123":
			// Simulate processing then success
			// Simple toggle based on some condition or just always return success for speed
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"status":   "completed",
				"file_url": "http://" + r.Host + "/file.png",
			})

		case "/file.png":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("fake image content"))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// 2. Prepare Task
	input := map[string]interface{}{
		"target_url":         mockServer.URL + "/submit",
		"query_url_template": mockServer.URL + "/status/%s",
		"payload": map[string]interface{}{
			"prompt": "test_prompt",
		},
	}
	inputBytes, _ := json.Marshal(input)

	task := &models.Task{
		ID:        1,
		InputData: datatypes.JSON(inputBytes),
	}

	// 3. Mock Uploader
	mockUploader := func(localPath string, objectKey string) (string, error) {
		assert.Contains(t, objectKey, "tasks/1/")
		return "https://oss.example.com/" + objectKey, nil
	}

	executor := RemoteAPITaskExecutor{
		Uploader: mockUploader,
	}

	// 4. Execute
	result, err := executor.Execute(task)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "remote-123", result["remote_task_id"])
	assert.Contains(t, result["oss_url"], "https://oss.example.com/tasks/1/")
}
