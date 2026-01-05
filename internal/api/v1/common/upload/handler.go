package upload

import (
	"aigentools-backend/internal/services"
	"aigentools-backend/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetOSSToken godoc
// @Summary Get OSS STS Token
// @Description Get STS token for uploading files to Alibaba Cloud OSS
// @Tags common
// @Accept json
// @Produce json
// @Success 200 {object} utils.Response{data=services.STSCredentials}
// @Router /common/upload/token [get]
func GetOSSToken(c *gin.Context) {
	token, err := services.GetOSSTSToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to get OSS token: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("OSS token retrieved successfully", token))
}
