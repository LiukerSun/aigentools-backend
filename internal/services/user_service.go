package services

import (
	"aigentools-backend/config"
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
var ErrInsufficientBalance = errors.New("insufficient balance")

// ...

// DeductBalance decreases user's balance and checks for sufficient funds.
func DeductBalance(userID uint, amount float64, reason string, meta TransactionMetadata) (*models.User, error) {
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	user, err := DeductBalanceTx(tx, userID, amount, reason, meta)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Invalidate cache
	if database.RedisClient != nil {
		cacheKey := fmt.Sprintf("user:%d", userID)
		database.RedisClient.Del(database.Ctx, cacheKey)
	}

	return user, nil
}

// DeductBalanceTx executes the deduction logic within a provided transaction.
func DeductBalanceTx(tx *gorm.DB, userID uint, amount float64, reason string, meta TransactionMetadata) (*models.User, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}

	var user models.User
	if err := tx.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Calculate available balance (Balance + CreditLimit)
	availableBalance := user.Balance + user.CreditLimit
	if availableBalance < amount {
		return nil, ErrInsufficientBalance
	}

	balanceBefore := user.Balance
	balanceAfter := balanceBefore - amount
	totalConsumed := user.TotalConsumed + amount

	// Update user balance and version
	currentVersion := user.Version
	updates := map[string]interface{}{
		"balance":        balanceAfter,
		"total_consumed": totalConsumed,
		"version":        currentVersion + 1,
	}

	// Apply updates with optimistic lock
	result := tx.Model(&user).Where("version = ?", currentVersion).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrOptimisticLock
	}

	// Record transaction
	transaction := models.Transaction{
		UserID:        userID,
		Amount:        -amount, // Negative for deduction
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		Reason:        reason,
		Operator:      meta.Operator,
		OperatorID:    meta.OperatorID,
		Type:          meta.Type,
		IPAddress:     meta.IPAddress,
		DeviceInfo:    meta.DeviceInfo,
		CreatedAt:     time.Now(),
	}

	// Generate hash
	cfg, _ := config.LoadConfig()
	secret := "default-secret"
	if cfg != nil && cfg.JWTSecret != "" {
		secret = cfg.JWTSecret
	}
	transaction.Hash = transaction.GenerateHash(secret)

	if err := tx.Create(&transaction).Error; err != nil {
		return nil, err
	}

	// Return updated user object (reload to get latest state if needed, but we have values)
	// For consistency, let's update the user struct we have
	user.Balance = balanceAfter
	user.TotalConsumed = totalConsumed
	user.Version++
	return &user, nil
}

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
	Username      string
	Role          string
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

	if filter.Username != "" {
		query = query.Where("username LIKE ?", "%"+filter.Username+"%")
	}
	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}
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

	if creditLimit, ok := updates["credit_limit"].(float64); ok {
		updates["credit_limit"] = creditLimit
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

// TransactionMetadata contains additional information for transaction logging
type TransactionMetadata struct {
	Operator   string
	OperatorID uint
	Type       models.TransactionType
	IPAddress  string
	DeviceInfo string
}

// AdjustBalance updates user's balance and records the transaction.
func AdjustBalance(userID uint, amount float64, reason string, meta TransactionMetadata) (*models.User, error) {
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var user models.User
	if err := tx.First(&user, userID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	balanceBefore := user.Balance
	balanceAfter := balanceBefore + amount

	// Update user balance and version
	currentVersion := user.Version
	updates := map[string]interface{}{
		"balance": balanceAfter,
		"version": currentVersion + 1,
	}

	// Update TotalConsumed if this is a refund (amount > 0 and type is refund)
	// Or if it's a manual deduction (amount < 0)
	if meta.Type == models.TransactionTypeUserRefund {
		// Refund reduces total consumed
		updates["total_consumed"] = user.TotalConsumed - amount
	} else if amount < 0 {
		// Manual deduction (admin) increases total consumed?
		// Assuming admin debit counts as consumption.
		updates["total_consumed"] = user.TotalConsumed + (-amount)
	}

	// Status management logic
	if balanceAfter == 0 {
		updates["is_active"] = false
		now := time.Now()
		updates["deactivated_at"] = &now
	}

	// Apply updates with optimistic lock
	result := tx.Model(&user).Where("version = ?", currentVersion).Updates(updates)
	if result.Error != nil {
		tx.Rollback()
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		return nil, ErrOptimisticLock
	}

	// Record transaction
	transaction := models.Transaction{
		UserID:        userID,
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		Reason:        reason,
		Operator:      meta.Operator,
		OperatorID:    meta.OperatorID,
		Type:          meta.Type,
		IPAddress:     meta.IPAddress,
		DeviceInfo:    meta.DeviceInfo,
		CreatedAt:     time.Now(),
	}

	// Generate hash for tamper-proofing
	// In production, secret should come from config/env
	cfg, _ := config.LoadConfig() // Assuming LoadConfig is cheap or cached
	secret := "default-secret"
	if cfg != nil && cfg.JWTSecret != "" {
		secret = cfg.JWTSecret
	}
	transaction.Hash = transaction.GenerateHash(secret)

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Invalidate cache
	if database.RedisClient != nil {
		cacheKey := fmt.Sprintf("user:%d", userID)
		database.RedisClient.Del(database.Ctx, cacheKey)
	}

	// Fetch updated user
	database.DB.First(&user, userID)

	return &user, nil
}

// DeleteUser permanently deletes a user and their transactions.
func DeleteUser(id uint) error {
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if user exists
	var user models.User
	if err := tx.First(&user, id).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	// Delete all transactions for the user
	if err := tx.Where("user_id = ?", id).Delete(&models.Transaction{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete the user
	if err := tx.Delete(&models.User{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Invalidate cache
	if database.RedisClient != nil {
		cacheKey := fmt.Sprintf("user:%d", id)
		database.RedisClient.Del(database.Ctx, cacheKey)
	}

	// Log operation (placeholder for audit log)
	fmt.Printf("User %d and their transactions permanently deleted.\n", id)

	return nil
}
