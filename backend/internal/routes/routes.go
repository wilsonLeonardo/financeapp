// Package routes maps URLs to handlers. Keeping it here leaves handlers
// unaware of their paths and main.go a bootstrap.
package routes

import (
	"net/http"

	_ "github.com/financeapp/backend/docs"
	"github.com/financeapp/backend/internal/auth"
	"github.com/financeapp/backend/internal/category"
	"github.com/financeapp/backend/internal/expense"
	"github.com/financeapp/backend/internal/importer"
	"github.com/financeapp/backend/pkg/config"
	"github.com/financeapp/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Handlers collects the handlers the router needs.
type Handlers struct {
	Auth     *auth.Handler
	Expense  *expense.Handler
	Category *category.Handler
	Import   *importer.Handler
}

// Register mounts the health check and the /api/v1 tree on r. The Swagger UI is
// left out in production, where publishing the API surface is not worth it.
func Register(r *gin.Engine, h Handlers, cfg *config.Config, rdb *redis.Client) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	if cfg.App.Env != "production" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	v1 := r.Group("/api/v1")

	registerPublic(v1, h)
	registerProtected(v1, h, &cfg.JWT, rdb)
}

// registerPublic mounts the routes reachable without a token.
func registerPublic(v1 *gin.RouterGroup, h Handlers) {
	authRoutes := v1.Group("/auth")
	{
		authRoutes.POST("/register", h.Auth.Register)
		authRoutes.POST("/login", h.Auth.Login)
	}
}

// registerProtected mounts everything behind the auth middleware.
func registerProtected(v1 *gin.RouterGroup, h Handlers, jwtCfg *config.JWTConfig, rdb *redis.Client) {
	protected := v1.Group("")
	protected.Use(middleware.Auth(jwtCfg, rdb))

	protected.POST("/auth/logout", h.Auth.Logout)

	expenses := protected.Group("/expenses")
	{
		expenses.POST("", h.Expense.Create)
		expenses.GET("", h.Expense.List)
		expenses.GET("/:id", h.Expense.GetByID)
		expenses.PUT("/:id", h.Expense.Update)
		expenses.DELETE("/:id", h.Expense.Delete)
	}

	reports := protected.Group("/reports")
	{
		reports.GET("/monthly", h.Expense.MonthlySummary)
		reports.GET("/categories", h.Expense.CategorySummary)
	}

	categories := protected.Group("/categories")
	{
		categories.POST("", h.Category.Create)
		categories.GET("", h.Category.List)
		categories.PUT("/:id", h.Category.Update)
		categories.DELETE("/:id", h.Category.Delete)
	}

	imports := protected.Group("/imports")
	{
		imports.POST("", h.Import.Import)
		imports.GET("", h.Import.List)
		imports.DELETE("/:id", h.Import.Revert)
	}
}
