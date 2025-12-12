package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	APIKeys        []string
	BrowserURL     string
	MaxConcurrency int
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8087"
	}

	apiKeysStr := os.Getenv("API_KEYS")
	if apiKeysStr == "" {
		apiKeysStr = "dev-key-123"
	}
	apiKeys := strings.Split(apiKeysStr, ",")

	browserURL := os.Getenv("BROWSER_URL")

	maxConcurrency := 3
	if mc := os.Getenv("MAX_CONCURRENCY"); mc != "" {
		if v, err := strconv.Atoi(mc); err == nil && v > 0 {
			maxConcurrency = v
		}
	}

	return &Config{
		Port:           port,
		APIKeys:        apiKeys,
		BrowserURL:     browserURL,
		MaxConcurrency: maxConcurrency,
	}
}
