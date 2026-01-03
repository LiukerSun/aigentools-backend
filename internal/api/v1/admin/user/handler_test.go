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

	// Migrate schema
	err = db.AutoMigrate(&models.User{})
	if err != nil {
		panic("failed to migrate database")
	}

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

func TestUpdateUser(t *testing.T) {
	setupTestDB()
	gin.SetMode(gin.TestMode)

	// Seed user
	seedUser := models.User{
		Username:  "testuser",
		Role:      "user",
		Password:  "oldpassword",
		Version:   1,
		IsActive:  true,
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
				Username:  "testuser",
				Role:      "user",
				Password:  "oldpassword",
				Version:   1,
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
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
