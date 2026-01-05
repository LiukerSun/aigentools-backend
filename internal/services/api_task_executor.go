package services

import (
	"aigentools-backend/internal/models"
	"aigentools-backend/internal/utils"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RemoteAPITaskExecutor implements the TaskExecutor interface for remote API tasks
type RemoteAPITaskExecutor struct {
	Uploader func(localPath string, objectKey string) (string, error)
}

// Execute performs the task logic
func (e RemoteAPITaskExecutor) Execute(task *models.Task) (map[string]interface{}, error) {
	// Use default uploader if nil
	uploader := e.Uploader
	if uploader == nil {
		uploader = UploadFile
	}

	// 1. Parse Input Data
	var input map[string]interface{}
	if err := json.Unmarshal(task.InputData, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input data: %v", err)
	}

	targetURL, _ := input["target_url"].(string)
	method, _ := input["method"].(string)
	if method == "" {
		method = "POST"
	}
	headers := map[string]interface{}{}
	headers["Authorization"] = fmt.Sprintf("Bearer %s", os.Getenv("JIEKOU_API"))
	headers["Content-Type"] = "application/json"

	payload, _ := input["payload"].(map[string]interface{})

	if targetURL == "" {
		return nil, errors.New("missing target_url")
	}

	// 2. Send Request
	payloadBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest(method, targetURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		if s, ok := v.(string); ok {
			req.Header.Set(k, s)
		}
	}

	client := utils.NewHTTPClient(30 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("api returned error status: %d", resp.StatusCode)
	}

	var respData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// 3. Extract Task ID
	// Assumes task_id is in the root of the JSON response
	remoteTaskID, _ := respData["task_id"].(string)
	if remoteTaskID == "" {
		// Fallback: maybe it's "id" or user specified a key
		if id, ok := respData["id"].(string); ok {
			remoteTaskID = id
		} else {
			return nil, errors.New("could not find task_id in response")
		}
	}

	// 4. Poll for Status
	queryTemplate, _ := input["query_url_template"].(string)
	if queryTemplate == "" {
		// Default assumption: target_url/{task_id}
		queryTemplate = strings.TrimRight(targetURL, "/") + "/%s"
	}
	// If template contains %s, format it, otherwise append ID
	queryURL := queryTemplate
	if strings.Contains(queryTemplate, "%s") {
		queryURL = fmt.Sprintf(queryTemplate, remoteTaskID)
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(10 * time.Minute) // Safety timeout
	var fileURL string

	for {
		select {
		case <-timeout:
			return nil, errors.New("task polling timed out")
		case <-ticker.C:
			// Check status
			statusReq, _ := http.NewRequest("GET", queryURL, nil)
			for k, v := range headers {
				if s, ok := v.(string); ok {
					statusReq.Header.Set(k, s)
				}
			}

			statusResp, err := client.Do(statusReq)
			if err != nil {
				fmt.Printf("Polling error: %v\n", err)
				continue
			}
			defer statusResp.Body.Close() // Close immediately for loop

			var statusData map[string]interface{}
			// Read body to reuse later if needed, or decode directly
			bodyBytes, _ := io.ReadAll(statusResp.Body)
			json.Unmarshal(bodyBytes, &statusData)

			// Determine status. User might configure status field and success value.
			// Defaults: status field "status", success value "completed" or "success"
			statusField := "status"
			statusVal, _ := statusData[statusField].(string)
			statusVal = strings.ToLower(statusVal)

			if statusVal == "completed" || statusVal == "success" || statusVal == "succeeded" {
				// 5. Extract File URL
				// User can specify result_file_key
				fileKey, _ := input["result_file_key"].(string)
				if fileKey == "" {
					fileKey = "file_url"
				}

				if url, ok := statusData[fileKey].(string); ok {
					fileURL = url
					goto Download
				} else if url, ok := statusData["result_url"].(string); ok {
					fileURL = url
					goto Download
				} else if url, ok := statusData["url"].(string); ok {
					fileURL = url
					goto Download
				}

				return nil, errors.New("completed but file url not found")
			} else if statusVal == "failed" || statusVal == "error" {
				return nil, fmt.Errorf("remote task failed: %v", statusData)
			}
			// Continue polling
		}
	}

Download:
	// 6. Download File
	if fileURL == "" {
		return nil, errors.New("empty file url")
	}

	fmt.Printf("Downloading file from %s...\n", fileURL)
	fileResp, err := http.Get(fileURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %v", err)
	}
	defer fileResp.Body.Close()

	// Create temp file
	tmpName := filepath.Join(os.TempDir(), fmt.Sprintf("task_%d_%s", task.ID, uuid.New().String()))
	// Try to guess extension from URL or Content-Type
	// Simple approach: look at URL
	if idx := strings.LastIndex(fileURL, "."); idx != -1 {
		ext := fileURL[idx:]
		if len(ext) < 10 && !strings.Contains(ext, "/") { // sanity check
			tmpName += ext
		}
	}

	out, err := os.Create(tmpName)
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(out, fileResp.Body)
	out.Close()
	if err != nil {
		os.Remove(tmpName)
		return nil, err
	}
	defer os.Remove(tmpName) // Cleanup after upload

	// 7. Upload to OSS
	ossKey := fmt.Sprintf("tasks/%d/%s", task.ID, filepath.Base(tmpName))
	ossURL, err := uploader(tmpName, ossKey)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to oss: %v", err)
	}

	// 8. Return Result
	return map[string]interface{}{
		"oss_url":        ossURL,
		"original_url":   fileURL,
		"remote_task_id": remoteTaskID,
	}, nil
}

// Register in init
func init() {
	RegisterExecutor("remote_api", RemoteAPITaskExecutor{})
}
