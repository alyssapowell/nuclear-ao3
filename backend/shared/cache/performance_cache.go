// Nuclear AO3: High-Performance Caching Layer
// Optimized for handling millions of works with sub-100ms response times

package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/patrickmn/go-cache"
)

// PerformanceCache implements a multi-layer caching strategy
type PerformanceCache struct {
	redis      *redis.Client
	localCache *cache.Cache
	config     *CacheConfig
}

// CacheConfig defines performance-optimized cache settings
type CacheConfig struct {
	// Redis settings
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
	RedisPoolSize     int
	RedisMinIdleConns int
	RedisMaxConnAge   time.Duration

	// Local cache settings
	LocalCacheSize       int
	LocalCacheExpiration time.Duration
	LocalCacheCleanup    time.Duration

	// Performance settings
	EnableCompression bool
	EnablePipeline    bool
	BatchSize         int
	ConnectionTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
}

// DefaultPerformanceConfig returns optimized cache configuration
func DefaultPerformanceConfig() *CacheConfig {
	return &CacheConfig{
		RedisAddr:         "localhost:6379",
		RedisPassword:     "",
		RedisDB:           0,
		RedisPoolSize:     50, // High connection pool for performance
		RedisMinIdleConns: 10,
		RedisMaxConnAge:   30 * time.Minute,

		LocalCacheSize:       10000, // 10k items in local cache
		LocalCacheExpiration: 5 * time.Minute,
		LocalCacheCleanup:    10 * time.Minute,

		EnableCompression: true,
		EnablePipeline:    true,
		BatchSize:         100,
		ConnectionTimeout: 5 * time.Second,
		ReadTimeout:       3 * time.Second,
		WriteTimeout:      3 * time.Second,
	}
}

// NewPerformanceCache creates a new high-performance cache instance
func NewPerformanceCache(config *CacheConfig) (*PerformanceCache, error) {
	// Configure Redis with performance optimizations
	rdb := redis.NewClient(&redis.Options{
		Addr:         config.RedisAddr,
		Password:     config.RedisPassword,
		DB:           config.RedisDB,
		PoolSize:     config.RedisPoolSize,
		MinIdleConns: config.RedisMinIdleConns,
		MaxConnAge:   config.RedisMaxConnAge,
		DialTimeout:  config.ConnectionTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,

		// Performance optimizations
		PoolTimeout:     4 * time.Second,
		IdleTimeout:     5 * time.Minute,
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Create local cache for ultra-fast access
	localCache := cache.New(config.LocalCacheExpiration, config.LocalCacheCleanup)

	return &PerformanceCache{
		redis:      rdb,
		localCache: localCache,
		config:     config,
	}, nil
}

// Get retrieves a value from cache (local first, then Redis)
func (pc *PerformanceCache) Get(ctx context.Context, key string) ([]byte, error) {
	// Check local cache first (sub-millisecond access)
	if data, found := pc.localCache.Get(key); found {
		return data.([]byte), nil
	}

	// Check Redis cache
	data, err := pc.redis.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("redis get error: %w", err)
	}

	// Store in local cache for next access
	pc.localCache.Set(key, data, cache.DefaultExpiration)

	return data, nil
}

// Set stores a value in both local and Redis cache
func (pc *PerformanceCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// Serialize value
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	// Store in Redis
	if err := pc.redis.Set(ctx, key, data, expiration).Err(); err != nil {
		return fmt.Errorf("failed to set in Redis: %w", err)
	}

	// Store in local cache
	pc.localCache.Set(key, data, cache.DefaultExpiration)

	return nil
}

// GetJSON retrieves and deserializes a JSON value from cache
func (pc *PerformanceCache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := pc.Get(ctx, key)
	if err != nil {
		return err
	}
	if data == nil {
		return redis.Nil // Cache miss
	}

	return json.Unmarshal(data, dest)
}

// SetJSON serializes and stores a JSON value in cache
func (pc *PerformanceCache) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return pc.Set(ctx, key, value, expiration)
}

