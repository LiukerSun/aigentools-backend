package main

import (
	"aigentools-backend/internal/api"
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"log"

	"golang.org/x/crypto/bcrypt"
)

// @title aigentools-backend API
// @version 1.0
// @description This is a sample server for aigentools-backend.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

func main() {
	router, err := api.NewRouter()
	if err != nil {
		log.Fatalf("failed to create router: %v", err)
	}

	// Migrate the schema
	err = database.DB.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	initAdminUser()

	if err := router.Run(":8080"); err != nil {
		log.Fatalf("failed to run server: %v", err)
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
				log.Fatalf("failed to hash admin password: %v", err)
			}

			adminUser = models.User{
				Username: adminUsername,
				Password: string(hashedPassword),
				Role:     "admin",
			}

			if err := database.DB.Create(&adminUser).Error; err != nil {
				log.Fatalf("failed to create admin user: %v", err)
			}
			log.Println("Admin user created successfully!")
		} else {
			log.Fatalf("failed to check for admin user: %v", result.Error)
		}
	} else {
		log.Println("Admin user already exists.")
	}
}
