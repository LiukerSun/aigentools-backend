package user

import (
	"aigentools-backend/internal/models"
	"aigentools-backend/internal/services"
	"aigentools-backend/internal/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type UserListItem struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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

	users, total, err := services.FindUsers(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to fetch users"))
		return
	}

	var userItems []UserListItem
	for _, u := range users {
		userItems = append(userItems, UserListItem{
			ID:        u.ID,
			Username:  u.Username,
			Role:      u.Role,
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
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
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty" binding:"omitempty,min=6"`
	Role     *string `json:"role,omitempty" binding:"omitempty,oneof=admin user"`
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
		ID:        updatedUser.ID,
		Username:  updatedUser.Username,
		Role:      updatedUser.Role,
		CreatedAt: updatedUser.CreatedAt,
		UpdatedAt: updatedUser.UpdatedAt,
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("User updated successfully", response))
}
