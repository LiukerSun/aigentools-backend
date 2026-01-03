package user_test

import (
	"aigentools-backend/internal/api/v1/user"
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestCurrentUser(t *testing.T) {
	setupTestDB()
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		user           models.User
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "Normal User with Credit",
			user: models.User{
				Username:    "credituser",
				Role:        "user",
				IsActive:    true,
				Balance:     1000.0,
				CreditLimit: 5000.0,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Code int               `json:"status"`
					Data user.UserResponse `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.Equal(t, 200, resp.Code)
				assert.NotNil(t, resp.Data.Credit)
				// Total = 5000 (Credit) + 1000 (Own) = 6000
				assert.Equal(t, 6000.0, resp.Data.Credit.Total)
				// Available = 5000 + 1000 = 6000
				assert.Equal(t, 6000.0, resp.Data.Credit.Available)
				assert.Equal(t, 0.0, resp.Data.Credit.Used)
				assert.Equal(t, 0.0, resp.Data.Credit.UsagePercentage)
			},
		},
		{
			name: "User with Zero Balance and Limit",
			user: models.User{
				Username:    "zerouser",
				Role:        "user",
				IsActive:    true,
				Balance:     0.0,
				CreditLimit: 0.0,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Code int               `json:"status"`
					Data user.UserResponse `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.Equal(t, 200, resp.Code)
				assert.NotNil(t, resp.Data.Credit)
				assert.Equal(t, 0.0, resp.Data.Credit.Total)
				assert.Equal(t, 0.0, resp.Data.Credit.Available)
				assert.Equal(t, 0.0, resp.Data.Credit.Used)
				assert.Equal(t, 0.0, resp.Data.Credit.UsagePercentage)
			},
		},
		{
			name: "User with Negative Balance (Overdraft/Debt)",
			user: models.User{
				Username:    "debtuser",
				Role:        "user",
				IsActive:    true,
				Balance:     -500.0,
				CreditLimit: 2000.0,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Code int               `json:"status"`
					Data user.UserResponse `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.Equal(t, 200, resp.Code)
				assert.NotNil(t, resp.Data.Credit)
				// Total = 2000 (Credit) + 0 (Own) = 2000
				assert.Equal(t, 2000.0, resp.Data.Credit.Total)
				// Available = 2000 + (-500) = 1500
				assert.Equal(t, 1500.0, resp.Data.Credit.Available)
				// Used = 2000 - 1500 = 500
				assert.Equal(t, 500.0, resp.Data.Credit.Used)
				assert.Equal(t, 25.0, resp.Data.Credit.UsagePercentage) // (500/2000)*100 = 25
			},
		},
		{
			name: "User with Surplus Balance (Deposit)",
			user: models.User{
				Username:    "richuser",
				Role:        "user",
				IsActive:    true,
				Balance:     6000.0,
				CreditLimit: 5000.0,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Code int               `json:"status"`
					Data user.UserResponse `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.Equal(t, 200, resp.Code)
				assert.NotNil(t, resp.Data.Credit)
				// Total = 5000 + 6000 = 11000
				assert.Equal(t, 11000.0, resp.Data.Credit.Total)
				// Available = 5000 + 6000 = 11000
				assert.Equal(t, 11000.0, resp.Data.Credit.Available)
				assert.Equal(t, 0.0, resp.Data.Credit.Used)
				assert.Equal(t, 0.0, resp.Data.Credit.UsagePercentage)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure user exists in DB if needed by middleware or token generation logic?
			// CurrentUser handler uses c.Get("user"), which is usually set by middleware.
			// But it also calls utils.GenerateToken(u.ID, u.Role).
			// And the handler directly uses the user object from context to respond,
			// except for token generation.
			// Wait, does it fetch from DB?
			// Code: u := user.(models.User) ... c.JSON(...)
			// It does NOT fetch from DB inside the handler. It uses the object from context.
			// So we don't strictly need to insert into DB for the handler logic itself,
			// UNLESS middleware is involved. But here we mock middleware.

			// However, GenerateToken might be fine.
			// Let's just pass the user object in context.

			r := gin.New()
			r.Use(func(c *gin.Context) {
				// Simulate auth middleware
				// We need to set ID because GenerateToken might need it.
				// Let's ensure ID is set.
				if tt.user.ID == 0 {
					tt.user.ID = 1 // Mock ID
				}
				c.Set("user", tt.user)
				c.Next()
			})
			r.GET("/auth/user", user.CurrentUser)

			req, _ := http.NewRequest(http.MethodGet, "/auth/user", nil)
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
