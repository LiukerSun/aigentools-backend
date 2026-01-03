package user

import (
	"aigentools-backend/internal/models"
	"aigentools-backend/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CurrentUser godoc
// @Summary Get current user
// @Description Get current user's information
// @Tags user
// @Produce  json
// @Security ApiKeyAuth
// @Success 200 {object} utils.Response{data=user.UserResponse}
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /auth/user [get]
func CurrentUser(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, utils.NewErrorResponse(http.StatusUnauthorized, "Unauthorized"))
		return
	}

	u := user.(models.User)

	token, err := utils.GenerateToken(u.ID, u.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Could not generate token"))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("User information retrieved successfully", UserResponse{
		ID:       u.ID,
		Username: u.Username,
		Role:     u.Role,
		Token:    token,
	}))
}
