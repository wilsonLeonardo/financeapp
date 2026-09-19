// Package security holds password hashing and JWT signing. It takes plain
// parameters, not app config, so it never imports internal packages.
package security

import (
	"time"

	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims is the JWT payload carried by an access token.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateToken signs an access token for a user.
func GenerateToken(userID uuid.UUID, secret string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseToken verifies a token and returns its claims. Every failure collapses
// into ErrUnauthorized so forged and expired tokens look the same.
func ParseToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, apperrors.ErrUnauthorized
		}
		return []byte(secret), nil
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
