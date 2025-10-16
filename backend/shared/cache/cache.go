package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache provides a Redis-based caching layer
type Cache struct {
	client *redis.Client
	prefix string
}

// NewCache creates a new cache instance
func NewCache(client *redis.Client, prefix string) *Cache {
	return &Cache{
		client: client,
		prefix: prefix,
	}
}

// Get retrieves a value from cache and unmarshals it
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := c.client.Get(ctx, c.key(key)).Result()
	if err == redis.Nil {
		return ErrCacheMiss
	}
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}

// Set stores a value in cache with expiration
func (c *Cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, c.key(key), data, expiration).Err()
}

// Delete removes a key from cache
func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, c.key(key)).Err()
}

// DeletePattern removes all keys matching a pattern
func (c *Cache) DeletePattern(ctx context.Context, pattern string) error {
	keys, err := c.client.Keys(ctx, c.key(pattern)).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}
	return nil
}

// Exists checks if a key exists in cache
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.client.Exists(ctx, c.key(key)).Result()
	return count > 0, err
}

// GetOrSet gets a value from cache, or sets it if not found
func (c *Cache) GetOrSet(ctx context.Context, key string, dest interface{}, expiration time.Duration, setter func() (interface{}, error)) error {
	err := c.Get(ctx, key, dest)
	if err == nil {
		return nil // Cache hit
	}
	if err != ErrCacheMiss {
		return err // Redis error
	}

	// Cache miss - get value from setter
	value, err := setter()
	if err != nil {
		return err
	}

	// Store in cache for next time
	if err := c.Set(ctx, key, value, expiration); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to set cache: %v\n", err)
	}

	// Marshal the value into dest
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// Increment atomically increments a counter
func (c *Cache) Increment(ctx context.Context, key string, expiration time.Duration) (int64, error) {
	pipe := c.client.TxPipeline()
	incr := pipe.Incr(ctx, c.key(key))
	pipe.Expire(ctx, c.key(key), expiration)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

// key prefixes the key with service prefix
func (c *Cache) key(key string) string {
	return fmt.Sprintf("%s:%s", c.prefix, key)
}

// Common cache errors
var (
	ErrCacheMiss = fmt.Errorf("cache miss")
)

// Common cache durations
const (
	ShortTTL  = 5 * time.Minute  // For frequently changing data
	MediumTTL = 30 * time.Minute // For moderately stable data
	LongTTL   = 2 * time.Hour    // For stable data
	DayTTL    = 24 * time.Hour   // For daily aggregates
)
