package services

import (
	"aigentools-backend/config"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/google/uuid"
)

// OSSUploader defines the interface for OSS operations
type OSSUploader interface {
	UploadFile(localPath string) (string, error)
}

// STSClientManager handles STS token management and OSS client creation
type STSClientManager struct {
	config *config.Config
}

func NewSTSClientManager() *STSClientManager {
	cfg, _ := config.LoadConfig()
	return &STSClientManager{config: cfg}
}

// UploadWithSTS uploads a file using STS credentials with advanced features:
// 1. Multipart upload for large files (>100MB)
// 2. Safe file naming (UUID + Timestamp)
// 3. STS token usage (mapped from SecurityToken)
// 4. Automatic retry mechanism
func (m *STSClientManager) UploadWithSTS(localPath string) (string, error) {
	// 1. Get STS Token
	stsCreds, err := GetOSSTSToken()
	if err != nil {
		return "", fmt.Errorf("failed to get STS token: %v", err)
	}

	// 2. Initialize OSS Client with STS Token
	// CRITICAL: Map SecurityToken to SecurityToken parameter in oss.New
	client, err := oss.New(
		m.config.OSSEndpoint,
		stsCreds.AccessKeyId,
		stsCreds.AccessKeySecret,
		oss.SecurityToken(stsCreds.SecurityToken),
		oss.Timeout(60, 120), // Connect timeout 60s, Read/Write timeout 120s
	)
	if err != nil {
		return "", fmt.Errorf("failed to create OSS client: %v", err)
	}

	bucket, err := client.Bucket(m.config.OSSBucketName)
	if err != nil {
		return "", fmt.Errorf("failed to get bucket: %v", err)
	}

	// 3. Generate Safe Object Key
	ext := ""
	if idx := strings.LastIndex(localPath, "."); idx != -1 {
		ext = localPath[idx:]
	}
	now := time.Now()
	// Structure: tasks/2024/01/uuid.ext
	objectKey := fmt.Sprintf("tasks/%d/%02d/%s%s", 
		now.Year(), now.Month(), uuid.New().String(), ext)

	// 4. Determine Upload Strategy (Multipart vs Simple)
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %v", err)
	}

	const MultipartThreshold = 100 * 1024 * 1024 // 100MB

	var uploadErr error
	if fileInfo.Size() > MultipartThreshold {
		// Large File: Multipart Upload
		// Parallel: 3 routines, PartSize: 1MB
		uploadErr = bucket.UploadFile(objectKey, localPath, 1024*1024, oss.Routines(3), oss.Checkpoint(true, ""))
	} else {
		// Small File: Simple Put
		uploadErr = bucket.PutObjectFromFile(objectKey, localPath)
	}

	// 5. Retry Logic (Simple wrapper, SDK usually has internal retry for some errors)
	if uploadErr != nil {
		// Check for token expiry or network error, simple retry once
		// In a real robust system, we would check error type and maybe refresh token
		fmt.Printf("Upload failed, retrying once... Error: %v\n", uploadErr)
		// Refresh token
		stsCreds, err = GetOSSTSToken()
		if err == nil {
			// Re-create client
			client, _ = oss.New(m.config.OSSEndpoint, stsCreds.AccessKeyId, stsCreds.AccessKeySecret, oss.SecurityToken(stsCreds.SecurityToken))
			bucket, _ = client.Bucket(m.config.OSSBucketName)
			if fileInfo.Size() > MultipartThreshold {
				uploadErr = bucket.UploadFile(objectKey, localPath, 1024*1024, oss.Routines(3), oss.Checkpoint(true, ""))
			} else {
				uploadErr = bucket.PutObjectFromFile(objectKey, localPath)
			}
		}
	}

	if uploadErr != nil {
		return "", fmt.Errorf("upload failed after retry: %v", uploadErr)
	}

	// 6. Construct Public URL
	endpoint := m.config.OSSEndpoint
	if !strings.HasPrefix(endpoint, "http") {
		endpoint = "https://" + endpoint
	}
	
	url := ""
	if strings.Contains(endpoint, "://") {
		parts := strings.Split(endpoint, "://")
		url = fmt.Sprintf("%s://%s.%s/%s", parts[0], m.config.OSSBucketName, parts[1], objectKey)
	} else {
		url = fmt.Sprintf("https://%s.%s/%s", m.config.OSSBucketName, endpoint, objectKey)
	}

	return url, nil
}
