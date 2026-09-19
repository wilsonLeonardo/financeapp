package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/financeapp/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func corsRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.CORS())
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestCORS_AllowsTheFrontendOrigins(t *testing.T) {
	for _, origin := range []string{
		"http://localhost:3000", "http://localhost:5173", "http://127.0.0.1:3000",
	} {
		t.Run(origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			req.Header.Set("Origin", origin)
			rec := httptest.NewRecorder()
			corsRouter().ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, origin, rec.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
		})
	}
}

func TestCORS_RejectsAnUnknownOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	corsRouter().ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"),
		"an unlisted origin must not be echoed back")
}

func TestCORS_PreflightAdvertisesMethodsAndHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "PUT")
	req.Header.Set("Access-Control-Request-Headers", "Authorization")
	rec := httptest.NewRecorder()
	corsRouter().ServeHTTP(rec, req)

	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "PUT")
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Authorization")
}
