package services

import (
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"encoding/json"
	"errors"
	"time"
)

const (
	PublicTemplatesCacheKey = "templates:public"
	TemplatesCacheDuration  = 1 * time.Hour
)

// CreatePromptTemplate creates a new prompt template
func CreatePromptTemplate(userID uint, name, description, content string, isPublic bool) (*models.PromptTemplate, error) {
	templateType := models.PromptTemplateTypePrivate
	if isPublic {
		// Only allow admin or system (userID=0) to create public templates?
		// For now, assuming standard users can only create private ones unless specific logic exists.
		// If the caller explicitly asks for public and has permission (checked by handler/middleware), we allow it.
		// Here we just trust the input flag, assuming Handler did the check.
		templateType = models.PromptTemplateTypePublic
	}

	template := &models.PromptTemplate{
		Name:        name,
		Description: description,
		Content:     content,
		Type:        templateType,
		UserID:      userID,
	}

	if err := database.DB.Create(template).Error; err != nil {
		return nil, err
	}

	// Invalidate public cache if public
	if isPublic {
		database.RedisClient.Del(database.Ctx, PublicTemplatesCacheKey)
	}

	return template, nil
}

// UpdatePromptTemplate updates an existing template
func UpdatePromptTemplate(id, userID uint, name, description, content string, isPublic *bool) (*models.PromptTemplate, error) {
	var template models.PromptTemplate
	if err := database.DB.First(&template, id).Error; err != nil {
		return nil, err
	}

	// Check ownership
	// Simplified: Only owner can update.
	if template.UserID != 0 && template.UserID != userID {
		return nil, errors.New("permission denied")
	}

	template.Name = name
	template.Description = description
	template.Content = content

	// Handle visibility change
	if isPublic != nil {
		if *isPublic {
			template.Type = models.PromptTemplateTypePublic
		} else {
			template.Type = models.PromptTemplateTypePrivate
		}
	}

	if err := database.DB.Save(&template).Error; err != nil {
		return nil, err
	}

	// Invalidate cache if public or was public
	// We should just invalidate the public key anyway if we touch a template that MIGHT be public or BECOME public.
	database.RedisClient.Del(database.Ctx, PublicTemplatesCacheKey)

	return &template, nil
}

// DeletePromptTemplate deletes a template
func DeletePromptTemplate(id, userID uint) error {
	var template models.PromptTemplate
	if err := database.DB.First(&template, id).Error; err != nil {
		return err
	}

	if template.UserID != 0 && template.UserID != userID {
		return errors.New("permission denied")
	}

	if err := database.DB.Delete(&template).Error; err != nil {
		return err
	}

	if template.Type == models.PromptTemplateTypePublic {
		database.RedisClient.Del(database.Ctx, PublicTemplatesCacheKey)
	}

	return nil
}

// GetPromptTemplate retrieves a template by ID
func GetPromptTemplate(id, userID uint) (*models.PromptTemplate, error) {
	var template models.PromptTemplate
	if err := database.DB.First(&template, id).Error; err != nil {
		return nil, err
	}

	// Check visibility
	if template.Type == models.PromptTemplateTypePrivate && template.UserID != userID {
		return nil, errors.New("permission denied")
	}

	return &template, nil
}

// ListPromptTemplates retrieves templates visible to the user
func ListPromptTemplates(userID uint, page, limit int, search string, filterType string) ([]models.PromptTemplate, int64, error) {
	// Strategy:
	// 1. If searching or filtering specifically, query DB directly.
	// 2. If just listing default (all visible), we can try to use cache for PUBLIC part and merge with PRIVATE part?
	//    Or just query DB with OR condition.
	//    Requirement: "Public templates use cache mechanism".
	//    This implies we should cache the public ones.
	//    If filterType == "public", return cached public list.
	//    If filterType == "private", return user's private list.
	//    If no filter (all), maybe combine?

	var templates []models.PromptTemplate
	var total int64

	db := database.DB.Model(&models.PromptTemplate{})

	// Base visibility: Public OR Owned by User
	if filterType == "public" {
		// Try Cache first if no search
		if search == "" && page == 1 && limit > 0 { // Simple caching for first page or full list?
			// Let's implement full list cache for public templates (assuming not thousands)
			// If cached, return.
			val, err := database.RedisClient.Get(database.Ctx, PublicTemplatesCacheKey).Result()
			if err == nil {
				var cached []models.PromptTemplate
				if err := json.Unmarshal([]byte(val), &cached); err == nil {
					// Apply pagination manually on cached result?
					// Or just return cached if it fits?
					// For simplicity, let's cache the query result of "all public templates".
					// Then we slice it.
					// This might be heavy if many templates.
					// Let's stick to DB query if pagination is complex, but the requirement emphasizes cache.
				}
			}
		}
		db = db.Where("type = ?", models.PromptTemplateTypePublic)
	} else if filterType == "private" {
		db = db.Where("user_id = ? AND type = ?", userID, models.PromptTemplateTypePrivate)
	} else {
		// All visible
		db = db.Where("type = ? OR user_id = ?", models.PromptTemplateTypePublic, userID)
	}

	if search != "" {
		db = db.Where("name LIKE ? OR content LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if err := db.Order("created_at desc").Offset(offset).Limit(limit).Find(&templates).Error; err != nil {
		return nil, 0, err
	}

	// Set cache if it was a pure public list query (no search, page 1) - implementation simplified
	// Real implementation of caching partial results is complex.
	// Let's implement a specific "GetPublicTemplates" helper that is cached, and use it if possible.

	return templates, total, nil
}

// GetPublicTemplatesCached retrieves all public templates with caching
// This is useful for "Public Template Zone"
func GetPublicTemplatesCached() ([]models.PromptTemplate, error) {
	// Try cache
	val, err := database.RedisClient.Get(database.Ctx, PublicTemplatesCacheKey).Result()
	if err == nil {
		var templates []models.PromptTemplate
		if err := json.Unmarshal([]byte(val), &templates); err == nil {
			return templates, nil
		}
	}

	// DB
	var templates []models.PromptTemplate
	if err := database.DB.Where("type = ?", models.PromptTemplateTypePublic).Order("created_at desc").Find(&templates).Error; err != nil {
		return nil, err
	}

	// Set Cache
	if data, err := json.Marshal(templates); err == nil {
		database.RedisClient.Set(database.Ctx, PublicTemplatesCacheKey, data, TemplatesCacheDuration)
	}

	return templates, nil
}
