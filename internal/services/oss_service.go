package services

import (
	"aigentools-backend/config"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
)

type STSCredentials struct {
	AccessKeyId     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
	SecurityToken   string `json:"securityToken"`
	Expiration      string `json:"expiration"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
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
