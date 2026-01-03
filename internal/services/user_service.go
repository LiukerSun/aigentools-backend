package services

import (
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
)

func FindUserByID(userID uint) (models.User, error) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return user, err
	}
	return user, nil
}
