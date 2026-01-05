package services

import (
	"aigentools-backend/config"
	"fmt"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type STSCredentials struct {
	AccessKeyId     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
	SecurityToken   string `json:"securityToken"`
	Expiration      string `json:"expiration"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
}

// UploadFile uploads a file from a local path to OSS and returns the public URL
func UploadFile(localPath string, objectKey string) (string, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return "", err
	}

	client, err := oss.New(cfg.OSSEndpoint, cfg.OSSAccessKeyID, cfg.OSSAccessKeySecret)
	if err != nil {
		return "", err
	}

	bucket, err := client.Bucket(cfg.OSSBucketName)
	if err != nil {
		return "", err
	}

	err = bucket.PutObjectFromFile(objectKey, localPath)
	if err != nil {
		return "", err
	}

	// Assuming the bucket is public-read or we generate a signed URL.
	// For public-read buckets: https://<bucket>.<endpoint>/<objectKey>
	// But endpoint might contain http/https.
	endpoint := cfg.OSSEndpoint
	if !strings.HasPrefix(endpoint, "http") {
		endpoint = "https://" + endpoint
	}

	// Simple URL construction (assumes standard OSS URL format)
	// If endpoint is "oss-cn-beijing.aliyuncs.com", url is "https://bucket.oss-cn-beijing.aliyuncs.com/key"
	// If endpoint already has protocol, we need to be careful.

	// Let's use a safer way if possible, or just simple string manipulation
	url := ""
	if strings.Contains(endpoint, "://") {
		parts := strings.Split(endpoint, "://")
		url = fmt.Sprintf("%s://%s.%s/%s", parts[0], cfg.OSSBucketName, parts[1], objectKey)
	} else {
		url = fmt.Sprintf("https://%s.%s/%s", cfg.OSSBucketName, endpoint, objectKey)
	}

	return url, nil
}

func GetOSSTSToken() (*STSCredentials, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}

	// STS client requires region ID without "oss-" prefix (e.g., "cn-beijing" instead of "oss-cn-beijing")
	stsRegion := cfg.OSSRegion
	if after, ok := strings.CutPrefix(stsRegion, "oss-"); ok {
		stsRegion = after
	}

	client, err := sts.NewClientWithAccessKey(stsRegion, cfg.OSSAccessKeyID, cfg.OSSAccessKeySecret)
	if err != nil {
		return nil, err
	}

	request := sts.CreateAssumeRoleRequest()
	request.Scheme = "https"
	request.RoleArn = cfg.OSSRoleArn
	request.RoleSessionName = "aigentools-session"
	request.DurationSeconds = "3600" // 1 hour

	response, err := client.AssumeRole(request)
	if err != nil {
		return nil, err
	}

	return &STSCredentials{
		AccessKeyId:     response.Credentials.AccessKeyId,
		AccessKeySecret: response.Credentials.AccessKeySecret,
		SecurityToken:   response.Credentials.SecurityToken,
		Expiration:      response.Credentials.Expiration,
		Region:          cfg.OSSRegion,
		Bucket:          cfg.OSSBucketName,
	}, nil
}
