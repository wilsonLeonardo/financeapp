package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration.
type Config struct {
	App      AppConfig
	Server   ServerConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	JWT      JWTConfig
}

type AppConfig struct {
	Env      string
	LogLevel string
}

type ServerConfig struct {
	Port string
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret      string
	ExpiryHours int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		App: AppConfig{
			Env:      getEnv("APP_ENV", "development"),
			LogLevel: getEnv("LOG_LEVEL", "info"),
		},
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
		},
		Postgres: PostgresConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "financeapp"),
			Password: getEnv("POSTGRES_PASSWORD", "financeapp123"),
			DBName:   getEnv("POSTGRES_DB", "financeapp"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", "redis123"),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:      getEnv("JWT_SECRET", "default-secret-change-in-production"),
			ExpiryHours: getEnvInt("JWT_EXPIRY_HOURS", 24),
		},
	}
}

// DSN returns a PostgreSQL connection string.
func (c *PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		c.Host, c.Port, c.User, c.Password, c.DBName,
	)
}

// Addr returns the Redis address.
func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
