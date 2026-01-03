package transaction_test

import (
	"aigentools-backend/internal/api/v1/admin/transaction"
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"aigentools-backend/internal/services"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() {
	// Use in-memory SQLite for testing
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Drop tables if exist to ensure clean state and schema update
	db.Migrator().DropTable(&models.User{}, &models.Transaction{})

	// Migrate schema
	err = db.AutoMigrate(&models.User{}, &models.Transaction{})
	if err != nil {
		panic("failed to migrate database")
	}

	// Clean up data
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM transactions")

	// Assign to global DB
	database.DB = db
}

func TestListTransactions(t *testing.T) {
	setupTestDB()
	gin.SetMode(gin.TestMode)

	// Seed transactions
	t1 := models.Transaction{
		UserID:        1,
		Amount:        100.0,
		BalanceBefore: 0.0,
		BalanceAfter:  100.0,
		Reason:        "Deposit",
		Operator:      "admin",
		Type:          models.TransactionTypeSystemAdmin,
		CreatedAt:     time.Now().Add(-2 * time.Hour),
		Hash:          "hash1",
	}
	t2 := models.Transaction{
		UserID:        1,
		Amount:        -50.0,
		BalanceBefore: 100.0,
		BalanceAfter:  50.0,
		Reason:        "Consume",
		Operator:      "system",
		Type:          models.TransactionTypeUserConsume,
		CreatedAt:     time.Now().Add(-1 * time.Hour),
		Hash:          "hash2",
	}
	t3 := models.Transaction{
		UserID:        2,
		Amount:        200.0,
		BalanceBefore: 0.0,
		BalanceAfter:  200.0,
		Reason:        "Deposit",
		Operator:      "admin",
		Type:          models.TransactionTypeSystemAdmin,
		CreatedAt:     time.Now(),
		Hash:          "hash3",
	}
	database.DB.Create(&t1)
	database.DB.Create(&t2)
	database.DB.Create(&t3)

	tests := []struct {
		name           string
		query          string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "List All",
			query:          "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Code int                                 `json:"status"`
					Data transaction.TransactionListResponse `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.Equal(t, 200, resp.Code)
				assert.Equal(t, int64(3), resp.Data.Total)
				assert.Len(t, resp.Data.Transactions, 3)
			},
		},
		{
			name:           "Filter by UserID",
			query:          "?user_id=1",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Code int                                 `json:"status"`
					Data transaction.TransactionListResponse `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.Equal(t, 200, resp.Code)
				assert.Equal(t, int64(2), resp.Data.Total)
				assert.Equal(t, uint(1), resp.Data.Transactions[0].UserID)
			},
		},
		{
			name:           "Filter by Type",
			query:          "?type=" + string(models.TransactionTypeUserConsume),
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Code int                                 `json:"status"`
					Data transaction.TransactionListResponse `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.Equal(t, 200, resp.Code)
				assert.Equal(t, int64(1), resp.Data.Total)
				assert.Equal(t, models.TransactionTypeUserConsume, resp.Data.Transactions[0].Type)
			},
		},
		{
			name:           "Filter by MinAmount",
			query:          "?min_amount=150",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Code int                                 `json:"status"`
					Data transaction.TransactionListResponse `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.Equal(t, 200, resp.Code)
				assert.Equal(t, int64(1), resp.Data.Total)
				assert.Equal(t, 200.0, resp.Data.Transactions[0].Amount)
			},
		},
		{
			name:           "Filter by MaxAmount",
			query:          "?max_amount=-10",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Code int                                 `json:"status"`
					Data transaction.TransactionListResponse `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.Equal(t, 200, resp.Code)
				assert.Equal(t, int64(1), resp.Data.Total)
				assert.Equal(t, -50.0, resp.Data.Transactions[0].Amount)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/admin/transactions", transaction.ListTransactions)

			req, _ := http.NewRequest(http.MethodGet, "/admin/transactions"+tt.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Logf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestExportTransactions(t *testing.T) {
	setupTestDB()
	gin.SetMode(gin.TestMode)

	// Seed transactions
	t1 := models.Transaction{
		UserID:        1,
		Amount:        100.0,
		BalanceBefore: 0.0,
		BalanceAfter:  100.0,
		Reason:        "Deposit",
		Operator:      "admin",
		Type:          models.TransactionTypeSystemAdmin,
		CreatedAt:     time.Now(),
		Hash:          "hash1",
	}
	database.DB.Create(&t1)

	r := gin.New()
	r.GET("/admin/transactions/export", transaction.ExportTransactions)

	req, _ := http.NewRequest(http.MethodGet, "/admin/transactions/export", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/csv", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment; filename=")

	csvContent := w.Body.String()
	assert.Contains(t, csvContent, "ID,Time,User ID,Type,Amount")
	assert.Contains(t, csvContent, "100.00")
	assert.Contains(t, csvContent, "Deposit")
}

func TestGenerateTransactionCSV(t *testing.T) {
	trans := []models.Transaction{
		{
			ID:            1,
			UserID:        10,
			Amount:        50.50,
			BalanceBefore: 100,
			BalanceAfter:  150.50,
			Reason:        "Test",
			Operator:      "admin",
			Type:          models.TransactionTypeSystemAdmin,
			CreatedAt:     time.Now(),
			IPAddress:     "127.0.0.1",
			DeviceInfo:    "Mozilla",
			Hash:          "abc",
		},
	}

	data, err := services.GenerateTransactionCSV(trans)
	assert.NoError(t, err)
	assert.NotNil(t, data)

	content := string(data)
	assert.Contains(t, content, "50.50")
	assert.Contains(t, content, "127.0.0.1")
}
