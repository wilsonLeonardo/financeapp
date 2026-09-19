package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/financeapp/backend/pkg/config"
	"github.com/financeapp/backend/pkg/middleware"
	"github.com/financeapp/backend/pkg/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var jwtCfg = &config.JWTConfig{Secret: "test-secret", ExpiryHours: 1}

// unreachableRedis points nowhere. The middleware treats a failed denylist
// lookup as "not revoked", so this exercises the path taken when Redis is down.
func unreachableRedis() *redis.Client {
	return redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 50 * time.Millisecond})
}

// protectedRouter mounts one guarded route that echoes the authenticated id.
func protectedRouter(rdb *redis.Client) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/me", middleware.Auth(jwtCfg, rdb), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": middleware.GetUserID(c).String()})
	})
	return r
}

func do(r *gin.Engine, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestAuth_AcceptsAValidToken(t *testing.T) {
	userID := uuid.New()
	token, err := security.GenerateToken(userID, jwtCfg.Secret, time.Hour)
	require.NoError(t, err)

	rec := do(protectedRouter(unreachableRedis()), "Bearer "+token)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), userID.String(), "the handler must see the caller id")
}

// The scheme is compared case-insensitively.
func TestAuth_AcceptsAnyCaseOfBearer(t *testing.T) {
	token, err := security.GenerateToken(uuid.New(), jwtCfg.Secret, time.Hour)
	require.NoError(t, err)

	for _, scheme := range []string{"Bearer", "bearer", "BEARER", "BeArEr"} {
		t.Run(scheme, func(t *testing.T) {
			rec := do(protectedRouter(unreachableRedis()), scheme+" "+token)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestAuth_RejectsMissingHeader(t *testing.T) {
	rec := do(protectedRouter(unreachableRedis()), "")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "authorization header required")
}

func TestAuth_RejectsMalformedHeader(t *testing.T) {
	token, err := security.GenerateToken(uuid.New(), jwtCfg.Secret, time.Hour)
	require.NoError(t, err)

	cases := map[string]string{
		"no scheme":    token,
		"wrong scheme": "Basic " + token,
		"scheme only":  "Bearer",
	}
	for name, header := range cases {
		t.Run(name, func(t *testing.T) {
			rec := do(protectedRouter(unreachableRedis()), header)
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			assert.Contains(t, rec.Body.String(), "invalid authorization format")
		})
	}
}

func TestAuth_RejectsInvalidToken(t *testing.T) {
	expired, err := security.GenerateToken(uuid.New(), jwtCfg.Secret, -time.Hour)
	require.NoError(t, err)
	forged, err := security.GenerateToken(uuid.New(), "another-secret", time.Hour)
	require.NoError(t, err)

	cases := map[string]string{
		"garbage": "not-a-jwt",
		"expired": expired,
		"forged":  forged,
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			rec := do(protectedRouter(unreachableRedis()), "Bearer "+token)
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			assert.Contains(t, rec.Body.String(), "invalid or expired token")
		})
	}
}

// The middleware must stop the chain, not just set a status.
func TestAuth_AbortsBeforeTheHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reached := false
	r := gin.New()
	r.GET("/me", middleware.Auth(jwtCfg, unreachableRedis()), func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})

	do(r, "")
	assert.False(t, reached, "the handler ran despite a rejected request")
}

// The raw token is published for the logout handler to revoke.
func TestAuth_PutsTheTokenInTheContext(t *testing.T) {
	token, err := security.GenerateToken(uuid.New(), jwtCfg.Secret, time.Hour)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	var seen any
	r := gin.New()
	r.GET("/me", middleware.Auth(jwtCfg, unreachableRedis()), func(c *gin.Context) {
		seen, _ = c.Get("token")
		c.Status(http.StatusOK)
	})

	do(r, "Bearer "+token)
	assert.Equal(t, token, seen)
}

func TestGetUserID_PanicsWithoutTheMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	assert.Panics(t, func() { middleware.GetUserID(c) },
		"an unauthenticated context must fail loudly, not return the zero uuid")
}