// MGet retrieves multiple values efficiently using pipeline
func (pc *PerformanceCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte)
	localHits := make(map[string]bool)

	// Check local cache first
	for _, key := range keys {
		if data, found := pc.localCache.Get(key); found {
			result[key] = data.([]byte)
			localHits[key] = true
		}
	}

	// Get remaining keys from Redis
	var redisKeys []string
	for _, key := range keys {
		if !localHits[key] {
			redisKeys = append(redisKeys, key)
		}
	}

	if len(redisKeys) > 0 {
		// Use pipeline for efficient batch retrieval
		pipe := pc.redis.Pipeline()
		cmds := make(map[string]*redis.StringCmd)

		for _, key := range redisKeys {
			cmds[key] = pipe.Get(ctx, key)
		}

		_, err := pipe.Exec(ctx)
		if err != nil && err != redis.Nil {
			return nil, fmt.Errorf("pipeline exec error: %w", err)
		}

		// Process results
		for key, cmd := range cmds {
			data, err := cmd.Bytes()
			if err == nil {
				result[key] = data
				// Store in local cache
				pc.localCache.Set(key, data, cache.DefaultExpiration)
			}
		}
	}

	return result, nil
}

// MSet stores multiple values efficiently using pipeline
func (pc *PerformanceCache) MSet(ctx context.Context, keyValues map[string]interface{}, expiration time.Duration) error {
	if len(keyValues) == 0 {
		return nil
	}

	// Use pipeline for efficient batch storage
	pipe := pc.redis.Pipeline()

	for key, value := range keyValues {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to serialize value for key %s: %w", key, err)
		}

		pipe.Set(ctx, key, data, expiration)

		// Store in local cache
		pc.localCache.Set(key, data, cache.DefaultExpiration)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("pipeline exec error: %w", err)
	}

	return nil
}

// Delete removes a value from both caches
func (pc *PerformanceCache) Delete(ctx context.Context, key string) error {
	// Remove from local cache
	pc.localCache.Delete(key)

	// Remove from Redis
	return pc.redis.Del(ctx, key).Err()
}

// InvalidatePattern removes all keys matching a pattern
func (pc *PerformanceCache) InvalidatePattern(ctx context.Context, pattern string) error {
	// Get all matching keys
	keys, err := pc.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get keys for pattern %s: %w", pattern, err)
	}

	if len(keys) == 0 {
		return nil
	}

	// Delete from Redis
	if err := pc.redis.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("failed to delete keys: %w", err)
	}

	// Clear local cache (pattern matching would be expensive)
	pc.localCache.Flush()

	return nil
}

// GetStats returns cache performance statistics
func (pc *PerformanceCache) GetStats(ctx context.Context) (map[string]interface{}, error) {
	// Redis stats
	info, err := pc.redis.Info(ctx, "memory", "stats").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}

	// Local cache stats
	localStats := map[string]interface{}{
		"item_count": pc.localCache.ItemCount(),
	}

	return map[string]interface{}{
		"redis_info":  info,
		"local_cache": localStats,
		"config":      pc.config,
	}, nil
}

// Close closes all cache connections
func (pc *PerformanceCache) Close() error {
	if pc.redis != nil {
		return pc.redis.Close()
	}
	return nil
}

// Cache key builders for different data types
type CacheKeys struct{}

func (CacheKeys) WorkSearch(query string, filters map[string]interface{}) string {
	return fmt.Sprintf("search:works:%s:%x", query, hashFilters(filters))
}

func (CacheKeys) TagAutocomplete(query, category string) string {
	return fmt.Sprintf("tags:autocomplete:%s:%s", category, query)
}

func (CacheKeys) WorkDetails(workID string) string {
	return fmt.Sprintf("work:details:%s", workID)
}

func (CacheKeys) UserBookmarks(userID string, page int) string {
	return fmt.Sprintf("user:bookmarks:%s:%d", userID, page)
}

func (CacheKeys) PopularTags(category string, timeRange string) string {
	return fmt.Sprintf("tags:popular:%s:%s", category, timeRange)
}

func (CacheKeys) SearchFacets(query string) string {
	return fmt.Sprintf("search:facets:%s", query)
}

// hashFilters creates a consistent hash for filter combinations
func hashFilters(filters map[string]interface{}) uint32 {
	// Simple hash implementation for filter combinations
	// In production, use a proper hash function
	var hash uint32 = 0
	for k, v := range filters {
		hash += uint32(len(k)) + uint32(len(fmt.Sprintf("%v", v)))
	}
	return hash
}
