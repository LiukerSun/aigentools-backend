package services

import (
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"aigentools-backend/internal/payment"
	"aigentools-backend/internal/payment/epay"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

func GetPaymentMethods() ([]models.PaymentConfig, error) {
	var methods []models.PaymentConfig
	if err := database.DB.Where("enable = ?", true).Find(&methods).Error; err != nil {
		return nil, err
	}
	return methods, nil
}

func GetAllPaymentConfigs() ([]models.PaymentConfig, error) {
	var methods []models.PaymentConfig
	if err := database.DB.Find(&methods).Error; err != nil {
		return nil, err
	}
	return methods, nil
}

func CreatePaymentConfig(name string, method string, config map[string]interface{}, enable bool) (*models.PaymentConfig, error) {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	paymentConfig := &models.PaymentConfig{
		UUID:          uuid.New().String(),
		Name:          name,
		PaymentMethod: method,
		Config:        datatypes.JSON(configJSON),
		Enable:        enable,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := database.DB.Create(paymentConfig).Error; err != nil {
		return nil, err
	}
	return paymentConfig, nil
}

func UpdatePaymentConfig(id uint, name string, config map[string]interface{}, enable *bool) (*models.PaymentConfig, error) {
	var paymentConfig models.PaymentConfig
	if err := database.DB.First(&paymentConfig, id).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}
	if name != "" {
		updates["name"] = name
	}
	if config != nil {
		configJSON, err := json.Marshal(config)
		if err != nil {
			return nil, err
		}
		updates["config"] = datatypes.JSON(configJSON)
	}
	if enable != nil {
		updates["enable"] = *enable
	}
	updates["updated_at"] = time.Now()

	if err := database.DB.Model(&paymentConfig).Updates(updates).Error; err != nil {
		return nil, err
	}
	return &paymentConfig, nil
}

func DeletePaymentConfig(id uint) error {
	return database.DB.Delete(&models.PaymentConfig{}, id).Error
}

func CreatePaymentOrder(userID uint, amount float64, paymentUUID string) (*models.PaymentOrderRecord, error) {
	order := &models.PaymentOrderRecord{
		ID:          strings.ReplaceAll(uuid.New().String(), "-", ""),
		UserID:      userID,
		Amount:      amount,
		Status:      "pending",
		PaymentUUID: paymentUUID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := database.DB.Create(order).Error; err != nil {
		return nil, err
	}
	return order, nil
}

func GetPaymentJumpURL(orderID string, paymentMethodUUID string, paymentChannel string, notifyBaseURL string, returnURL string) (string, error) {
	var config models.PaymentConfig
	if err := database.DB.Where("uuid = ?", paymentMethodUUID).First(&config).Error; err != nil {
		return "", err
	}

	if !config.Enable {
		return "", errors.New("payment method is disabled")
	}

	var driver payment.Driver
	switch config.PaymentMethod {
	case "epay":
		driver = epay.NewEpayDriver()
	default:
		return "", errors.New("unsupported payment method")
	}

	// Parse config
	var configMap map[string]interface{}
	if err := json.Unmarshal(config.Config, &configMap); err != nil {
		return "", err
	}

	if err := driver.SetConfig(configMap); err != nil {
		return "", err
	}

	// Find Order
	var order models.PaymentOrderRecord
	if err := database.DB.Where("id = ?", orderID).First(&order).Error; err != nil {
		return "", err
	}

	// Construct Notify URL with UUID
	fullNotifyURL := fmt.Sprintf("%s/%s", strings.TrimRight(notifyBaseURL, "/"), config.UUID)

	params := map[string]interface{}{
		"type": paymentChannel,
	}

	return driver.Pay(order.ID, order.Amount, fullNotifyURL, returnURL, params)
}

func HandlePaymentNotify(paymentUUID string, params map[string]interface{}) error {
	var config models.PaymentConfig
	if err := database.DB.Where("uuid = ?", paymentUUID).First(&config).Error; err != nil {
		return err
	}

	var driver payment.Driver
	switch config.PaymentMethod {
	case "epay":
		driver = epay.NewEpayDriver()
	default:
		return errors.New("unsupported payment method")
	}

	// Parse config
	var configMap map[string]interface{}
	if err := json.Unmarshal(config.Config, &configMap); err != nil {
		return err
	}
	if err := driver.SetConfig(configMap); err != nil {
		return err
	}

	isValid, orderID, externalID, err := driver.Notify(params)
	if err != nil {
		return err
	}
	if !isValid {
		return errors.New("invalid signature")
	}

	// 更新外部交易ID
	database.DB.Model(&models.PaymentOrderRecord{}).Where("id = ?", orderID).Update("external_id", externalID)

	// 使用通用的完成订单方法
	return CompleteOrder(orderID, 0, "system")
}
