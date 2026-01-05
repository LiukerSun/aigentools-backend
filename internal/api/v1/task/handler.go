package task

import (
	"aigentools-backend/internal/models"
	"aigentools-backend/internal/services"
	"aigentools-backend/internal/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// SubmitTask godoc
// @Summary Submit a new task
// @Description Submit a new task with body and user information
// @Tags tasks
// @Accept json
// @Produce json
// @Param request body CreateTaskRequest true "Task creation request"
// @Success 200 {object} utils.Response{data=models.Task}
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /tasks [post]
func SubmitTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	var task *models.Task
	var err error
	task, err = services.CreateTask(req.Body, req.User.CreatorID, req.User.CreatorName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Task submitted successfully", task))
}

// ApproveTask godoc
// @Summary Approve a task
// @Description Approve a pending audit task and push it to the execution queue
// @Tags tasks
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {object} utils.Response{data=models.Task}
// @Failure 400 {object} utils.Response
// @Router /tasks/{id}/approve [patch]
func ApproveTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid task ID"))
		return
	}

	task, err := services.ApproveTask(uint(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Task approved successfully", task))
}

// UpdateTask godoc
// @Summary Update task parameters
// @Description Update task input data (only if not yet processing)
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path int true "Task ID"
// @Param request body UpdateTaskRequest true "Task update request"
// @Success 200 {object} utils.Response{data=models.Task}
// @Failure 400 {object} utils.Response
// @Router /tasks/{id} [put]
func UpdateTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid task ID"))
		return
	}

	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	task, err := services.UpdateTask(uint(id), req.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Task updated successfully", task))
}

// ListTasks godoc
// @Summary List tasks
// @Description List tasks with pagination and filtering
// @Tags tasks
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Page size (default 10)"
// @Param creator_id query int false "Creator ID"
// @Param status query int false "Task Status"
// @Success 200 {object} utils.Response{data=TaskListResponse}
// @Failure 500 {object} utils.Response
// @Router /tasks [get]
func ListTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	creatorID, _ := strconv.Atoi(c.Query("creator_id"))
	
	var status *models.TaskStatus
	if statusStr := c.Query("status"); statusStr != "" {
		s, err := strconv.Atoi(statusStr)
		if err == nil {
			ts := models.TaskStatus(s)
			status = &ts
		}
	}

	tasks, total, err := services.GetTasks(page, pageSize, uint(creatorID), status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Tasks retrieved successfully", TaskListResponse{
		Total: total,
		Items: tasks,
	}))
}

// GetTaskDetail godoc
// @Summary Get task detail
// @Description Get a single task by ID
// @Tags tasks
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {object} utils.Response{data=models.Task}
// @Failure 404 {object} utils.Response
// @Router /tasks/{id} [get]
func GetTaskDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "Invalid task ID"))
		return
	}

	task, err := services.GetTaskByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, utils.NewErrorResponse(http.StatusNotFound, "Task not found"))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Task retrieved successfully", task))
}
