package payment

import (
	"aigentools-backend/internal/models"
	"aigentools-backend/internal/services"
	"aigentools-backend/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// GetPaymentMethods returns a list of available payment methods
func (h *Handler) GetPaymentMethods(c *gin.Context) {
	methods, err := services.GetPaymentMethods()
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	var response []PaymentMethodResponse
	for _, m := range methods {
		response = append(response, PaymentMethodResponse{
			UUID: m.UUID,
			Type: m.PaymentMethod,
			Name: m.Name,
		})
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("success", response))
}

// CreatePayment initiates a payment
func (h *Handler) CreatePayment(c *gin.Context) {
	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	// Check auth
	userRaw, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, utils.NewErrorResponse(http.StatusUnauthorized, "Unauthorized"))
		return
	}
	user, ok := userRaw.(models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, utils.NewErrorResponse(http.StatusUnauthorized, "Unauthorized"))
		return
	}
	userID := user.ID

	order, err := services.CreatePaymentOrder(userID, req.Amount, req.PaymentMethodUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	// Construct Notify Base URL
	scheme := "http"
	if c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := c.Request.Host
	notifyBaseURL := scheme + "://" + host + "/api/v1/payment/notify"

	jumpURL, err := services.GetPaymentJumpURL(order.ID, req.PaymentMethodUUID, req.PaymentChannel, notifyBaseURL, req.ReturnURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("success", CreatePaymentResponse{
		JumpURL: jumpURL,
		OrderID: order.ID,
	}))
}

// Notify handles the callback
func (h *Handler) Notify(c *gin.Context) {
	uuid := c.Param("uuid")
	if uuid == "" {
		c.String(http.StatusBadRequest, "Missing UUID")
		return
	}

	// Get all query params
	params := make(map[string]interface{})

	// Handle GET params
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	// Handle POST params
	if c.Request.Method == "POST" {
		c.Request.ParseForm()
		for k, v := range c.Request.PostForm {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	}

	err := services.HandlePaymentNotify(uuid, params)
	if err != nil {
		c.String(http.StatusBadRequest, "Fail: "+err.Error())
		return
	}

	c.String(http.StatusOK, "success")
}
