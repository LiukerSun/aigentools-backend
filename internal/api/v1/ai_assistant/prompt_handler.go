package ai_assistant

import (
	"aigentools-backend/internal/services"
	"aigentools-backend/internal/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CreatePrompt godoc
// @Summary Create a new prompt
// @Description Create a new system prompt with a unique code
// @Tags ai_assistant_prompts
// @Accept json
// @Produce json
// @Param request body CreatePromptRequest true "Create Prompt Request"
// @Success 200 {object} utils.Response{data=models.Prompt}
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /ai-assistant/prompts [post]
func CreatePrompt(c *gin.Context) {
	var req CreatePromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	prompt, err := services.CreatePrompt(req.Code, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Prompt created successfully", prompt))
}

// BatchCreatePrompts godoc
// @Summary Batch create prompts
// @Description Create multiple system prompts
// @Tags ai_assistant_prompts
// @Accept json
// @Produce json
// @Param request body BatchCreatePromptRequest true "Batch Create Prompt Request"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /ai-assistant/prompts/batch [post]
func BatchCreatePrompts(c *gin.Context) {
	var req BatchCreatePromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	// Convert DTO to anonymous struct for service
	var serviceReqs []struct{ Code, Content string }
	for _, p := range req.Prompts {
		serviceReqs = append(serviceReqs, struct{ Code, Content string }{Code: p.Code, Content: p.Content})
	}

	if err := services.BatchCreatePrompts(serviceReqs); err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Prompts created successfully", nil))
}

// UpdatePrompt godoc
// @Summary Update an existing prompt
// @Description Update the content of a system prompt by code
// @Tags ai_assistant_prompts
// @Accept json
// @Produce json
// @Param code path string true "Prompt Code"
// @Param request body UpdatePromptRequest true "Update Prompt Request"
// @Success 200 {object} utils.Response{data=models.Prompt}
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /ai-assistant/prompts/{code} [put]
func UpdatePrompt(c *gin.Context) {
	code := c.Param("code")
	var req UpdatePromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	prompt, err := services.UpdatePrompt(code, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Prompt updated successfully", prompt))
}

// DeletePrompt godoc
// @Summary Delete a prompt
// @Description Delete a system prompt by code
// @Tags ai_assistant_prompts
// @Accept json
// @Produce json
// @Param code path string true "Prompt Code"
// @Success 200 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /ai-assistant/prompts/{code} [delete]
func DeletePrompt(c *gin.Context) {
	code := c.Param("code")
	if err := services.DeletePrompt(code); err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Prompt deleted successfully", nil))
}

// GetPrompt godoc
// @Summary Get a prompt
// @Description Get a system prompt by code
// @Tags ai_assistant_prompts
// @Accept json
// @Produce json
// @Param code path string true "Prompt Code"
// @Success 200 {object} utils.Response{data=models.Prompt}
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /ai-assistant/prompts/{code} [get]
func GetPrompt(c *gin.Context) {
	code := c.Param("code")
	prompt, err := services.GetPromptByCode(code)
	if err != nil {
		c.JSON(http.StatusNotFound, utils.NewErrorResponse(http.StatusNotFound, "Prompt not found"))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Success", prompt))
}

// ListPrompts godoc
// @Summary List prompts
// @Description Get a paginated list of system prompts
// @Tags ai_assistant_prompts
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Page size" default(10)
// @Success 200 {object} utils.Response{data=PromptListResponse}
// @Failure 500 {object} utils.Response
// @Router /ai-assistant/prompts [get]
func ListPrompts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	prompts, total, err := services.ListPrompts(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Success", PromptListResponse{
		Total: total,
		Items: prompts,
	}))
}
