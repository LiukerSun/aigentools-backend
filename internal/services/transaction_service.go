package services

import (
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"bytes"
	"encoding/csv"
	"fmt"
	"time"
)

// TransactionFilter defines criteria for filtering transactions
type TransactionFilter struct {
	UserID    *uint
	Type      *models.TransactionType
	StartTime *time.Time
	EndTime   *time.Time
	MinAmount *float64
	MaxAmount *float64
	Page      int
	Limit     int
}

// FindTransactions retrieves a paginated list of transactions with filtering
func FindTransactions(filter TransactionFilter) ([]models.Transaction, int64, error) {
	var transactions []models.Transaction
	var total int64

	query := database.DB.Model(&models.Transaction{})

	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.Type != nil {
		query = query.Where("type = ?", *filter.Type)
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
	if err := query.Order("created_at desc").Limit(filter.Limit).Offset(offset).Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

// GenerateTransactionCSV generates a CSV file content for transactions
func GenerateTransactionCSV(transactions []models.Transaction) ([]byte, error) {
	b := &bytes.Buffer{}
	w := csv.NewWriter(b)

	// Write header
	header := []string{
		"ID", "Time", "User ID", "Type", "Amount",
		"Balance Before", "Balance After", "Reason",
		"Operator", "IP Address", "Device Info", "Hash",
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}

	// Write data
	for _, t := range transactions {
		record := []string{
			fmt.Sprintf("%d", t.ID),
			t.CreatedAt.Format(time.RFC3339Nano),
			fmt.Sprintf("%d", t.UserID),
			string(t.Type),
			fmt.Sprintf("%.2f", t.Amount),
			fmt.Sprintf("%.2f", t.BalanceBefore),
			fmt.Sprintf("%.2f", t.BalanceAfter),
			t.Reason,
			t.Operator,
			t.IPAddress,
			t.DeviceInfo,
			t.Hash,
		}
		if err := w.Write(record); err != nil {
			return nil, err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
