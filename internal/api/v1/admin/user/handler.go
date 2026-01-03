package user

import (
	"aigentools-backend/internal/models"
	"aigentools-backend/internal/services"
	"aigentools-backend/internal/utils"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type UserListItem struct {
	ID            uint       `json:"id"`
	Username      string     `json:"username"`
	Role          string     `json:"role"`
	IsActive      bool       `json:"is_active"`
	ActivatedAt   *time.Time `json:"activated_at,omitempty"`
	DeactivatedAt *time.Time `json:"deactivated_at,omitempty"`
	Balance       float64    `json:"balance"`
	CreditLimit   float64    `json:"creditLimit"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type UserListResponse struct {
	Users []UserListItem `json:"users"`
	Total int64          `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}

// ListUsers godoc
// @Summary List all users
// @Description Get a paginated list of users. Admin only.
// @Tags admin
// @Produce json
// @Security Bearer
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param is_active query bool false "Filter by active status"
// @Param created_after query string false "Filter by creation time (start) - RFC3339"
// @Param created_before query string false "Filter by creation time (end) - RFC3339"
// @Success 200 {object} utils.Response{data=UserListResponse}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /admin/users [get]
func ListUsers(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid page number"))
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid limit number"))
		return
	}

	filter := services.UserFilter{
		Page:  page,
		Limit: limit,
	}

	if isActiveStr, exists := c.GetQuery("is_active"); exists {
		isActive, err := strconv.ParseBool(isActiveStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid is_active parameter"))
			return
		}
		filter.IsActive = &isActive
	}

	if createdAfterStr, exists := c.GetQuery("created_after"); exists {
		createdAfter, err := time.Parse(time.RFC3339, createdAfterStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid created_after parameter format"))
			return
		}
		filter.CreatedAfter = &createdAfter
	}

	if createdBeforeStr, exists := c.GetQuery("created_before"); exists {
		createdBefore, err := time.Parse(time.RFC3339, createdBeforeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid created_before parameter format"))
			return
		}
		filter.CreatedBefore = &createdBefore
	}

	users, total, err := services.FindUsers(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to fetch users"))
		return
	}

	var userItems []UserListItem
	for _, u := range users {
		userItems = append(userItems, UserListItem{
			ID:            u.ID,
			Username:      u.Username,
			Role:          u.Role,
			IsActive:      u.IsActive,
			ActivatedAt:   u.ActivatedAt,
			DeactivatedAt: u.DeactivatedAt,
			Balance:       u.Balance,
			CreditLimit:   u.CreditLimit,
			CreatedAt:     u.CreatedAt,
			UpdatedAt:     u.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Users retrieved successfully", UserListResponse{
		Users: userItems,
		Total: total,
		Page:  page,
		Limit: limit,
	}))
}

// UpdateUserRequest represents the request body for updating a user
type UpdateUserRequest struct {
	Username    *string  `json:"username,omitempty"`
	Password    *string  `json:"password,omitempty" binding:"omitempty,min=6"`
	Role        *string  `json:"role,omitempty" binding:"omitempty,oneof=admin user"`
	IsActive    *bool    `json:"is_active,omitempty"`
	CreditLimit *float64 `json:"creditLimit,omitempty"`
}

