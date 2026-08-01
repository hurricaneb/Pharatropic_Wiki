package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth verifies the X-API-Key header if configured
func APIKeyAuth(masterKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		// If a key is sent, check it
		if apiKey != "" && apiKey != masterKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Ogiltig API-nyckel"})
			c.Abort()
			return
		}

		c.Next()
	}
}
