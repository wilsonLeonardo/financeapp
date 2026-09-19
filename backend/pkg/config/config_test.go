package config_test

import (
	"testing"

	"github.com/financeapp/backend/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestLoad_Defaults(t *testing.T) {
	// t.Setenv("", "") is not possible, so the defaults are exercised by
	// clearing the keys this test cares about.
	for _, k := range []string{"APP_ENV", "LOG_LEVEL", "SERVER_PORT", "POSTGRES_HOST", "REDIS_DB", "JWT_EXPIRY_HOURS"} {
		t.Setenv(k, "")
	}
	cfg := config.Load()

	assert.Equal(t, "development", cfg.App.Env)
	assert.Equal(t, "info", cfg.App.LogLevel)
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Postgres.Host)
	assert.Equal(t, 0, cfg.Redis.DB)
	assert.Equal(t, 24, cfg.JWT.ExpiryHours)
}

func TestLoad_ReadsTheEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("POSTGRES_HOST", "db.internal")
	t.Setenv("POSTGRES_PORT", "6543")
	t.Setenv("POSTGRES_USER", "app")
	t.Setenv("POSTGRES_PASSWORD", "s3cret")
	t.Setenv("POSTGRES_DB", "finance")
	t.Setenv("REDIS_HOST", "cache.internal")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDIS_DB", "3")
	t.Setenv("JWT_SECRET", "a-real-secret")
	t.Setenv("JWT_EXPIRY_HOURS", "8")

	cfg := config.Load()

	assert.Equal(t, "production", cfg.App.Env)
	assert.Equal(t, "warn", cfg.App.LogLevel)
	assert.Equal(t, "9090", cfg.Server.Port)
	assert.Equal(t, "db.internal", cfg.Postgres.Host)
	assert.Equal(t, 3, cfg.Redis.DB)
	assert.Equal(t, "a-real-secret", cfg.JWT.Secret)
	assert.Equal(t, 8, cfg.JWT.ExpiryHours)
}

// A non-numeric value must not crash startup; the default takes over.
func TestLoad_NonNumericIntFallsBackToTheDefault(t *testing.T) {
	t.Setenv("JWT_EXPIRY_HOURS", "not-a-number")
	t.Setenv("REDIS_DB", "abc")

	cfg := config.Load()
	assert.Equal(t, 24, cfg.JWT.ExpiryHours)
	assert.Equal(t, 0, cfg.Redis.DB)
}

func TestPostgresDSN(t *testing.T) {
	cfg := config.PostgresConfig{
		Host: "db", Port: "5432", User: "app", Password: "pw", DBName: "finance",
	}
	dsn := cfg.DSN()

	assert.Contains(t, dsn, "host=db")
	assert.Contains(t, dsn, "port=5432")
	assert.Contains(t, dsn, "user=app")
	assert.Contains(t, dsn, "dbname=finance")
	assert.Contains(t, dsn, "sslmode=disable")
	assert.Contains(t, dsn, "TimeZone=UTC")
}

func TestRedisAddr(t *testing.T) {
	cfg := config.RedisConfig{Host: "cache", Port: "6379"}
	assert.Equal(t, "cache:6379", cfg.Addr())
}
