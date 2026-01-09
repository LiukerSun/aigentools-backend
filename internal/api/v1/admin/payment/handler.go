package payment

import (
	"aigentools-backend/internal/services"
	"aigentools-backend/internal/utils"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// ListPaymentConfigs returns all payment configurations
func (h *Handler) ListPaymentConfigs(c *gin.Context) {
	configs, err := services.GetAllPaymentConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	var response []PaymentConfigResponse
	for _, cfg := range configs {
		var configMap map[string]interface{}
		_ = json.Unmarshal(cfg.Config, &configMap)

		response = append(response, PaymentConfigResponse{
			ID:            cfg.ID,
			UUID:          cfg.UUID,
			Name:          cfg.Name,
			PaymentMethod: cfg.PaymentMethod,
			Config:        configMap,
			Enable:        cfg.Enable,
			CreatedAt:     cfg.CreatedAt.Format(time.RFC3339),
			UpdatedAt:     cfg.UpdatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("success", response))
}

// CreatePaymentConfig creates a new payment configuration
func (h *Handler) CreatePaymentConfig(c *gin.Context) {
	var req CreatePaymentConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	cfg, err := services.CreatePaymentConfig(req.Name, req.PaymentMethod, req.Config, req.Enable)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("success", gin.H{"id": cfg.ID, "uuid": cfg.UUID}))
}

// UpdatePaymentConfig updates an existing payment configuration
func (h *Handler) UpdatePaymentConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid ID"))
		return
	}

	var req UpdatePaymentConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	_, err = services.UpdatePaymentConfig(uint(id), req.Name, req.Config, req.Enable)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("success", nil))
}

// DeletePaymentConfig deletes a payment configuration
func (h *Handler) DeletePaymentConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid ID"))
		return
	}

	if err := services.DeletePaymentConfig(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("success", nil))
}
