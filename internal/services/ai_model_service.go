package services

import (
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"aigentools-backend/pkg/logger"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type AIModelFilter struct {
	Name   string
	Status string
	Page   int
	Limit  int
}

// FindAIModels retrieves a paginated list of AI models with filtering
func FindAIModels(filter AIModelFilter) ([]models.AIModel, int64, error) {
	var aiModels []models.AIModel
	var total int64

	query := database.DB.Model(&models.AIModel{})

	if filter.Name != "" {
		query = query.Where("name LIKE ?", "%"+filter.Name+"%")
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.Limit
	if err := query.Order("created_at desc").Limit(filter.Limit).Offset(offset).Find(&aiModels).Error; err != nil {
		return nil, 0, err
	}

	return aiModels, total, nil
}

// CreateAIModel creates a new AI model
func CreateAIModel(model *models.AIModel) error {
	logger.Log.Info("Creating new AI model", zap.Any("model", model))
	if err := models.ValidateModelParameters(model.Parameters); err != nil {
		return err
	}
	return database.DB.Create(model).Error
}

// UpdateAIModel updates an existing AI model
func UpdateAIModel(model *models.AIModel) error {
	if err := models.ValidateModelParameters(model.Parameters); err != nil {
		return err
	}
	if err := database.DB.Save(model).Error; err != nil {
		return err
	}

	// Invalidate cache
	if database.RedisClient != nil {
		cacheKey := fmt.Sprintf("model_params:%d", model.ID)
		database.RedisClient.Del(database.Ctx, cacheKey)
	}

	return nil
}

// GetAIModelByID retrieves a model by ID
func GetAIModelByID(id uint) (*models.AIModel, error) {
	var model models.AIModel
	if err := database.DB.First(&model, id).Error; err != nil {
		return nil, err
	}
	return &model, nil
}

// UpdateModelStatus updates the status of a model
func UpdateModelStatus(id uint, status models.AIModelStatus) error {
	return database.DB.Model(&models.AIModel{}).Where("id = ?", id).Update("status", status).Error
}

// GetAllModelsSimple retrieves all AI models without parameters
func GetAllModelsSimple() ([]models.AIModel, error) {
	var modelsList []models.AIModel
	// Select all columns except parameters
	// Explicitly selecting columns is better than trying to omit
	if err := database.DB.Select("id, name, description, status, url, created_at, updated_at").Where("status = ?", models.AIModelStatusOpen).Find(&modelsList).Error; err != nil {
		return nil, err
	}
	return modelsList, nil
}

// GetModelParametersByID retrieves model parameters by ID with caching
func GetModelParametersByID(id uint) (models.JSON, error) {
	// Try to get from cache
	cacheKey := fmt.Sprintf("model_params:%d", id)
	val, err := database.RedisClient.Get(database.Ctx, cacheKey).Result()
	if err == nil {
		var params models.JSON
		if err := json.Unmarshal([]byte(val), &params); err == nil {
			return params, nil
		}
		// If unmarshal fails, log and continue to fetch from DB
		logger.Log.Error("Failed to unmarshal cached parameters", zap.Error(err))
	} else if err != redis.Nil {
		// Log error but continue to DB
		logger.Log.Error("Failed to get from cache", zap.Error(err))
	}

	// Get from DB
	var model models.AIModel
	if err := database.DB.Select("parameters").First(&model, id).Error; err != nil {
		return nil, err
	}

	// Save to cache
	paramsJSON, err := json.Marshal(model.Parameters)
	if err == nil {
		if err := database.RedisClient.Set(database.Ctx, cacheKey, paramsJSON, time.Hour).Err(); err != nil {
			logger.Log.Error("Failed to set cache", zap.Error(err))
		}
	} else {
		logger.Log.Error("Failed to marshal parameters for cache", zap.Error(err))
	}

	return model.Parameters, nil
}
