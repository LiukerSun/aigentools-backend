package main

import (
	"aigentools-backend/internal/api"
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"aigentools-backend/pkg/logger"
	"log"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// @title aigentools-backend API
// @version 1.0
// @description This is a sample server for aigentools-backend.
// @description Authentication: Bearer Token (JWT).
// @description For Apifox/Swagger UI: Click "Authorize", enter "Bearer " followed by your token.
// @description Example: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	router, err := api.NewRouter()
	if err != nil {
		log.Fatalf("failed to create router: %v", err)
	}
	defer logger.Sync()

	// Migrate the schema
	err = database.DB.AutoMigrate(&models.User{}, &models.Transaction{}, &models.AIModel{})
	if err != nil {
		logger.Log.Fatal("failed to migrate database", zap.Error(err))
	}

	initAdminUser()

	if err := router.Run(":8080"); err != nil {
		logger.Log.Fatal("failed to run server", zap.Error(err))
	}
}

func initAdminUser() {
	adminUsername := "admin@admin.com"
	adminPassword := "RealX1234"

	var adminUser models.User
	result := database.DB.Where("username = ?", adminUsername).First(&adminUser)

	if result.Error != nil {
		if result.Error.Error() == "record not found" {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
			if err != nil {
				logger.Log.Fatal("failed to hash admin password", zap.Error(err))
			}

			adminUser = models.User{
				Username: adminUsername,
				Password: string(hashedPassword),
				Role:     "admin",
			}

			if err := database.DB.Create(&adminUser).Error; err != nil {
				logger.Log.Fatal("failed to create admin user", zap.Error(err))
			}
			logger.Log.Info("Admin user created successfully!")
		} else {
			logger.Log.Fatal("failed to check for admin user", zap.Error(result.Error))
		}
	} else {
		logger.Log.Info("Admin user already exists.")
	}
}
