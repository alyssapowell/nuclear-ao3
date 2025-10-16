package database

import (
	"context"
	"database/sql"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// Config holds database and Redis configuration
type Config struct {
	DatabaseURL   string
	RedisURL      string
	RedisDB       int
	RedisPassword string
}

// Connection holds database and Redis connections
type Connection struct {
	DB    *sql.DB
	Redis *redis.Client
}

// NewConnection creates new database and Redis connections
func NewConnection(config Config) (*Connection, error) {
	// Database connection
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		return nil, err
	}

	// Test database connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	// Optimized connection pool settings for resource efficiency
	// Total connections across all services should not exceed DB limits
	db.SetMaxOpenConns(10)                  // Reduced for single-server deployment
	db.SetMaxIdleConns(3)                   // Keep fewer idle connections
	db.SetConnMaxLifetime(30 * time.Minute) // Shorter lifetime for better resource cleanup
	db.SetConnMaxIdleTime(5 * time.Minute)  // Close idle connections faster

	// Optimized Redis connection pool
	rdb := redis.NewClient(&redis.Options{
		Addr:            config.RedisURL,
		Password:        config.RedisPassword,
		DB:              config.RedisDB,
		PoolSize:        15, // Increased for caching
		MinIdleConns:    5,  // More idle connections for caching
		MaxRetries:      3,
		PoolTimeout:     30 * time.Second, // Pool timeout
		ConnMaxIdleTime: 5 * time.Minute,  // Close idle connections
		ConnMaxLifetime: 30 * time.Minute, // Connection max age
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		db.Close()
		rdb.Close()
		return nil, err
	}

	return &Connection{
		DB:    db,
		Redis: rdb,
	}, nil
}

// Close closes both database and Redis connections
func (c *Connection) Close() {
	if c.DB != nil {
		c.DB.Close()
	}
	if c.Redis != nil {
		c.Redis.Close()
	}
}

// GetEnv gets environment variable with fallback
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// DefaultConfig returns default database configuration
func DefaultConfig(serviceName string, redisDB int) Config {
	return Config{
		DatabaseURL:   GetEnv("DATABASE_URL", "postgres://ao3_user:ao3_password@localhost/ao3_nuclear?sslmode=disable"),
		RedisURL:      GetEnv("REDIS_URL", "localhost:6379"),
		RedisDB:       redisDB,
		RedisPassword: GetEnv("REDIS_PASSWORD", ""),
	}
}
