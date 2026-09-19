package routes_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/financeapp/backend/internal/auth"
	"github.com/financeapp/backend/internal/category"
	"github.com/financeapp/backend/internal/expense"
	"github.com/financeapp/backend/internal/importer"
	"github.com/financeapp/backend/internal/routes"
	"github.com/financeapp/backend/pkg/config"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

type route struct{ method, path string }

// The exact set of routes the API exposes. Moving registration out of main.go
// must not add, drop or rename a single one, and this is what proves it.
var wantRoutes = []route{
	{http.MethodGet, "/health"},
	{http.MethodGet, "/swagger/*any"},
	{http.MethodPost, "/api/v1/auth/register"},
	{http.MethodPost, "/api/v1/auth/login"},
	{http.MethodPost, "/api/v1/auth/logout"},
	{http.MethodPost, "/api/v1/expenses"},
	{http.MethodGet, "/api/v1/expenses"},
	{http.MethodGet, "/api/v1/expenses/:id"},
	{http.MethodPut, "/api/v1/expenses/:id"},
	{http.MethodDelete, "/api/v1/expenses/:id"},
	{http.MethodGet, "/api/v1/reports/monthly"},
	{http.MethodGet, "/api/v1/reports/categories"},
	{http.MethodPost, "/api/v1/categories"},
	{http.MethodGet, "/api/v1/categories"},
	{http.MethodPut, "/api/v1/categories/:id"},
	{http.MethodDelete, "/api/v1/categories/:id"},
	{http.MethodPost, "/api/v1/imports"},
	{http.MethodGet, "/api/v1/imports"},
	{http.MethodDelete, "/api/v1/imports/:id"},
}

// publicRoutes are the only ones reachable without a token.
var publicRoutes = map[route]bool{
	{http.MethodGet, "/health"}:                true,
	{http.MethodGet, "/swagger/*any"}:          true,
	{http.MethodPost, "/api/v1/auth/register"}: true,
	{http.MethodPost, "/api/v1/auth/login"}:    true,
}

// newRouter builds the real router for an environment. The handlers get nil
// services because no test here reaches them: protected routes stop at the
// middleware, and the public ones fail payload binding first. Redis points
// nowhere on purpose — the middleware treats a lookup error as "not revoked".
func newRouter(env string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	routes.Register(r, routes.Handlers{
		Auth:     auth.NewHandler(nil),
		Expense:  expense.NewHandler(nil),
		Category: category.NewHandler(nil),
		Import:   importer.NewHandler(nil),
	}, &config.Config{
		App: config.AppConfig{Env: env},
		JWT: config.JWTConfig{Secret: "test-secret", ExpiryHours: 1},
	}, redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"}))
	return r
}

func TestRegister_MountsTheExpectedRoutes(t *testing.T) {
	var got []route
	for _, ri := range newRouter("development").Routes() {
		got = append(got, route{ri.Method, ri.Path})
	}
	assert.ElementsMatch(t, wantRoutes, got, "the route table changed")
}

// Every route that is not explicitly public must answer 401 without a token.
// A regression here would expose one user's data to anyone.
func TestRegister_ProtectedRoutesRequireAToken(t *testing.T) {
	r := newRouter("development")

	for _, rt := range wantRoutes {
		if publicRoutes[rt] {
			continue
		}
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(rt.method, concrete(rt.path), nil))
			assert.Equal(t, http.StatusUnauthorized, rec.Code, "route is reachable without a token")
		})
	}
}

func TestRegister_PublicRoutesDoNotRequireAToken(t *testing.T) {
	r := newRouter("development")

	for rt := range publicRoutes {
		if rt.path == "/swagger/*any" {
			continue // covered by TestRegister_SwaggerIsOffInProduction
		}
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(rt.method, rt.path, nil))
			assert.NotEqual(t, http.StatusUnauthorized, rec.Code,
				"this route must be reachable without a token")
		})
	}
}

// concrete replaces the :id placeholder so the request matches the route.
func concrete(path string) string {
	if i := len(path) - len("/:id"); i > 0 && path[i:] == "/:id" {
		return path[:i] + "/00000000-0000-0000-0000-000000000000"
	}
	return path
}

// The Swagger UI publishes the whole API surface, so it must not be mounted in
// production.
func TestRegister_SwaggerIsOffInProduction(t *testing.T) {
	has := func(r *gin.Engine) bool {
		for _, ri := range r.Routes() {
			if strings.HasPrefix(ri.Path, "/swagger") {
				return true
			}
		}
		return false
	}

	assert.True(t, has(newRouter("development")), "the UI should be available in development")
	assert.False(t, has(newRouter("production")), "the UI must not be exposed in production")
}
