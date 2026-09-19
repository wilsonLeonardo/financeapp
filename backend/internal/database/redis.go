package database

import (
	"context"
	"log/slog"

	"github.com/financeapp/backend/pkg/config"
	"github.com/redis/go-redis/v9"
)

// NewRedis creates and validates a new Redis client connection.
func NewRedis(cfg *config.Config, log *slog.Logger) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, err
	}

	log.Info("redis connected", "addr", cfg.Redis.Addr())
	return client, nil
}
