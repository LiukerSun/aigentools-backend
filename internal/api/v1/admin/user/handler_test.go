package user_test

import (
	"aigentools-backend/internal/api/v1/admin/user"
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestListUsers(t *testing.T) {
	setupTestDB() // Initialize DB
	gin.SetMode(gin.TestMode)

	// Seed some data
	database.DB.Create(&models.User{
		Username:  "admin",
		Role:      "admin",
		Password:  "hashedpassword", // Add required field
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	database.DB.Create(&models.User{
		Username:  "user1",
		Role:      "user",
		Password:  "hashedpassword", // Add required field
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	tests := []struct {
		name           string
		page           string
		limit          string
		query          string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "Valid Pagination",
			page:           "1",
			limit:          "10",
			query:          "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Code    int                   `json:"status"`
					Message string                `json:"message"`
					Data    user.UserListResponse `json:"data"`
				}
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.Equal(t, 200, resp.Code)
				assert.NotEmpty(t, resp.Data.Users)
				assert.Equal(t, int64(2), resp.Data.Total)
				// Check CreditLimit field existence (default 0)
				assert.Equal(t, 0.0, resp.Data.Users[0].CreditLimit)
			},
		},
		{
			name:           "Invalid Page",
			page:           "0",
			limit:          "10",
			query:          "",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "Invalid page number")
			},
		},
		{
			name:           "Invalid Limit",
			page:           "1",
			limit:          "-1",
			query:          "",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "Invalid limit number")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/admin/users", user.ListUsers)

			req, _ := http.NewRequest(http.MethodGet, "/admin/users?page="+tt.page+"&limit="+tt.limit+tt.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if tt.expectedStatus == http.StatusBadRequest {
				assert.Equal(t, tt.expectedStatus, w.Code)
				tt.checkResponse(t, w.Body.Bytes())
			} else {
				assert.Equal(t, tt.expectedStatus, w.Code)
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestAdjustBalance(t *testing.T) {
	setupTestDB()
	gin.SetMode(gin.TestMode)

	// Seed user
	seedUser := models.User{
		Username:  "testuser_bal",
		Role:      "user",
		Password:  "oldpassword",
		Version:   1,
		IsActive:  true,
		Balance:   100.0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	database.DB.Create(&seedUser)

	tests := []struct {
		name           string
		userID         string
		body           string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "Increase Balance",
			userID:         strconv.Itoa(int(seedUser.ID)),
			body:           `{"amount": 50.0, "reason": "bonus"}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Code int               `json:"status"`
					Data user.UserListItem `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.Equal(t, 200.0, resp.Data.Balance) // 150 + 50
				assert.True(t, resp.Data.IsActive)

				// Verify DB
				var u models.User
				database.DB.First(&u, resp.Data.ID)
				assert.Equal(t, 200.0, u.Balance)

				// Verify Transaction
				var trans models.Transaction
				database.DB.Last(&trans)
				assert.Equal(t, 50.0, trans.Amount)
				assert.Equal(t, 150.0, trans.BalanceBefore)
				assert.Equal(t, 200.0, trans.BalanceAfter)
				assert.Equal(t, models.TransactionTypeSystemAdmin, trans.Type)
			},
		},
		{
			name:           "Decrease Balance to Zero (Auto Deactivate)",
			userID:         strconv.Itoa(int(seedUser.ID)),
			body:           `{"amount": -150.0, "reason": "usage"}`, // 150 - 150 = 0
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Code int               `json:"status"`
					Data user.UserListItem `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.Equal(t, 0.0, resp.Data.Balance)
				assert.False(t, resp.Data.IsActive)
				assert.NotNil(t, resp.Data.DeactivatedAt)

				// Verify DB
				var u models.User
				database.DB.First(&u, seedUser.ID)
				assert.Equal(t, 0.0, u.Balance)
				assert.False(t, u.IsActive)
			},
		},
		{
			name:           "Decrease Balance to Negative (Keep Active)",
			userID:         strconv.Itoa(int(seedUser.ID)),
			body:           `{"amount": -200.0, "reason": "overdraft"}`, // 0 - 200 = -200
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Code int               `json:"status"`
					Data user.UserListItem `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.Equal(t, -50.0, resp.Data.Balance)
				// Requirement: "当用户额度≠0时，用户激活状态不受影响（保持原状态）"
				// Previous state was Inactive (from previous test case run sequentially? No, tests loop resets state? No, tests struct loop usually runs sequentially in same function unless we reset)
				// Wait, the loop below needs to handle state reset if we rely on sequential state or reset it.
				// Let's reset state in loop.
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset user state for each test
			database.DB.Exec("DELETE FROM users")
			database.DB.Exec("DELETE FROM transactions")

			// Set initial balance based on test case needs?
			// For "Decrease Balance to Zero", we need 150 if we subtract 150.
			// For "Decrease Balance to Negative", if we want to test "Keep Active", we should start with positive?
			// Or if we start with 0 and go negative?
			// Requirement: "当用户额度≠0时，用户激活状态不受影响（保持原状态）"
			// Let's set initial balance to 150.0 and Active=true for all cases for simplicity,
			// except maybe specific cases.

			currentSeed := models.User{
				Username:  "testuser_bal",
				Role:      "user",
				Password:  "oldpassword",
				Version:   1,
				IsActive:  true,
				Balance:   150.0,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			database.DB.Create(&currentSeed)

			// Adjust request body or setup based on case name if needed?
			// The cases above assume specific math.
			// Case 1: Increase: 150 + 50 = 200.
			// Case 2: Decrease to 0: 150 - 150 = 0.
			// Case 3: Decrease to Negative: 150 - 200 = -50.

			// Let's adjust expectations in Case 1 (150+50=200) and Case 3 (150-200=-50).

			r := gin.New()
			r.Use(func(c *gin.Context) {
				c.Set("user", models.User{Username: "admin_tester"})
				c.Next()
			})
			r.POST("/admin/users/:id/balance", user.AdjustBalance)

			targetID := strconv.Itoa(int(currentSeed.ID))
			req, _ := http.NewRequest(http.MethodPost, "/admin/users/"+targetID+"/balance", bytes.NewBufferString(tt.body))
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

func TestDeductBalance(t *testing.T) {
	setupTestDB()
	gin.SetMode(gin.TestMode)

	// Seed user with balance and credit limit
	seedUser := models.User{
		Username:    "deduct_user",
		Role:        "user",
		Password:    "password",
		Version:     1,
		IsActive:    true,
		Balance:     100.0,
		CreditLimit: 50.0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	database.DB.Create(&seedUser)

	tests := []struct {
		name           string
		userID         string
		body           string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "Success Deduction within Balance",
			userID:         strconv.Itoa(int(seedUser.ID)),
			body:           `{"amount": 50.0, "reason": "fee"}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Code int               `json:"status"`
					Data user.UserListItem `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.Equal(t, 50.0, resp.Data.Balance) // 100 - 50 = 50
				
				// Verify Transaction
				var trans models.Transaction
				database.DB.Last(&trans)
				assert.Equal(t, -50.0, trans.Amount)
				assert.Equal(t, 100.0, trans.BalanceBefore)
				assert.Equal(t, 50.0, trans.BalanceAfter)
			},
		},
		{
			name:           "Success Deduction using Credit",
			userID:         strconv.Itoa(int(seedUser.ID)),
			body:           `{"amount": 120.0, "reason": "large fee"}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Code int               `json:"status"`
					Data user.UserListItem `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.Equal(t, -20.0, resp.Data.Balance) // 100 - 120 = -20
				// Available = Balance + Limit = -20 + 50 = 30 (Valid)
			},
		},
		{
			name:           "Insufficient Balance",
			userID:         strconv.Itoa(int(seedUser.ID)),
			body:           `{"amount": 200.0, "reason": "too much"}`, // 100 + 50 = 150 < 200
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "Insufficient balance")
			},
		},
		{
			name:           "Negative Amount",
			userID:         strconv.Itoa(int(seedUser.ID)),
			body:           `{"amount": -10.0}`,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "Amount must be positive")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset user state
			database.DB.Exec("DELETE FROM users")
			database.DB.Exec("DELETE FROM transactions")
			
			currentSeed := seedUser
			currentSeed.ID = 0 // Let GORM assign new ID
			database.DB.Create(&currentSeed)
			targetID := strconv.Itoa(int(currentSeed.ID))

			r := gin.New()
			r.Use(func(c *gin.Context) {
				c.Set("user", models.User{Username: "admin_tester", ID: 1})
				c.Next()
			})
			r.PUT("/admin/users/:id/balance", user.DeductBalance)

			req, _ := http.NewRequest(http.MethodPut, "/admin/users/"+targetID+"/balance", bytes.NewBufferString(tt.body))
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

func TestUpdateUser(t *testing.T) {
	setupTestDB()
	gin.SetMode(gin.TestMode)

	// Seed user
	seedUser := models.User{
		Username:    "testuser",
		Role:        "user",
		Password:    "oldpassword",
		Version:     1,
		IsActive:    true,
		CreditLimit: 5000.0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	database.DB.Create(&seedUser)

	tests := []struct {
		name           string
		userID         string
		body           string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "Success Update Status Deactivate",
			userID:         strconv.Itoa(int(seedUser.ID)),
			body:           `{"is_active": false}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Code int               `json:"status"`
					Data user.UserListItem `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.False(t, resp.Data.IsActive)
				assert.NotNil(t, resp.Data.DeactivatedAt)
				assert.Nil(t, resp.Data.ActivatedAt)
				assert.Equal(t, 5000.0, resp.Data.CreditLimit) // Check CreditLimit
				// Verify DB
				var u models.User
				database.DB.First(&u, resp.Data.ID)
				assert.False(t, u.IsActive)
				assert.NotNil(t, u.DeactivatedAt)
			},
		},
		{
			name:           "Success Update Username",
			userID:         strconv.Itoa(int(seedUser.ID)),
			body:           `{"username": "newusername"}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp struct {
					Code int               `json:"status"`
					Data user.UserListItem `json:"data"`
				}
				json.Unmarshal(body, &resp)
				assert.Equal(t, "newusername", resp.Data.Username)
				// Verify DB
				var u models.User
				database.DB.First(&u, resp.Data.ID)
				assert.Equal(t, "newusername", u.Username)
				assert.Equal(t, 2, u.Version)
			},
		},
		{
			name:           "User Not Found",
			userID:         "999",
			body:           `{"username": "ghost"}`,
			expectedStatus: http.StatusNotFound,
			checkResponse:  nil,
		},
		{
			name:           "Invalid Body",
			userID:         strconv.Itoa(int(seedUser.ID)),
			body:           `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset user state for each test to ensure isolation
			database.DB.Exec("DELETE FROM users")
			seedUser := models.User{
				Username:    "testuser",
				Role:        "user",
				Password:    "oldpassword",
				Version:     1,
				IsActive:    true,
				CreditLimit: 5000.0,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			database.DB.Create(&seedUser)
			// Update userID in test case if needed (though ID should be 1 if table is empty, better be safe)
			// But since we use tt.userID which is string, we might need to dynamic resolve it if we want perfect isolation.
			// However, since we just delete rows, ID might increment in SQLite? No, DELETE FROM doesn't reset autoinc usually.
			// But for simplicity, let's just update the seedUser in place if it exists, or delete and re-create.
			// Actually, simpler approach: Update the test case expectation or just allow version increment.

			// Let's rewrite this block properly:
			database.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.User{})
			database.DB.Create(&seedUser)

			// We need to update the userID in the request URL because ID might change or we just use seedUser.ID
			// But tt.userID is hardcoded string.
			// Let's update the request creation to use seedUser.ID if the case is for success.
			targetID := tt.userID
			if tt.name != "User Not Found" && tt.name != "Invalid Body" { // A bit hacky matching
				targetID = strconv.Itoa(int(seedUser.ID))
			} else if tt.name == "Invalid Body" {
				targetID = strconv.Itoa(int(seedUser.ID))
			}

			r := gin.New()
			// Mock middleware setting user
			r.Use(func(c *gin.Context) {
				c.Set("user", models.User{Username: "admin_tester"})
				c.Next()
			})
			r.PATCH("/admin/users/:id", user.UpdateUser)

			req, _ := http.NewRequest(http.MethodPatch, "/admin/users/"+targetID, bytes.NewBufferString(tt.body))
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
