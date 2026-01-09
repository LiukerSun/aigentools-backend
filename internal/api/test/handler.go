package test

import (
	"aigentools-backend/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TestTaskResponse defines the structure of the test task response
type TestTaskResponse struct {
	TaskID string `json:"task_id"`
}

// TestCreateTaskRequest defines the structure for creating a test task
type TestCreateTaskRequest struct {
	Image    string `json:"image" binding:"required"`
	Ratio    string `json:"ratio" binding:"required"`
	Prompt   string `json:"prompt" binding:"required"`
	ModelID  int    `json:"modelId" binding:"required"`
	Duration int    `json:"duration" binding:"required"`
}

// CreateTestTaskHandler godoc
// @Summary      Create a test task
// @Description  Create a test task and get a fixed task ID back.
// @Tags         test
// @Accept       json
// @Produce      json
// @Param        request body TestCreateTaskRequest true "Test task creation request"
// @Success      200 {object} utils.Response{data=TestTaskResponse}
// @Router       /test/task [post]
func CreateTestTaskHandler(c *gin.Context) {
	var req TestCreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse(
		"Test task created successfully",
		TestTaskResponse{TaskID: "1111-2222-3333-4444"},
	))
}
