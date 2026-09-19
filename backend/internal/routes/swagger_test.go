package routes_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The UI and its spec must actually be served, not just registered.
func TestSwaggerUIIsServed(t *testing.T) {
	r := newRouter("development")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, strings.ToLower(rec.Body.String()), "swagger")

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"FinanceApp API"`)
	assert.Contains(t, rec.Body.String(), `"/expenses"`)
}
