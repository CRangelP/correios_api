package routes

import (
	"github.com/cleberrangel/correios_api/internal/api/handlers"
	"github.com/cleberrangel/correios_api/internal/api/middleware"
	"github.com/cleberrangel/correios_api/internal/auth"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Setup(r *gin.Engine, trackerHandler *handlers.TrackerHandler, validator *auth.Validator, limiter *middleware.RateLimiter) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	api.Use(middleware.APIKeyAuth(validator))
	api.Use(middleware.RateLimit(limiter))
	{
		api.POST("/tracker/cpf", trackerHandler.TrackCPF)
	}
}
