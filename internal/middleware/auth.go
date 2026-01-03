package middleware

import (
	"aigentools-backend/internal/services"
	"aigentools-backend/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
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
			c.JSON(http.StatusUnauthorized, utils.NewErrorResponse(http.StatusUnauthorized, "Invalid or expired token"))
			c.Abort()
			return
		}

		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, utils.NewErrorResponse(http.StatusUnauthorized, "Invalid user ID in token"))
			c.Abort()
			return
		}
		userID := uint(userIDFloat)

		user, err := services.FindUserByID(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, utils.NewErrorResponse(http.StatusUnauthorized, "User not found"))
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	}
}
