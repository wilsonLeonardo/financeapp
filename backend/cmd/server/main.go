package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/financeapp/backend/internal/auth"
	"github.com/financeapp/backend/internal/category"
	"github.com/financeapp/backend/internal/config"
	"github.com/financeapp/backend/internal/database"
	"github.com/financeapp/backend/internal/expense"
	"github.com/financeapp/backend/internal/importer"
	"github.com/financeapp/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db, err := database.NewPostgres(cfg)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}

	rdb, err := database.NewRedis(cfg)
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	// ── Repositories ────────────────────────────────────────────────────────
	authRepo := auth.NewRepository(db)
	expenseRepo := expense.NewRepository(db)
	categoryRepo := category.NewRepository(db)
	importRepo := importer.NewRepository(db)

	// ── Services ─────────────────────────────────────────────────────────────
	authService := auth.NewService(authRepo, rdb, &cfg.JWT)
	expenseService := expense.NewService(expenseRepo)
	categoryService := category.NewService(categoryRepo)
	importService := importer.NewService(importRepo, expenseRepo)

	// ── Handlers ─────────────────────────────────────────────────────────────
	authHandler := auth.NewHandler(authService)
	expenseHandler := expense.NewHandler(expenseService)
	categoryHandler := category.NewHandler(categoryService)
	importHandler := importer.NewHandler(importService)

	// ── Router ────────────────────────────────────────────────────────────────
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")

	// Public routes
	authRoutes := v1.Group("/auth")
	{
		authRoutes.POST("/register", authHandler.Register)
		authRoutes.POST("/login", authHandler.Login)
	}

	// Protected routes
	protected := v1.Group("")
	protected.Use(middleware.Auth(&cfg.JWT, rdb))
	{
		protected.POST("/auth/logout", authHandler.Logout)

		// Expenses
		protected.POST("/expenses", expenseHandler.Create)
		protected.GET("/expenses", expenseHandler.List)
		protected.GET("/expenses/:id", expenseHandler.GetByID)
		protected.PUT("/expenses/:id", expenseHandler.Update)
		protected.DELETE("/expenses/:id", expenseHandler.Delete)

		// Reports
		protected.GET("/reports/monthly", expenseHandler.MonthlySummary)
		protected.GET("/reports/categories", expenseHandler.CategorySummary)

		// Categories
		protected.POST("/categories", categoryHandler.Create)
		protected.GET("/categories", categoryHandler.List)
		protected.PUT("/categories/:id", categoryHandler.Update)
		protected.DELETE("/categories/:id", categoryHandler.Delete)

		// Import
		protected.POST("/imports", importHandler.Import)
		protected.GET("/imports", importHandler.List)
		protected.DELETE("/imports/:id", importHandler.Revert)
	}

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("🚀 Server running on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
