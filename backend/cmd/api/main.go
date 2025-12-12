package main

import (
	"log"
	"time"

	"github.com/cleberrangel/correios_api/internal/api/handlers"
	"github.com/cleberrangel/correios_api/internal/api/middleware"
	"github.com/cleberrangel/correios_api/internal/api/routes"
	"github.com/cleberrangel/correios_api/internal/auth"
	"github.com/cleberrangel/correios_api/internal/config"
	"github.com/cleberrangel/correios_api/internal/domain/scraper"
	"github.com/gin-gonic/gin"

	_ "github.com/cleberrangel/correios_api/docs"
)

// @title Correios API
// @version 1.0
// @description API for tracking packages by CPF
// @host localhost:8087
// @BasePath /
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
func main() {
	cfg := config.Load()

	rodScraper, err := scraper.NewRodScraper(cfg.BrowserURL)
	if err != nil {
		log.Fatalf("Failed to initialize scraper: %v", err)
	}
	defer rodScraper.Close()

	validator := auth.NewValidator(cfg.APIKeys)
	limiter := middleware.NewRateLimiter(100, time.Minute)
	trackerHandler := handlers.NewTrackerHandler(rodScraper)

	r := gin.Default()
	routes.Setup(r, trackerHandler, validator, limiter)

	log.Printf("Starting server on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
