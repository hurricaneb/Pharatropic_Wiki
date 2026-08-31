package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"wiki/internal/models"
	"wiki/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var JWTSecret []byte

// InitJWTSecret sets the signing key used for JWTs. If secret is empty, a
// random key is generated for this process run and a warning is logged,
// since that invalidates every session on restart.
func InitJWTSecret(secret string) {
	if secret == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			log.Fatalf("Kunde inte generera JWT-hemlighet: %v", err)
		}
		secret = hex.EncodeToString(b)
		log.Println("⚠️  JWT_SECRET är inte satt — genererade en tillfällig hemlighet för denna körning. Sätt miljövariabeln JWT_SECRET för att sessioner ska överleva omstarter.")
	}
	JWTSecret = []byte(secret)
}

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(user *models.User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

func ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return JWTSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("ogiltig token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("ogiltiga token claims")
	}
	return claims, nil
}

// AuthMiddleware handles JWT Bearer tokens and User API Keys
func AuthMiddleware(repo *repository.WikiRepository, masterKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Check Bearer Token
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := ParseToken(tokenStr)
			if err == nil {
				user, err := repo.GetUserByID(claims.UserID)
				if err == nil {
					c.Set("user", user)
					c.Set("username", user.Username)
					c.Next()
					return
				}
			}
		}

		// 2. Check X-API-Key or query param
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		if apiKey != "" {
			// Check master key
			if masterKey != "" && apiKey == masterKey {
				c.Set("username", "MasterAdmin")
				c.Next()
				return
			}

			// Check DB User API Key
			user, _, err := repo.ValidateApiKey(apiKey)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				c.Abort()
				return
			}

			c.Set("user", user)
			c.Set("username", user.Username)
			c.Next()
			return
		}

		c.Next()
	}
}

// RequireAuth ensures the request has an authenticated user or master API key
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, existsUser := c.Get("user")
		_, existsUsername := c.Get("username")

		if !existsUser && !existsUsername {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Du måste vara inloggad för att utföra denna åtgärd"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireAdmin ensures the authenticated user has the 'admin' role
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userVal, exists := c.Get("user")
		if !exists {
			// Check if master key admin
			if username, ok := c.Get("username"); ok && username == "MasterAdmin" {
				c.Next()
				return
			}

			c.JSON(http.StatusForbidden, gin.H{"error": "Endast administratörer har behörighet"})
			c.Abort()
			return
		}

		user, ok := userVal.(*models.User)
		if !ok || user.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Endast administratörer har behörighet"})
			c.Abort()
			return
		}

		c.Next()
	}
}