// UpdateUser godoc
// @Summary Update a user
// @Description Update user details. Admin only.
// @Tags admin
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "User ID"
// @Param body body UpdateUserRequest true "User details to update"
// @Success 200 {object} utils.Response{data=UserListItem}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 409 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /admin/users/{id} [patch]
func UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid user ID"))
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	updates := make(map[string]interface{})
	if req.Username != nil {
		updates["username"] = *req.Username
	}
	if req.Password != nil {
		updates["password"] = *req.Password
	}
	if req.Role != nil {
		updates["role"] = *req.Role
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.CreditLimit != nil {
		updates["credit_limit"] = *req.CreditLimit
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "No fields to update"))
		return
	}

	operator := "unknown"
	if userVal, exists := c.Get("user"); exists {
		if u, ok := userVal.(models.User); ok {
			operator = u.Username
		}
	}

	updatedUser, err := services.UpdateUser(uint(id), updates, operator)
	if err != nil {
		if err == services.ErrUserNotFound {
			c.JSON(http.StatusNotFound, utils.NewErrorResponse(http.StatusNotFound, "User not found"))
			return
		}
		if err == services.ErrOptimisticLock {
			c.JSON(http.StatusConflict, utils.NewErrorResponse(http.StatusConflict, err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to update user"))
		return
	}

	response := UserListItem{
		ID:            updatedUser.ID,
		Username:      updatedUser.Username,
		Role:          updatedUser.Role,
		IsActive:      updatedUser.IsActive,
		ActivatedAt:   updatedUser.ActivatedAt,
		DeactivatedAt: updatedUser.DeactivatedAt,
		Balance:       updatedUser.Balance,
		CreditLimit:   updatedUser.CreditLimit,
		CreatedAt:     updatedUser.CreatedAt,
		UpdatedAt:     updatedUser.UpdatedAt,
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("User updated successfully", response))
}

// BalanceAdjustmentRequest represents the request body for adjusting user balance
type BalanceAdjustmentRequest struct {
	Amount float64 `json:"amount" binding:"required"`
	Reason string  `json:"reason"` // Optional as per requirement "reason: 字符串类型，记录扣减原因（可选）"
}

// AdjustBalance godoc
// @Summary Adjust user balance
// @Description Increase or decrease user balance. Admin only.
// @Tags admin
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "User ID"
// @Param body body BalanceAdjustmentRequest true "Balance adjustment details"
// @Success 200 {object} utils.Response{data=UserListItem}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 409 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /admin/users/{id}/balance [post]
func AdjustBalance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid user ID"))
		return
	}

	var req BalanceAdjustmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	operator := "unknown"
	var operatorID uint
	if userVal, exists := c.Get("user"); exists {
		if u, ok := userVal.(models.User); ok {
			operator = u.Username
			operatorID = u.ID
		}
	}

	meta := services.TransactionMetadata{
		Operator:   operator,
		OperatorID: operatorID,
		Type:       models.TransactionTypeSystemAdmin,
		IPAddress:  c.ClientIP(),
		DeviceInfo: c.GetHeader("User-Agent"),
	}

	updatedUser, err := services.AdjustBalance(uint(id), req.Amount, req.Reason, meta)
	if err != nil {
		if err == services.ErrUserNotFound {
			c.JSON(http.StatusNotFound, utils.NewErrorResponse(http.StatusNotFound, "User not found"))
			return
		}
		if err == services.ErrOptimisticLock {
			c.JSON(http.StatusConflict, utils.NewErrorResponse(http.StatusConflict, err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to adjust balance: %v", err)))
		return
	}

	response := UserListItem{
		ID:            updatedUser.ID,
		Username:      updatedUser.Username,
		Role:          updatedUser.Role,
		IsActive:      updatedUser.IsActive,
		ActivatedAt:   updatedUser.ActivatedAt,
		DeactivatedAt: updatedUser.DeactivatedAt,
		Balance:       updatedUser.Balance,
		CreatedAt:     updatedUser.CreatedAt,
		UpdatedAt:     updatedUser.UpdatedAt,
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Balance adjusted successfully", response))
}

// DeductBalance godoc
// @Summary Deduct user balance
// @Description Deduct amount from user's balance. Admin only.
// @Tags admin
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "User ID"
// @Param request body user.BalanceAdjustmentRequest true "Deduction details"
// @Success 200 {object} utils.Response{data=user.UserListItem}
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 409 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /admin/users/{id}/balance [put]
func DeductBalance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid user ID"))
		return
	}

	var req BalanceAdjustmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Amount must be positive"))
		return
	}

	operator := "unknown"
	var operatorID uint
	if userVal, exists := c.Get("user"); exists {
		if u, ok := userVal.(models.User); ok {
			operator = u.Username
			operatorID = u.ID
		}
	}

	meta := services.TransactionMetadata{
		Operator:   operator,
		OperatorID: operatorID,
		Type:       models.TransactionTypeSystemAdmin, // Or maybe a new type like AdminDeduction? Using AdminAdjustment for now.
		IPAddress:  c.ClientIP(),
		DeviceInfo: c.GetHeader("User-Agent"),
	}

	updatedUser, err := services.DeductBalance(uint(id), req.Amount, req.Reason, meta)
	if err != nil {
		if err == services.ErrUserNotFound {
			c.JSON(http.StatusNotFound, utils.NewErrorResponse(http.StatusNotFound, "User not found"))
			return
		}
		if err == services.ErrInsufficientBalance {
			c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Insufficient balance"))
			return
		}
		if err == services.ErrOptimisticLock {
			c.JSON(http.StatusConflict, utils.NewErrorResponse(http.StatusConflict, err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to deduct balance: %v", err)))
		return
	}

	response := UserListItem{
		ID:            updatedUser.ID,
		Username:      updatedUser.Username,
		Role:          updatedUser.Role,
		IsActive:      updatedUser.IsActive,
		ActivatedAt:   updatedUser.ActivatedAt,
		DeactivatedAt: updatedUser.DeactivatedAt,
		Balance:       updatedUser.Balance,
		CreditLimit:   updatedUser.CreditLimit,
		CreatedAt:     updatedUser.CreatedAt,
		UpdatedAt:     updatedUser.UpdatedAt,
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Balance deducted successfully", response))
}
