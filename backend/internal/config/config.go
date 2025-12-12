package config

import (
	"os"
	"strings"
)

type Config struct {
	Port       string
	APIKeys    []string
	BrowserURL string
}

func Load() *Config {
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

	return &Config{
		Port:       port,
		APIKeys:    apiKeys,
		BrowserURL: browserURL,
	}
}
