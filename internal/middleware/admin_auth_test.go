package middleware

import (
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/utils"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

// Mock config for testing token generation
func setupTestConfig() {
	os.Setenv("JWT_SECRET", "test_secret")
	os.Setenv("DB_HOST", "localhost")
}

func setupMockRedis() *miniredis.Miniredis {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}

	database.RedisClient = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return mr
}

func TestAdminAuthMiddleware(t *testing.T) {
	setupTestConfig()
	mr := setupMockRedis()
	defer mr.Close()

	gin.SetMode(gin.TestMode)

	// Helper to generate test tokens
	generateTestToken := func(role string, expired bool) string {
		claims := jwt.MapClaims{
			"user_id": 1,
			"role":    role,
			"exp":     time.Now().Add(time.Hour).Unix(),
		}
		if expired {
			claims["exp"] = time.Now().Add(-time.Hour).Unix()
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tString, _ := token.SignedString([]byte("test_secret"))
		return tString
	}

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Missing Authorization Header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "authorization header is required",
		},
		{
			name:           "Invalid Token Format",
			authHeader:     "InvalidToken",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "bearer token not found",
		},
		{
			name:           "Invalid Token Signature",
			authHeader:     "Bearer invalid.token.signature",
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Invalid or expired token",
		},
		{
			name:           "Non-Admin User",
			authHeader:     "Bearer " + generateTestToken("user", false),
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Forbidden: Admins only",
		},
		{
			name:           "Admin User",
			authHeader:     "Bearer " + generateTestToken("admin", false),
			expectedStatus: http.StatusOK,
			expectedBody:   "Success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(AdminAuthMiddleware())
			r.GET("/admin/test", func(c *gin.Context) {
				c.String(http.StatusOK, "Success")
			})

			req, _ := http.NewRequest(http.MethodGet, "/admin/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus != http.StatusOK {
				var resp utils.Response
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Contains(t, resp.Message, tt.expectedBody)
			} else {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}
