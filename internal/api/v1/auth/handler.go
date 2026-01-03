package auth

import (
	"aigentools-backend/internal/api/v1/user"
	"aigentools-backend/internal/services"
	"aigentools-backend/internal/utils"
	"errors" // Keep errors for errors.Is
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type LogoutResponse struct {
	Message string `json:"message"`
}

type RegisterInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Register godoc
// @Summary Register a new user
// @Description Register a new user with a username and password
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   input     body   RegisterInput  true  "Register Input"
// @Success 201 {object} utils.Response{data=user.UserResponse}
// @Failure 400 {object} utils.Response
// @Failure 409 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /auth/register [post]
func Register(c *gin.Context) {
	var input RegisterInput
	if !utils.BindAndValidate(c, &input) {
		return
	}

	u, err := services.RegisterUser(input.Username, input.Password)
	if err != nil {
		if errors.Is(err, services.ErrUserAlreadyExists) {
			c.JSON(http.StatusConflict, utils.NewErrorResponse(http.StatusConflict, err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to register user due to an internal error"))
		return
	}

	token, err := utils.GenerateToken(u.ID, u.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Could not generate token"))
		return
	}

	c.JSON(http.StatusCreated, utils.NewSuccessResponse("User registered successfully", user.UserResponse{
		ID:       u.ID,
		Username: u.Username,
		Role:     u.Role,
		Token:    token,
	}))
}

type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password"binding:"required"`
}

// Login godoc
// @Summary Log in a user
// @Description Log in a user with a username and password
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   input     body   LoginInput  true  "Login Input"
// @Success 200 {object} utils.Response{data=user.UserResponse}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Router /auth/login [post]
func Login(c *gin.Context) {
	var input LoginInput
	if !utils.BindAndValidate(c, &input) {
		return
	}

	token, u, err := services.LoginUser(input.Username, input.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, utils.NewErrorResponse(http.StatusUnauthorized, "Invalid username or password"))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Logged in successfully", user.UserResponse{
		ID:       u.ID,
		Username: u.Username,
		Role:     u.Role,
		Token:    token,
	}))
}

// Logout godoc
// @Summary Log out a user
// @Description Invalidate the user's current token
// @Tags auth
// @Produce  json
// @Security ApiKeyAuth
// @Success 200 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /auth/logout [post]
func Logout(c *gin.Context) {
	tokenString, err := utils.ExtractToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, utils.NewErrorResponse(http.StatusUnauthorized, err.Error()))
		return
	}

	claims, err := utils.ValidateToken(tokenString)
	if err != nil {
		// It's already invalid, but we can still try to denylist it
		// We just can't get the expiration time
		err = services.AddToDenylist(tokenString, time.Hour*72) // Max token life
		if err != nil {
			c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to denylist token"))
			return
		}
		c.JSON(http.StatusOK, utils.NewSuccessResponse("Logged out successfully", nil))
		return
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Invalid token expiration"))
		return
	}

	expTime := time.Unix(int64(exp), 0)
	remaining := time.Until(expTime)

	err = services.AddToDenylist(tokenString, remaining)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to denylist token"))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessResponse("Logged out successfully", nil))
}
