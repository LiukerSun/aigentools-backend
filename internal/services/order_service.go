package services

import (
	"aigentools-backend/config"
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// 错误定义
var (
	ErrOrderNotFound      = errors.New("order not found")
	ErrOrderAlreadyPaid   = errors.New("order already paid")
	ErrOrderCancelled     = errors.New("order has been cancelled")
	ErrInvalidOrderStatus = errors.New("invalid order status for this operation")
)

// OrderFilter 订单查询过滤条件
type OrderFilter struct {
	UserID    *uint
	Status    *string
	OrderType *string
	StartTime *time.Time
	EndTime   *time.Time
	MinAmount *float64
	MaxAmount *float64
	Page      int
	Limit     int
}

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	UserID      uint
	Amount      float64
	OrderType   string // "payment" or "manual"
	PaymentUUID string // 仅 payment 类型需要
	Remark      string
}

// CreateOrder 创建订单（通用方法）
func CreateOrder(req CreateOrderRequest) (*models.PaymentOrderRecord, error) {
	order := &models.PaymentOrderRecord{
		ID:          strings.ReplaceAll(uuid.New().String(), "-", ""),
		UserID:      req.UserID,
		Amount:      req.Amount,
		Status:      models.OrderStatusPending,
		OrderType:   req.OrderType,
		PaymentUUID: req.PaymentUUID,
		Remark:      req.Remark,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := database.DB.Create(order).Error; err != nil {
		return nil, err
	}
	return order, nil
}

// CreateManualOrder 管理员创建手动订单
func CreateManualOrder(userID uint, amount float64, remark string) (*models.PaymentOrderRecord, error) {
	return CreateOrder(CreateOrderRequest{
		UserID:    userID,
		Amount:    amount,
		OrderType: models.OrderTypeManual,
		Remark:    remark,
	})
}

// CompleteOrder 完成订单并充值（管理员手动完成或支付回调完成）
func CompleteOrder(orderID string, operatorID uint, operatorName string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 加锁查询订单
		var order models.PaymentOrderRecord
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&order, "id = ?", orderID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrderNotFound
			}
			return err
		}

		// 2. 检查订单状态
		if order.Status == models.OrderStatusPaid {
			return ErrOrderAlreadyPaid
		}
		if order.Status == models.OrderStatusCancelled {
			return ErrOrderCancelled
		}

		// 3. 更新订单状态
		now := time.Now()
		order.Status = models.OrderStatusPaid
		order.CompletedAt = &now
		order.CompletedBy = operatorID
		order.UpdatedAt = now
		if err := tx.Save(&order).Error; err != nil {
			return err
		}

		// 4. 加锁查询用户
		var user models.User
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&user, order.UserID).Error; err != nil {
			return err
		}

		// 5. 更新用户余额
		balanceBefore := user.Balance
		user.Balance += order.Amount
		user.Version++
		if err := tx.Save(&user).Error; err != nil {
			return err
		}

		// 6. 创建交易记录
		transactionType := models.TransactionTypeUserTopup
		reason := fmt.Sprintf("充值订单: %s", order.ID)
		if order.OrderType == models.OrderTypeManual {
			transactionType = models.TransactionTypeManualTopup
			reason = fmt.Sprintf("管理员手动充值订单: %s", order.ID)
			if order.Remark != "" {
				reason += fmt.Sprintf(" (%s)", order.Remark)
			}
		}

		transaction := models.Transaction{
			UserID:        user.ID,
			Amount:        order.Amount,
			BalanceBefore: balanceBefore,
			BalanceAfter:  user.Balance,
			Reason:        reason,
			Operator:      operatorName,
			OperatorID:    operatorID,
			Type:          transactionType,
			CreatedAt:     time.Now(),
		}

		// 生成 Hash
		cfg, _ := config.LoadConfig()
		secret := "default-secret"
		if cfg != nil && cfg.JWTSecret != "" {
			secret = cfg.JWTSecret
		}
		transaction.Hash = transaction.GenerateHash(secret)

		if err := tx.Create(&transaction).Error; err != nil {
			return err
		}

		return nil
	})
}

// CancelOrder 取消订单
func CancelOrder(orderID string) error {
	var order models.PaymentOrderRecord
	if err := database.DB.First(&order, "id = ?", orderID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrOrderNotFound
		}
		return err
	}

	if order.Status != models.OrderStatusPending {
		return ErrInvalidOrderStatus
	}

	return database.DB.Model(&order).Updates(map[string]interface{}{
		"status":     models.OrderStatusCancelled,
		"updated_at": time.Now(),
	}).Error
}

// FindOrders 查询订单列表
func FindOrders(filter OrderFilter) ([]models.PaymentOrderRecord, int64, error) {
	var orders []models.PaymentOrderRecord
	var total int64

	query := database.DB.Model(&models.PaymentOrderRecord{})

	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.OrderType != nil {
		query = query.Where("order_type = ?", *filter.OrderType)
	}
	if filter.StartTime != nil {
		query = query.Where("created_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("created_at <= ?", *filter.EndTime)
	}
	if filter.MinAmount != nil {
		query = query.Where("amount >= ?", *filter.MinAmount)
	}
	if filter.MaxAmount != nil {
		query = query.Where("amount <= ?", *filter.MaxAmount)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.Limit
	if err := query.Order("created_at desc").Limit(filter.Limit).Offset(offset).Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// GetOrderByID 根据ID获取订单
func GetOrderByID(orderID string) (*models.PaymentOrderRecord, error) {
	var order models.PaymentOrderRecord
	if err := database.DB.First(&order, "id = ?", orderID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}
