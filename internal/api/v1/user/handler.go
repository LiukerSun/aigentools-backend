package user

import (
	"aigentools-backend/internal/database"
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
// @Security Bearer
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

	// Force reload user from DB to ensure we have the latest balance/stats
	// ignoring the cached version from middleware
	var latestUser models.User
	if err := database.DB.First(&latestUser, u.ID).Error; err == nil {
		u = latestUser
	}

	token, err := utils.GenerateToken(u.ID, u.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Could not generate token"))
		return
	}

	// Calculate credit info based on "Total = Balance + CreditLimit" model
	// This supports both prepaid (Balance > 0) and postpaid/overdraft (Balance < 0) scenarios.

	var total, available, used, usagePercentage float64

	// Available = Balance + CreditLimit
	// This represents the actual purchasing power.
	available = u.Balance + u.CreditLimit

	// Total = Max(Balance, 0) + CreditLimit
	// This represents the total capacity (Own Funds + Credit Line).
	if u.Balance > 0 {
		total = u.Balance + u.CreditLimit
	} else {
		total = u.CreditLimit
	}

	// Used = Total - Available
	used = total - available

	if total > 0 {
		usagePercentage = (used / total) * 100
	}

	creditInfo := &CreditInfo{
		Total:           total,
		Available:       available,
		Used:            used,
		UsagePercentage: usagePercentage,
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("User information retrieved successfully", UserResponse{
		ID:            u.ID,
		Username:      u.Username,
		Role:          u.Role,
		IsActive:      u.IsActive,
		ActivatedAt:   u.ActivatedAt,
		DeactivatedAt: u.DeactivatedAt,
		CreditLimit:   u.CreditLimit,
		TotalConsumed: u.TotalConsumed,
		Credit:        creditInfo,
		Token:         token,
	}))
}
