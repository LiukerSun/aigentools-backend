package order

import (
	"aigentools-backend/internal/models"
	"aigentools-backend/internal/services"
	"aigentools-backend/internal/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// ListOrders 获取订单列表
func (h *Handler) ListOrders(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	filter := services.OrderFilter{
		Page:  page,
		Limit: limit,
	}

	// 解析筛选参数
	if userIDStr, exists := c.GetQuery("user_id"); exists {
		userID, _ := strconv.Atoi(userIDStr)
		uid := uint(userID)
		filter.UserID = &uid
	}
	if status, exists := c.GetQuery("status"); exists {
		filter.Status = &status
	}
	if orderType, exists := c.GetQuery("order_type"); exists {
		filter.OrderType = &orderType
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
	if minAmountStr, exists := c.GetQuery("min_amount"); exists {
		if minAmount, err := strconv.ParseFloat(minAmountStr, 64); err == nil {
			filter.MinAmount = &minAmount
		}
	}
	if maxAmountStr, exists := c.GetQuery("max_amount"); exists {
		if maxAmount, err := strconv.ParseFloat(maxAmountStr, 64); err == nil {
			filter.MaxAmount = &maxAmount
		}
	}

	orders, total, err := services.FindOrders(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	var items []OrderListItem
	for _, o := range orders {
		items = append(items, OrderListItem{
			ID:          o.ID,
			UserID:      o.UserID,
			Amount:      o.Amount,
			Status:      o.Status,
			OrderType:   o.OrderType,
			PaymentUUID: o.PaymentUUID,
			ExternalID:  o.ExternalID,
			Remark:      o.Remark,
			CompletedAt: o.CompletedAt,
			CompletedBy: o.CompletedBy,
			CreatedAt:   o.CreatedAt,
			UpdatedAt:   o.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("success", OrderListResponse{
		Orders: items,
		Total:  total,
		Page:   page,
		Limit:  limit,
	}))
}

// GetOrder 获取订单详情
func (h *Handler) GetOrder(c *gin.Context) {
	orderID := c.Param("id")

	order, err := services.GetOrderByID(orderID)
	if err != nil {
		if err == services.ErrOrderNotFound {
			c.JSON(http.StatusNotFound, utils.NewErrorResponse(http.StatusNotFound, "Order not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	response := OrderDetailResponse{
		OrderListItem: OrderListItem{
			ID:          order.ID,
			UserID:      order.UserID,
			Amount:      order.Amount,
			Status:      order.Status,
			OrderType:   order.OrderType,
			PaymentUUID: order.PaymentUUID,
			ExternalID:  order.ExternalID,
			Remark:      order.Remark,
			CompletedAt: order.CompletedAt,
			CompletedBy: order.CompletedBy,
			CreatedAt:   order.CreatedAt,
			UpdatedAt:   order.UpdatedAt,
		},
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("success", response))
}

// CreateOrder 创建手动订单
func (h *Handler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	order, err := services.CreateManualOrder(req.UserID, req.Amount, req.Remark)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Order created successfully", gin.H{
		"id":     order.ID,
		"status": order.Status,
	}))
}

// CompleteOrder 完成订单
func (h *Handler) CompleteOrder(c *gin.Context) {
	orderID := c.Param("id")

	// 获取操作者信息
	operator := "system"
	var operatorID uint
	if userVal, exists := c.Get("user"); exists {
		if u, ok := userVal.(models.User); ok {
			operator = u.Username
			operatorID = u.ID
		}
	}

	err := services.CompleteOrder(orderID, operatorID, operator)
	if err != nil {
		switch err {
		case services.ErrOrderNotFound:
			c.JSON(http.StatusNotFound, utils.NewErrorResponse(http.StatusNotFound, "Order not found"))
		case services.ErrOrderAlreadyPaid:
			c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Order already paid"))
		case services.ErrOrderCancelled:
			c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Order has been cancelled"))
		default:
			c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Order completed successfully", nil))
}

// CancelOrder 取消订单
func (h *Handler) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")

	err := services.CancelOrder(orderID)
	if err != nil {
		switch err {
		case services.ErrOrderNotFound:
			c.JSON(http.StatusNotFound, utils.NewErrorResponse(http.StatusNotFound, "Order not found"))
		case services.ErrInvalidOrderStatus:
			c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Only pending orders can be cancelled"))
		default:
			c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Order cancelled successfully", nil))
}
