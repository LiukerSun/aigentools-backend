package api

import (
	"aigentools-backend/config"
	_ "aigentools-backend/docs"
	adminTransaction "aigentools-backend/internal/api/v1/admin/transaction"
	adminUser "aigentools-backend/internal/api/v1/admin/user"
	"aigentools-backend/internal/api/v1/auth"
	userRoutes "aigentools-backend/internal/api/v1/user"
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/middleware"

	"github.com/gin-contrib/cors" // Import the cors middleware
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewRouter() (*gin.Engine, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}

	_, err = database.Connect(cfg.DSN())
	if err != nil {
		return nil, err
	}

	err = database.ConnectRedis(cfg)
	if err != nil {
		return nil, err
	}

	router := gin.Default()

	// Configure CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:8080"}, // Allow frontend origin
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum age for preflight requests
	}))

	// Swagger
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1
	v1 := router.Group("/api/v1")
	{
		auth.RegisterRoutes(v1)

		authorized := v1.Group("/")
		authorized.Use(middleware.AuthMiddleware())
		{
			userRoutes.RegisterRoutes(authorized)
		}

		// Admin routes
		admin := v1.Group("/admin")
		admin.Use(middleware.AdminAuthMiddleware())
		{
			adminUser.RegisterRoutes(admin)
			adminTransaction.RegisterRoutes(admin)
		}
	}

	return router, nil
}
