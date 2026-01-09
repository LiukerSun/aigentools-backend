package ai_model

import (
	"aigentools-backend/internal/models"
	"aigentools-backend/internal/services"
	"aigentools-backend/internal/utils"
	"aigentools-backend/pkg/logger"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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
			URL:         m.URL,
			Price:       m.Price,
			Parameters:  m.Parameters,
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

	logger.Log.Info("Received request to create model", zap.String("user", user.Username), zap.String("model_name", req.Name))

	model := models.AIModel{
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
		URL:         req.URL,
		Price:       req.Price,
		Parameters:  req.Parameters,
	}

	if model.Parameters == nil {
		model.Parameters = models.JSON{
			"request_header":      []interface{}{},
			"request_body":        []interface{}{},
			"response_parameters": []interface{}{},
		}
	}
	logger.Log.Info("Model parameters validated successfully", zap.Any("parameters", model.Parameters))
	if err := models.ValidateModelParameters(model.Parameters); err != nil {
		// Log the validation error
		logger.Log.Warn("Model parameter validation failed", zap.Error(err), zap.Any("parameters", model.Parameters))
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid parameters: "+err.Error()))
		return
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
		URL:         model.URL,
		Price:       model.Price,
		Parameters:  model.Parameters,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}

	c.JSON(http.StatusCreated, utils.NewSuccessResponse("Model created successfully", responseItem))
}

// GetModelNames godoc
// @Summary Get list of all AI models (simplified)
// @Description Retrieve a list of all AI models with basic details (excluding parameters).
// @Tags models
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} utils.Response{data=[]AIModelSimpleItem}
// @Failure 500 {object} utils.Response
// @Router /models/names [get]
func GetModelNames(c *gin.Context) {
	modelsList, err := services.GetAllModelsSimple()
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to fetch models"))
		return
	}

	var responseItems []AIModelSimpleItem
	for _, m := range modelsList {
		responseItems = append(responseItems, AIModelSimpleItem{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			Status:      m.Status,
			URL:         m.URL,
			CreatedAt:   m.CreatedAt,
			UpdatedAt:   m.UpdatedAt,
		})
	}
	// Ensure non-nil slice for empty result
	if responseItems == nil {
		responseItems = []AIModelSimpleItem{}
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Success", responseItems))
}

// GetModelParameters godoc
// @Summary Get AI model parameters by ID
// @Description Retrieve the parameters of an AI model by its ID.
// @Tags models
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Model ID"
// @Success 200 {object} utils.Response{data=models.JSON}
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /models/{id}/parameters [get]
func GetModelParameters(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid model ID"))
		return
	}

	params, err := services.GetModelParametersByID(uint(id))
	if err != nil {
		// Check if it's a "record not found" error
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, utils.NewErrorResponse(http.StatusNotFound, "Model not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to fetch model parameters"))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Success", params))
}

// UpdateModel godoc
// @Summary Update an existing AI model
// @Description Update AI model details. Admin only.
// @Tags models
// @Accept json
// @Produce json
// @Param id path int true "Model ID"
// @Param request body UpdateModelRequest true "Model details"
// @Success 200 {object} utils.Response{data=AIModelListItem}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /models/{id} [put]
func UpdateModel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid model ID"))
		return
	}

	var req UpdateModelRequest
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
		c.JSON(http.StatusForbidden, utils.NewErrorResponse(http.StatusForbidden, "Only admin can update models"))
		return
	}

	model, err := services.GetAIModelByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, utils.NewErrorResponse(http.StatusNotFound, "Model not found"))
		return
	}

	// Update fields if provided
	if req.Name != "" {
		model.Name = req.Name
	}
	if req.Description != "" {
		model.Description = req.Description
	}
	if req.Status != "" {
		model.Status = req.Status
	}
	if req.URL != "" {
		model.URL = req.URL
	}
	if req.Price != nil {
		model.Price = *req.Price
	}
	if req.Parameters != nil {
		model.Parameters = req.Parameters
		if err := models.ValidateModelParameters(model.Parameters); err != nil {
			c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid parameters: "+err.Error()))
			return
		}
	}

	log.Printf("[SECURITY AUDIT] User %s (ID: %d) is updating model %d", user.Username, user.ID, id)

	if err := services.UpdateAIModel(model); err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to update model"))
		return
	}

	responseItem := AIModelListItem{
		ID:          model.ID,
		Name:        model.Name,
		Description: model.Description,
		Status:      model.Status,
		URL:         model.URL,
		Parameters:  model.Parameters,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Model updated successfully", responseItem))
}
