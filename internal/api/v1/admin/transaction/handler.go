package transaction

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

// ListTransactions godoc
// @Summary List transactions
// @Description Get a paginated list of transactions with filtering. Admin only.
// @Tags admin
// @Produce json
// @Security Bearer
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param user_id query int false "Filter by user ID"
// @Param type query string false "Filter by transaction type"
// @Param start_time query string false "Filter by start time (RFC3339)"
// @Param end_time query string false "Filter by end time (RFC3339)"
// @Param min_amount query number false "Filter by minimum amount"
// @Param max_amount query number false "Filter by maximum amount"
// @Success 200 {object} utils.Response{data=TransactionListResponse}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /admin/transactions [get]
func ListTransactions(c *gin.Context) {
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

	filter := services.TransactionFilter{
		Page:  page,
		Limit: limit,
	}

	if userIDStr, exists := c.GetQuery("user_id"); exists {
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid user_id"))
			return
		}
		uid := uint(userID)
		filter.UserID = &uid
	}

	if typeStr, exists := c.GetQuery("type"); exists {
		t := models.TransactionType(typeStr)
		filter.Type = &t
	}

	if startTimeStr, exists := c.GetQuery("start_time"); exists {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid start_time format"))
			return
		}
		filter.StartTime = &startTime
	}

	if endTimeStr, exists := c.GetQuery("end_time"); exists {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid end_time format"))
			return
		}
		filter.EndTime = &endTime
	}

	if minAmountStr, exists := c.GetQuery("min_amount"); exists {
		minAmount, err := strconv.ParseFloat(minAmountStr, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid min_amount"))
			return
		}
		filter.MinAmount = &minAmount
	}

	if maxAmountStr, exists := c.GetQuery("max_amount"); exists {
		maxAmount, err := strconv.ParseFloat(maxAmountStr, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid max_amount"))
			return
		}
		filter.MaxAmount = &maxAmount
	}

	transactions, total, err := services.FindTransactions(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to fetch transactions"))
		return
	}

	var items []TransactionListItem
	for _, t := range transactions {
		items = append(items, TransactionListItem{
			ID:            t.ID,
			CreatedAt:     t.CreatedAt,
			UserID:        t.UserID,
			Amount:        t.Amount,
			BalanceBefore: t.BalanceBefore,
			BalanceAfter:  t.BalanceAfter,
			Reason:        t.Reason,
			Operator:      t.Operator,
			Type:          t.Type,
			IPAddress:     t.IPAddress,
			DeviceInfo:    t.DeviceInfo,
			Hash:          t.Hash,
		})
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Transactions retrieved successfully", TransactionListResponse{
		Transactions: items,
		Total:        total,
		Page:         page,
		Limit:        limit,
	}))
}

// ExportTransactions godoc
// @Summary Export transactions
// @Description Export transactions to CSV. Admin only.
// @Tags admin
// @Produce text/csv
// @Security Bearer
// @Param user_id query int false "Filter by user ID"
// @Param type query string false "Filter by transaction type"
// @Param start_time query string false "Filter by start time (RFC3339)"
// @Param end_time query string false "Filter by end time (RFC3339)"
// @Success 200 {string} string "CSV content"
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /admin/transactions/export [get]
func ExportTransactions(c *gin.Context) {
	// Re-use filter logic (simplified for brevity, ideally shared)
	// For export, we might want higher limit or all records.
	// Let's assume export all matching records (limit=0/unlimited in service if we implemented that, or high number)

	filter := services.TransactionFilter{
		Page:  1,
		Limit: 10000, // Hard limit for safety
	}

	if userIDStr, exists := c.GetQuery("user_id"); exists {
		userID, err := strconv.Atoi(userIDStr)
		if err == nil {
			uid := uint(userID)
			filter.UserID = &uid
		}
	}
	if typeStr, exists := c.GetQuery("type"); exists {
		t := models.TransactionType(typeStr)
		filter.Type = &t
	}
	if startTimeStr, exists := c.GetQuery("start_time"); exists {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartTime = &startTime
		}
	}
	if endTimeStr, exists := c.GetQuery("end_time"); exists {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndTime = &endTime
		}
	}

	transactions, _, err := services.FindTransactions(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to fetch transactions"))
		return
	}

	csvContent, err := services.GenerateTransactionCSV(transactions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to generate CSV"))
		return
	}

	filename := fmt.Sprintf("transactions_%s.csv", time.Now().Format("20060102150405"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "text/csv", csvContent)
}
