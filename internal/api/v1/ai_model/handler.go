package ai_model

import (
	"aigentools-backend/internal/models"
	"aigentools-backend/internal/services"
	"aigentools-backend/internal/utils"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetModels godoc
// @Summary Get list of AI models
// @Description Retrieve a paginated list of AI models with filtering based on user role
// @Tags models
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Page size" default(10)
// @Param name query string false "Filter by name"
// @Param status query string false "Filter by status"
// @Success 200 {object} utils.Response{data=AIModelListResponse}
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /models [get]
func GetModels(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

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

	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, utils.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}
	user := userVal.(models.User)

	filter := services.AIModelFilter{
		Page:  page,
		Limit: limit,
		Name:  c.Query("name"),
	}

	reqStatus := c.Query("status")

	// Role-based filtering logic
	if user.Role == "admin" {
		filter.Status = reqStatus
	} else {
		// Non-admin users can ONLY see 'open' models
		if reqStatus != "" && reqStatus != string(models.AIModelStatusOpen) {
			c.JSON(http.StatusOK, utils.NewSuccessResponse("Success", AIModelListResponse{
				Models: []AIModelListItem{},
				Total:  0,
				Page:   page,
				Limit:  limit,
			}))
			return
		}
		filter.Status = string(models.AIModelStatusOpen)
	}

	modelsList, total, err := services.FindAIModels(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to fetch models"))
		return
	}

	var responseItems []AIModelListItem
	for _, m := range modelsList {
		responseItems = append(responseItems, AIModelListItem{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			Status:      m.Status,
			CreatedAt:   m.CreatedAt,
			UpdatedAt:   m.UpdatedAt,
		})
	}

	if responseItems == nil {
		responseItems = []AIModelListItem{}
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Success", AIModelListResponse{
		Models: responseItems,
		Total:  total,
		Page:   page,
		Limit:  limit,
	}))
}

// UpdateModelStatus godoc
// @Summary Update AI model status
// @Description Update the status of an AI model. Admin only.
// @Tags models
// @Accept json
// @Produce json
// @Param id path int true "Model ID"
// @Param request body UpdateStatusRequest true "New status"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /models/{id}/status [patch]
func UpdateModelStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid model ID"))
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, utils.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}
	user := userVal.(models.User)

	if user.Role != "admin" {
		c.JSON(http.StatusForbidden, utils.NewErrorResponse(http.StatusForbidden, "Only admin can update model status"))
		return
	}

	// Log sensitive operation
	log.Printf("[SECURITY AUDIT] User %s (ID: %d) is updating model %d status to %s", user.Username, user.ID, id, req.Status)

	if err := services.UpdateModelStatus(uint(id), req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to update status"))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Status updated successfully", nil))
}

// CreateModel godoc
// @Summary Create a new AI model
// @Description Create a new AI model. Admin only.
// @Tags models
// @Accept json
// @Produce json
// @Param request body CreateModelRequest true "Model details"
// @Success 201 {object} utils.Response{data=AIModelListItem}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /models/create [post]
func CreateModel(c *gin.Context) {
	var req CreateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, utils.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
		return
	}
	user := userVal.(models.User)

	// Check if user is admin
	if user.Role != "admin" {
		c.JSON(http.StatusForbidden, utils.NewErrorResponse(http.StatusForbidden, "Only admin can create models"))
		return
	}

	model := models.AIModel{
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
	}

	// Log sensitive operation
	log.Printf("[SECURITY AUDIT] User %s (ID: %d) is creating model %s", user.Username, user.ID, req.Name)

	if err := services.CreateAIModel(&model); err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to create model"))
		return
	}

	responseItem := AIModelListItem{
		ID:          model.ID,
		Name:        model.Name,
		Description: model.Description,
		Status:      model.Status,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}

	c.JSON(http.StatusCreated, utils.NewSuccessResponse("Model created successfully", responseItem))
}
