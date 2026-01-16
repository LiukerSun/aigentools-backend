package services

import (
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

const (
	PromptCacheKeyPrefix = "prompt:code:"
	PromptCacheDuration  = 24 * time.Hour
)

// CreatePrompt creates a new prompt
func CreatePrompt(code, content string) (*models.Prompt, error) {
	// Check if exists
	var existing models.Prompt
	if err := database.DB.Where("code = ?", code).First(&existing).Error; err == nil {
		return nil, errors.New("prompt code already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	prompt := &models.Prompt{
		Code:    code,
		Content: content,
	}

	if err := database.DB.Create(prompt).Error; err != nil {
		return nil, err
	}

	return prompt, nil
}

// BatchCreatePrompts creates multiple prompts in a transaction
func BatchCreatePrompts(requests []struct{ Code, Content string }) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		for _, req := range requests {
			prompt := &models.Prompt{
				Code:    req.Code,
				Content: req.Content,
			}
			if err := tx.Create(prompt).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdatePrompt updates an existing prompt
func UpdatePrompt(code, content string) (*models.Prompt, error) {
	var prompt models.Prompt
	if err := database.DB.Where("code = ?", code).First(&prompt).Error; err != nil {
		return nil, err
	}

	prompt.Content = content
	if err := database.DB.Save(&prompt).Error; err != nil {
		return nil, err
	}

	// Invalidate cache
	cacheKey := PromptCacheKeyPrefix + code
	database.RedisClient.Del(database.Ctx, cacheKey)

	return &prompt, nil
}

// DeletePrompt deletes a prompt by code
func DeletePrompt(code string) error {
	result := database.DB.Where("code = ?", code).Delete(&models.Prompt{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("prompt not found")
	}

	// Invalidate cache
	cacheKey := PromptCacheKeyPrefix + code
	database.RedisClient.Del(database.Ctx, cacheKey)

	return nil
}

// GetPromptByCode retrieves a prompt by code, using cache
func GetPromptByCode(code string) (*models.Prompt, error) {
	cacheKey := PromptCacheKeyPrefix + code

	// Try cache
	val, err := database.RedisClient.Get(database.Ctx, cacheKey).Result()
	if err == nil {
		var prompt models.Prompt
		if err := json.Unmarshal([]byte(val), &prompt); err == nil {
			return &prompt, nil
		}
	}

	// Fetch from DB
	var prompt models.Prompt
	if err := database.DB.Where("code = ?", code).First(&prompt).Error; err != nil {
		return nil, err
	}

	// Set cache
	if data, err := json.Marshal(prompt); err == nil {
		database.RedisClient.Set(database.Ctx, cacheKey, data, PromptCacheDuration)
	}

	return &prompt, nil
}

// ListPrompts retrieves a paginated list of prompts
func ListPrompts(page, pageSize int) ([]models.Prompt, int64, error) {
	var prompts []models.Prompt
	var total int64

	db := database.DB.Model(&models.Prompt{})

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := db.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&prompts).Error; err != nil {
		return nil, 0, err
	}

	return prompts, total, nil
}
