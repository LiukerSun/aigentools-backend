package api

import (
	"aigentools-backend/config"
	_ "aigentools-backend/docs"
	"aigentools-backend/internal/api/test"
	adminOrder "aigentools-backend/internal/api/v1/admin/order"
	adminPayment "aigentools-backend/internal/api/v1/admin/payment"
	adminTransaction "aigentools-backend/internal/api/v1/admin/transaction"
	adminUser "aigentools-backend/internal/api/v1/admin/user"
	aiAssistant "aigentools-backend/internal/api/v1/ai_assistant"
	aiModel "aigentools-backend/internal/api/v1/ai_model"
	"aigentools-backend/internal/api/v1/auth"
	"aigentools-backend/internal/api/v1/common/upload"
	"aigentools-backend/internal/api/v1/payment"
	"aigentools-backend/internal/api/v1/task"
	userRoutes "aigentools-backend/internal/api/v1/user"
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/middleware"
	"aigentools-backend/pkg/logger"

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

	// Initialize Logger
	err = logger.InitLogger(&logger.Config{
		Level:      cfg.LogLevel,
		Filename:   cfg.LogFilename,
		MaxSize:    cfg.LogMaxSize,
		MaxBackups: cfg.LogMaxBackups,
		MaxAge:     cfg.LogMaxAge,
		Compress:   cfg.LogCompress,
	})
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

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.SetTrustedProxies(nil)

	// Configure CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"*", // Allow all origins for development; restrict in production
		}, // Allow frontend origin
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
		// Test routes
		test.RegisterRoutes(v1)

		auth.RegisterRoutes(v1)
		aiModel.RegisterRoutes(v1)
		upload.RegisterRoutes(v1)
		task.RegisterRoutes(v1)
		payment.RegisterRoutes(v1)

		authorized := v1.Group("/")
		authorized.Use(middleware.AuthMiddleware())
		{
			userRoutes.RegisterRoutes(authorized)
			aiAssistant.RegisterRoutes(authorized)
		}

		// Admin routes
		admin := v1.Group("/admin")
		admin.Use(middleware.AdminAuthMiddleware())
		{
			adminUser.RegisterRoutes(admin)
			adminTransaction.RegisterRoutes(admin)
			adminPayment.RegisterRoutes(admin)
			adminOrder.RegisterRoutes(admin)
		}
	}

	return router, nil
}
