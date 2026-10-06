package main

import (
	"fmt"
	"os"

	"github.com/financeapp/backend/internal/auth"
	"github.com/financeapp/backend/internal/category"
	"github.com/financeapp/backend/internal/database"
	"github.com/financeapp/backend/internal/expense"
	"github.com/financeapp/backend/internal/routes"
	"github.com/financeapp/backend/pkg/config"
	"github.com/financeapp/backend/pkg/logger"
	"github.com/financeapp/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// @title           FinanceApp API
// @version         1.0
// @description     Personal finance tracker: transactions and categories.
// @BasePath        /api/v1
//
// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
// @description                Paste the token returned by /auth/login as: Bearer <token>
func main() {
	cfg := config.Load()
	log := logger.Init(logger.Options{Env: cfg.App.Env, Level: cfg.App.LogLevel})

	db, err := database.NewPostgres(cfg, log)
	if err != nil {
		log.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}

	rdb, err := database.NewRedis(cfg, log)
	if err != nil {
		log.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}

	// Repositories
	authRepo := auth.NewRepository(db, log)
	expenseRepo := expense.NewRepository(db)
	categoryRepo := category.NewRepository(db)

	// Services
	authService := auth.NewService(authRepo, rdb, &cfg.JWT, log)
	expenseService := expense.NewService(expenseRepo, log)
	categoryService := category.NewService(categoryRepo, log)

	// Handlers
	authHandler := auth.NewHandler(authService)
	expenseHandler := expense.NewHandler(expenseService)
	categoryHandler := category.NewHandler(categoryService)

	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.RequestLogger(log))
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	routes.Register(r, routes.Handlers{
		Auth:     authHandler,
		Expense:  expenseHandler,
		Category: categoryHandler,
	}, cfg, rdb)

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Info("server listening", "addr", addr)
	if err := r.Run(addr); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
