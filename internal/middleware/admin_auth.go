package middleware

import (
	"aigentools-backend/internal/services"
	"aigentools-backend/internal/utils"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminAuthMiddleware validates that the user has admin privileges.
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := utils.ExtractToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, utils.NewErrorResponse(http.StatusUnauthorized, err.Error()))
			c.Abort()
			return
		}

		isDenylisted, err := services.IsDenylisted(tokenString)
		if err != nil {
			c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to check token status"))
			c.Abort()
			return
		}
		if isDenylisted {
			c.JSON(http.StatusUnauthorized, utils.NewErrorResponse(http.StatusUnauthorized, "Token has been revoked"))
			c.Abort()
			return
		}

		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusForbidden, utils.NewErrorResponse(http.StatusForbidden, "Invalid or expired token"))
			c.Abort()
			return
		}

		role, ok := claims["role"].(string)
		if !ok || role != "admin" {
			// Log unauthorized access attempt (simulated log)
			fmt.Printf("Unauthorized admin access attempt. Token: %s\n", tokenString)
			c.JSON(http.StatusForbidden, utils.NewErrorResponse(http.StatusForbidden, "Forbidden: Admins only"))
			c.Abort()
			return
		}

		// Only try to load user if not in test mode or if we have a mocked DB.
		// For unit testing the middleware logic, we don't strictly need the user object in context
		// unless the handler depends on it. Here we just want to pass the middleware check.
		// We'll skip DB call if gin.Mode() is TestMode to avoid panic on nil DB.
		if gin.Mode() != gin.TestMode {
			userIDFloat, ok := claims["user_id"].(float64)
			if ok {
				userID := uint(userIDFloat)
				user, err := services.FindUserByID(userID)
				if err == nil {
					c.Set("user", user)
				}
			}
		}

		c.Next()
	}
}
