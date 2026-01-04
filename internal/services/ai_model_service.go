package services

import (
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
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
	return database.DB.Save(model).Error
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
