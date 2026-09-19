package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/financeapp/backend/pkg/config"
	"github.com/financeapp/backend/pkg/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const userIDKey = "user_id"

// Auth returns a Gin middleware that validates JWT tokens.
func Auth(cfg *config.JWTConfig, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "authorization header required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid authorization format"})
			return
		}

		tokenStr := parts[1]

		// Check if token is blacklisted in Redis.
		blacklistKey := "blacklist:" + tokenStr
		exists, err := rdb.Exists(context.Background(), blacklistKey).Result()
		if err == nil && exists > 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "token revoked"})
			return
		}

		claims, err := security.ParseToken(tokenStr, cfg.Secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid or expired token"})
			return
		}

		c.Set(userIDKey, claims.UserID)
		c.Set("token", tokenStr)
		c.Next()
	}
}

// GetUserID extracts the authenticated user ID from the Gin context.
func GetUserID(c *gin.Context) uuid.UUID {
	return c.MustGet(userIDKey).(uuid.UUID)
}
