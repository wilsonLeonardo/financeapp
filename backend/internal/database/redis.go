package database

import (
	"context"
	"log"

	"github.com/financeapp/backend/internal/config"
	"github.com/redis/go-redis/v9"
)

// NewRedis creates and validates a new Redis client connection.
func NewRedis(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, err
	}

	log.Println("✅ Redis connected")
	return client, nil
}
