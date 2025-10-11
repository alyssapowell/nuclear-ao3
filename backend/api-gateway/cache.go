package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// =============================================================================
// RATE LIMITER
// =============================================================================

// RateLimiter handles rate limiting using Redis sliding window
type RateLimiter struct {
	redis       *redis.Client
	defaultRate int           // requests per window
	window      time.Duration // time window
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rdb *redis.Client) *RateLimiter {
	return &RateLimiter{
		redis:       rdb,
		defaultRate: 1000,        // 1000 requests per window
		window:      time.Minute, // 1 minute window
	}
}

// CheckLimit checks if a client has exceeded their rate limit
func (rl *RateLimiter) CheckLimit(ctx context.Context, clientID string) (allowed bool, remaining int, resetTime time.Time) {
	if rl.redis == nil {
		// No Redis connection, allow all requests
		return true, rl.defaultRate, time.Now().Add(rl.window)
	}

	key := fmt.Sprintf("rate_limit:%s", clientID)
	now := time.Now()
	windowStart := now.Add(-rl.window)

	pipe := rl.redis.Pipeline()

	// Remove old entries outside the window
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))

	// Count current requests in window
	pipe.ZCard(ctx, key)

	// Add current request
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixNano()), Member: now.UnixNano()})

	// Set expiration
	pipe.Expire(ctx, key, rl.window)

	results, err := pipe.Exec(ctx)
	if err != nil {
		// On error, allow the request
		return true, rl.defaultRate, now.Add(rl.window)
	}

	// Get count result
	if len(results) < 2 {
		return true, rl.defaultRate, now.Add(rl.window)
	}

	count, err := results[1].(*redis.IntCmd).Result()
	if err != nil {
		return true, rl.defaultRate, now.Add(rl.window)
	}

	remaining = rl.defaultRate - int(count)
	if remaining < 0 {
		remaining = 0
	}

	resetTime = now.Add(rl.window)
	allowed = int(count) <= rl.defaultRate

	return allowed, remaining, resetTime
}

// CheckLimitWithConfig checks rate limit with custom configuration
func (rl *RateLimiter) CheckLimitWithConfig(ctx context.Context, clientID string, requests int, window time.Duration) (allowed bool, remaining int, resetTime time.Time) {
	if rl.redis == nil {
		// No Redis connection, allow all requests
		return true, requests, time.Now().Add(window)
	}

	key := fmt.Sprintf("rate_limit:%s", clientID)
	now := time.Now()
	windowStart := now.Add(-window)

	pipe := rl.redis.Pipeline()

	// Remove old entries outside the window
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))

	// Count current requests in window
	pipe.ZCard(ctx, key)

	// Add current request
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixNano()), Member: now.UnixNano()})

	// Set expiration
	pipe.Expire(ctx, key, window)

	results, err := pipe.Exec(ctx)
	if err != nil {
		// On error, allow the request
		return true, requests, now.Add(window)
	}

	// Get count result
	if len(results) < 2 {
		return true, requests, now.Add(window)
	}

	count, err := results[1].(*redis.IntCmd).Result()
	if err != nil {
		return true, requests, now.Add(window)
	}

	remaining = requests - int(count)
	if remaining < 0 {
		remaining = 0
	}

	resetTime = now.Add(window)
	allowed = int(count) <= requests

	return allowed, remaining, resetTime
}

// =============================================================================
// CACHE MANAGER
// =============================================================================

// CacheManager handles caching of API responses
type CacheManager struct {
	redis   *redis.Client
	metrics *GatewayMetrics
}

// NewCacheManager creates a new cache manager
func NewCacheManager(rdb *redis.Client) *CacheManager {
	return &CacheManager{
		redis: rdb,
	}
}

// SetMetrics sets the metrics collector
func (cm *CacheManager) SetMetrics(metrics *GatewayMetrics) {
	cm.metrics = metrics
}

// Get retrieves a value from cache
func (cm *CacheManager) Get(ctx context.Context, key string) ([]byte, error) {
	if cm.redis == nil {
		if cm.metrics != nil {
			cm.metrics.RecordCacheOperation("get", "miss")
		}
		return nil, fmt.Errorf("no redis connection")
	}

	val, err := cm.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			if cm.metrics != nil {
				cm.metrics.RecordCacheOperation("get", "miss")
			}
			return nil, nil // Cache miss
		}
		if cm.metrics != nil {
			cm.metrics.RecordCacheOperation("get", "error")
		}
		return nil, err
	}

	if cm.metrics != nil {
		cm.metrics.RecordCacheOperation("get", "hit")
	}
	return []byte(val), nil
}

// Set stores a value in cache with TTL
func (cm *CacheManager) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if cm.redis == nil {
		if cm.metrics != nil {
			cm.metrics.RecordCacheOperation("set", "error")
		}
		return fmt.Errorf("no redis connection")
	}

	err := cm.redis.Set(ctx, key, value, ttl).Err()
	if err != nil {
		if cm.metrics != nil {
			cm.metrics.RecordCacheOperation("set", "error")
		}
		return err
	}

	if cm.metrics != nil {
		cm.metrics.RecordCacheOperation("set", "success")
	}
	return nil
}

// SetJSON stores a JSON-serializable value in cache
func (cm *CacheManager) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		if cm.metrics != nil {
			cm.metrics.RecordCacheOperation("set_json", "error")
		}
		return err
	}
	return cm.Set(ctx, key, data, ttl)
}

// GetJSON retrieves and unmarshals a JSON value from cache
func (cm *CacheManager) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := cm.Get(ctx, key)
	if err != nil {
		return err
	}
	if data == nil {
		return nil // Cache miss
	}

	err = json.Unmarshal(data, dest)
	if err != nil {
		if cm.metrics != nil {
			cm.metrics.RecordCacheOperation("get_json", "error")
		}
		return err
	}

	return nil
}

// Delete removes a key from cache
func (cm *CacheManager) Delete(ctx context.Context, key string) error {
	if cm.redis == nil {
		return fmt.Errorf("no redis connection")
	}

	err := cm.redis.Del(ctx, key).Err()
	if err != nil {
		if cm.metrics != nil {
			cm.metrics.RecordCacheOperation("delete", "error")
		}
		return err
	}

	if cm.metrics != nil {
		cm.metrics.RecordCacheOperation("delete", "success")
	}
	return nil
}

// ClearPattern removes all keys matching a pattern
func (cm *CacheManager) ClearPattern(ctx context.Context, pattern string) error {
	if cm.redis == nil {
		return fmt.Errorf("no redis connection")
	}

	keys, err := cm.redis.Keys(ctx, pattern).Result()
	if err != nil {
		if cm.metrics != nil {
			cm.metrics.RecordCacheOperation("clear_pattern", "error")
		}
		return err
	}

	if len(keys) > 0 {
		err = cm.redis.Del(ctx, keys...).Err()
		if err != nil {
			if cm.metrics != nil {
				cm.metrics.RecordCacheOperation("clear_pattern", "error")
			}
			return err
		}
	}

	if cm.metrics != nil {
		cm.metrics.RecordCacheOperation("clear_pattern", "success")
	}
	return nil
}

// GenerateCacheKey creates a standardized cache key
func (cm *CacheManager) GenerateCacheKey(prefix, identifier string, params ...string) string {
	key := fmt.Sprintf("%s:%s", prefix, identifier)
	for _, param := range params {
		key += ":" + param
	}
	return key
}
