package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/financeapp/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// logRequest runs one request through the middleware and returns the entry.
func logRequest(t *testing.T, status int, target string) map[string]any {
	t.Helper()
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	r := gin.New()
	r.Use(middleware.RequestLogger(slog.New(slog.NewJSONHandler(&buf, nil))))
	r.GET("/items", func(c *gin.Context) { c.Status(status) })

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, target, nil))

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry), "output must be one JSON line")
	return entry
}

func TestRequestLogger_RecordsTheRequest(t *testing.T) {
	entry := logRequest(t, http.StatusOK, "/items")

	assert.Equal(t, "GET", entry["method"])
	assert.Equal(t, "/items", entry["path"])
	assert.EqualValues(t, 200, entry["status"])
	assert.Contains(t, entry, "duration_ms")
}

// The level has to follow the status, or 500s drown in the info stream.
func TestRequestLogger_LevelFollowsTheStatus(t *testing.T) {
	cases := map[int]string{
		http.StatusOK:                  "INFO",
		http.StatusNoContent:           "INFO",
		http.StatusBadRequest:          "WARN",
		http.StatusUnauthorized:        "WARN",
		http.StatusInternalServerError: "ERROR",
	}
	for status, want := range cases {
		t.Run(http.StatusText(status), func(t *testing.T) {
			assert.Equal(t, want, logRequest(t, status, "/items")["level"])
		})
	}
}

func TestRequestLogger_IncludesTheQueryWhenPresent(t *testing.T) {
	entry := logRequest(t, http.StatusOK, "/items?page=2&type=expense")
	assert.Equal(t, "page=2&type=expense", entry["query"])

	assert.NotContains(t, logRequest(t, http.StatusOK, "/items"), "query",
		"an empty query must not add a field")
}
