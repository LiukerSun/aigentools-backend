package services

import (
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrUserNotFound = errors.New("user not found")
var ErrOptimisticLock = errors.New("data has been modified by another user, please refresh and try again")

func FindUserByID(userID uint) (models.User, error) {
	// Try cache
	cacheKey := fmt.Sprintf("user:%d", userID)
	if database.RedisClient != nil {
		val, err := database.RedisClient.Get(database.Ctx, cacheKey).Result()
		if err == nil {
			var user models.User
			if err := json.Unmarshal([]byte(val), &user); err == nil {
				return user, nil
			}
		}
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return user, err
	}

	// Set cache
	if database.RedisClient != nil {
		if data, err := json.Marshal(user); err == nil {
			database.RedisClient.Set(database.Ctx, cacheKey, data, time.Hour)
		}
	}

	return user, nil
}

// UserFilter defines criteria for filtering users
type UserFilter struct {
	IsActive      *bool
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	Page          int
	Limit         int
}

// FindUsers retrieves a paginated list of users with optional filtering.
func FindUsers(filter UserFilter) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	query := database.DB.Model(&models.User{})

	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.CreatedAfter != nil {
		query = query.Where("created_at >= ?", *filter.CreatedAfter)
	}
	if filter.CreatedBefore != nil {
		query = query.Where("created_at <= ?", *filter.CreatedBefore)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.Limit
	if err := query.Limit(filter.Limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// UpdateUser updates a user with optimistic locking and selective fields.
func UpdateUser(id uint, updates map[string]interface{}, operator string) (*models.User, error) {
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var user models.User
	if err := tx.First(&user, id).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Password handling
	if password, ok := updates["password"].(string); ok && password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		updates["password"] = string(hashedPassword)
	}

	// Status handling
	if isActive, ok := updates["is_active"].(bool); ok {
		now := time.Now()
		if isActive {
			updates["activated_at"] = &now
			updates["deactivated_at"] = nil
		} else {
			updates["deactivated_at"] = &now
		}
	}

	// Optimistic Lock Check
	currentVersion := user.Version
	updates["version"] = currentVersion + 1

	// Apply updates
	// We use Where("version = ?", currentVersion) to ensure atomic update with version check
	result := tx.Model(&user).Where("version = ?", currentVersion).Updates(updates)
	if result.Error != nil {
		tx.Rollback()
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		return nil, ErrOptimisticLock
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Invalidate cache
	if database.RedisClient != nil {
		cacheKey := fmt.Sprintf("user:%d", id)
		database.RedisClient.Del(database.Ctx, cacheKey)
	}

	// Log operation (placeholder for audit log)
	fmt.Printf("User %d updated by %s: %v\n", id, operator, updates)

	// Fetch updated user to return full object
	database.DB.First(&user, id)

	return &user, nil
}
