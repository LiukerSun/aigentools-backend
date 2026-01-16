package ai_assistant

import (
	"aigentools-backend/internal/models"
	"aigentools-backend/internal/services"
	"aigentools-backend/internal/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CreateTemplate godoc
// @Summary Create a prompt template
// @Description Create a new personal or public prompt template
// @Tags ai_assistant_templates
// @Accept json
// @Produce json
// @Param request body CreateTemplateRequest true "Create Template Request"
// @Success 200 {object} utils.Response{data=models.PromptTemplate}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /ai-assistant/templates [post]
func CreateTemplate(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, utils.NewErrorResponse(http.StatusUnauthorized, "Unauthorized"))
		return
	}
	user := userVal.(models.User)

	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	// Requirement: "System automatically adds public templates" (implied admin/system action)
	// "User can create personal private templates" (implied user action)
	// Only Admin (Role="admin") can create public templates.
	isPublic := req.IsPublic
	if isPublic && user.Role != "admin" {
		// Return error instead of silently forcing private, to be clearer
		c.JSON(http.StatusForbidden, utils.NewErrorResponse(http.StatusForbidden, "Only administrators can create public templates"))
		return
	}

	template, err := services.CreatePromptTemplate(user.ID, req.Name, req.Description, req.Content, isPublic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Template created successfully", template))
}

// UpdateTemplate godoc
// @Summary Update a prompt template
// @Description Update an existing prompt template
// @Tags ai_assistant_templates
// @Accept json
// @Produce json
// @Param id path int true "Template ID"
// @Param request body UpdateTemplateRequest true "Update Template Request"
// @Success 200 {object} utils.Response{data=models.PromptTemplate}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /ai-assistant/templates/{id} [put]
func UpdateTemplate(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, utils.NewErrorResponse(http.StatusUnauthorized, "Unauthorized"))
		return
	}
	user := userVal.(models.User)

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid ID"))
		return
	}

	var req UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	// Check permission if trying to set public
	if req.IsPublic != nil && *req.IsPublic {
		if user.Role != "admin" {
			c.JSON(http.StatusForbidden, utils.NewErrorResponse(http.StatusForbidden, "Only administrators can set templates to public"))
			return
		}
	}

	template, err := services.UpdatePromptTemplate(uint(id), user.ID, req.Name, req.Description, req.Content, req.IsPublic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Template updated successfully", template))
}

// DeleteTemplate godoc
// @Summary Delete a prompt template
// @Description Delete an existing prompt template
// @Tags ai_assistant_templates
// @Accept json
// @Produce json
// @Param id path int true "Template ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /ai-assistant/templates/{id} [delete]
func DeleteTemplate(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, utils.NewErrorResponse(http.StatusUnauthorized, "Unauthorized"))
		return
	}
	user := userVal.(models.User)

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid ID"))
		return
	}

	if err := services.DeletePromptTemplate(uint(id), user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Template deleted successfully", nil))
}

// ListTemplates godoc
// @Summary List prompt templates
// @Description Get a paginated list of prompt templates
// @Tags ai_assistant_templates
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Page size" default(10)
// @Param type query string false "Filter by type (public/private)"
// @Param search query string false "Search by name or content"
// @Success 200 {object} utils.Response{data=TemplateListResponse}
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /ai-assistant/templates [get]
func ListTemplates(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, utils.NewErrorResponse(http.StatusUnauthorized, "Unauthorized"))
		return
	}
	user := userVal.(models.User)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	filterType := c.Query("type")
	search := c.Query("search")

	templates, total, err := services.ListPromptTemplates(user.ID, page, limit, search, filterType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Success", TemplateListResponse{
		Total: total,
		Items: templates,
	}))
}

// GetTemplate godoc
// @Summary Get a prompt template
// @Description Get a prompt template by ID
// @Tags ai_assistant_templates
// @Accept json
// @Produce json
// @Param id path int true "Template ID"
// @Success 200 {object} utils.Response{data=models.PromptTemplate}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /ai-assistant/templates/{id} [get]
func GetTemplate(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, utils.NewErrorResponse(http.StatusUnauthorized, "Unauthorized"))
		return
	}
	user := userVal.(models.User)

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid ID"))
		return
	}

	template, err := services.GetPromptTemplate(uint(id), user.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, utils.NewErrorResponse(http.StatusNotFound, "Template not found or access denied"))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Success", template))
}
