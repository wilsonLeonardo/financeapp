package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/financeapp/backend/internal/config"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const userIDKey = "user_id"

// Claims represents the JWT payload.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateToken creates a signed JWT for a user.
func GenerateToken(userID uuid.UUID, cfg *config.JWTConfig) (string, error) {
	expiresAt := time.Now().Add(time.Duration(cfg.ExpiryHours) * time.Hour)

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

// ParseToken validates and parses a JWT string.
func ParseToken(tokenStr string, cfg *config.JWTConfig) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, apperrors.ErrUnauthorized
		}
		return []byte(cfg.Secret), nil
	})
	if err != nil || !token.Valid {
		return nil, apperrors.ErrUnauthorized
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, apperrors.ErrUnauthorized
	}
	return claims, nil
}

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

		claims, err := ParseToken(tokenStr, cfg)
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
