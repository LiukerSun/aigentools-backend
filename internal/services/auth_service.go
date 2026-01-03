package services

import (
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"aigentools-backend/internal/utils"
	"errors"

	"gorm.io/gorm" // Import gorm for ErrRecordNotFound
	"golang.org/x/crypto/bcrypt"
)

var ErrUserAlreadyExists = errors.New("user with this username already exists")

func RegisterUser(username, password string) (*models.User, error) {
	// Check if user already exists
	var existingUser models.User
	result := database.DB.Where("username = ?", username).First(&existingUser)
	if result.Error == nil {
		return nil, ErrUserAlreadyExists // User already exists
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error // Other database error
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	var userCount int64
	database.DB.Model(&models.User{}).Count(&userCount)

	role := "user"
	if userCount == 0 {
		role = "admin"
	}

	user := &models.User{
		Username: username,
		Password: string(hashedPassword),
		Role:     role,
	}

	if err := database.DB.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func LoginUser(username, password string) (string, *models.User, error) {
	var user models.User
	if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return "", nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	token, err := utils.GenerateToken(user.ID, user.Role)
	if err != nil {
		return "", nil, err
	}

	return token, &user, nil
}
