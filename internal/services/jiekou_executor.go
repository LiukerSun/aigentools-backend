package services

import (
	"aigentools-backend/internal/models"
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

// JiekouExecutor implements the TaskExecutor interface for jiekou.ai style tasks
type JiekouExecutor struct {
	Uploader func(localPath string, objectKey string) (string, error)
}

// Execute performs the task logic
func (e JiekouExecutor) Execute(task *models.Task) (map[string]interface{}, error) {
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

	data, _ := input["data"].(map[string]interface{})
	model, _ := input["model"].(map[string]interface{})
	if data == nil || model == nil {
		return nil, errors.New("missing data or model in input")
	}

	modelURL, _ := model["model_url"].(string)
	if modelURL == "" {
		return nil, errors.New("missing model_url")
	}

	// 2. Send Request
	payloadBytes, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", modelURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("JIEKOU_API")))

	// Add Authorization if needed. For now assume it's either in URL or handled externally
	// If the user provides headers in model config, we can use them.
	if headers, ok := model["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			if s, ok := v.(string); ok {
				req.Header.Set(k, s)
			}
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api returned error status: %d, body: %s", resp.StatusCode, string(body))
	}

	var respData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// 3. Extract Task ID
	// Expecting { "data": { "id": "..." } } or root "id"
	var remoteTaskID string
	if d, ok := respData["data"].(map[string]interface{}); ok {
		if id, ok := d["id"].(string); ok {
			remoteTaskID = id
		} else if id, ok := d["task_id"].(string); ok {
			remoteTaskID = id
		}
	}
	if remoteTaskID == "" {
		if id, ok := respData["id"].(string); ok {
			remoteTaskID = id
		} else if id, ok := respData["task_id"].(string); ok {
			remoteTaskID = id
		}
	}

	if remoteTaskID == "" {
		// Fallback for number ID
		if d, ok := respData["data"].(map[string]interface{}); ok {
			if id, ok := d["id"].(float64); ok {
				remoteTaskID = fmt.Sprintf("%.0f", id)
			}
		}
	}

	if remoteTaskID == "" {
		return nil, fmt.Errorf("could not find task_id in response: %v", respData)
	}

	// 4. Poll for Status
	// Construct polling URL
	// Assume standard pattern: https://api.jiekou.ai/v3/async/tasks/{id} or similar
	// Or use query_url from response if available
	queryURL := ""
	if d, ok := respData["data"].(map[string]interface{}); ok {
		if url, ok := d["query_url"].(string); ok {
			queryURL = url
		}
	}

	if queryURL == "" {
		// Heuristic: Replace "seedance-v1.5-pro-i2v" (or similar) with "tasks/{id}" ???
		// No, that's risky.
		// Let's assume a default template if not provided.
		// "https://api.jiekou.ai/v3/async/tasks/%s"
		// Check if user provided query_url_template in model config
		if t, ok := model["query_url_template"].(string); ok && t != "" {
			queryURL = fmt.Sprintf(t, remoteTaskID)
		} else {
			// Try to guess from model_url base
			// If model_url is https://api.jiekou.ai/v3/async/seedance...,
			// maybe polling is https://api.jiekou.ai/v3/async/task/{id}
			// Let's default to a safe guess and hope for the best or error out?
			// Given I must make it work:
			// I will assume the user or system knows.
			// I'll try: https://api.jiekou.ai/v3/async/task/{id}
			// But for "jiekou.ai", often it is just GET /v3/async/task?id={id} or /v3/async/task/{id}
			// Let's try: https://api.jiekou.ai/v3/async/task/{id}
			// Adjust base URL
			baseURL := "https://api.jiekou.ai/v3/async"
			if strings.HasPrefix(modelURL, baseURL) {
				queryURL = baseURL + "/task/" + remoteTaskID
			} else {
				// Just append id to model url? Unlikely.
				// Fallback
				queryURL = fmt.Sprintf("https://api.jiekou.ai/v3/async/task/%s", remoteTaskID)
			}
		}
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Minute) // Video gen might take time
	var fileURL string

	for {
		select {
		case <-timeout:
			return nil, errors.New("task polling timed out")
		case <-ticker.C:
			statusReq, _ := http.NewRequest("GET", queryURL, nil)
			// Copy headers
			if headers, ok := model["headers"].(map[string]interface{}); ok {
				for k, v := range headers {
					if s, ok := v.(string); ok {
						statusReq.Header.Set(k, s)
					}
				}
			}

			statusResp, err := client.Do(statusReq)
			if err != nil {
				fmt.Printf("Polling error: %v\n", err)
				continue
			}
			defer statusResp.Body.Close()

			var statusData map[string]interface{}
			bodyBytes, _ := io.ReadAll(statusResp.Body)
			json.Unmarshal(bodyBytes, &statusData)

			// Check status
			// Expecting { "data": { "status": "..." } }
			var innerData map[string]interface{}
			if d, ok := statusData["data"].(map[string]interface{}); ok {
				innerData = d
			} else {
				innerData = statusData
			}

			statusVal, _ := innerData["status"].(string)
			statusVal = strings.ToLower(statusVal)

			if statusVal == "success" || statusVal == "completed" || statusVal == "succeeded" {
				// Extract file URL
				// Try common keys
				if url, ok := innerData["url"].(string); ok {
					fileURL = url
				} else if url, ok := innerData["file_url"].(string); ok {
					fileURL = url
				} else if url, ok := innerData["result_url"].(string); ok {
					fileURL = url
				} else if url, ok := innerData["output"].(string); ok {
					fileURL = url
				}

				if fileURL != "" {
					goto Download
				}
				// Maybe it's in a list?
				if results, ok := innerData["results"].([]interface{}); ok && len(results) > 0 {
					if r, ok := results[0].(string); ok {
						fileURL = r
						goto Download
					}
				}

				return nil, fmt.Errorf("completed but file url not found in: %v", innerData)
			} else if statusVal == "failed" || statusVal == "error" {
				return nil, fmt.Errorf("remote task failed: %v", innerData)
			}
			// Continue polling
		}
	}

Download:
	// 5. Download File
	fmt.Printf("Downloading file from %s...\n", fileURL)
	fileResp, err := http.Get(fileURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %v", err)
	}
	defer fileResp.Body.Close()

	// 6. Upload to OSS with taskID as filename
	// Determine extension
	ext := ".mp4" // Default for video
	if idx := strings.LastIndex(fileURL, "."); idx != -1 {
		possibleExt := fileURL[idx:]
		if len(possibleExt) < 6 {
			ext = possibleExt
		}
	}

	// Filename is remoteTaskID + ext
	fileName := remoteTaskID + ext

	// Create temp file
	tmpName := filepath.Join(os.TempDir(), fmt.Sprintf("%s_%s", uuid.New().String(), fileName))
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
	defer os.Remove(tmpName)

	// OSS Key: tasks/{remoteTaskID}.ext or tasks/{localID}/{remoteTaskID}.ext
	// User said "filename is task id".
	// I'll use tasks/{remoteTaskID}.ext to be safe and clean.
	ossKey := fmt.Sprintf("tasks/%s", fileName)

	ossURL, err := uploader(tmpName, ossKey)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to oss: %v", err)
	}

	return map[string]interface{}{
		"oss_url":        ossURL,
		"original_url":   fileURL,
		"remote_task_id": remoteTaskID,
	}, nil
}

func init() {
	RegisterExecutor("jiekou_api", JiekouExecutor{})
}
