package middleware

import (
	"net/http"

	"github.com/cleberrangel/correios_api/internal/auth"
	"github.com/gin-gonic/gin"
)

func APIKeyAuth(validator *auth.Validator) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			c.Abort()
			return
		}

		if !validator.IsValid(apiKey) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			c.Abort()
			return
		}

		c.Next()
	}
}
