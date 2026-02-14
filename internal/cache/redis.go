package cache

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient() (*RedisClient, error) {
	// 1. Connect to Redis (Using Docker Compose default)
	host := getEnv("REDIS_HOST", "localhost")
	port := getEnv("REDIS_PORT", "6379")

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: "", // No password set in local dev
		DB:       0,  // Default DB
	})

	// 2. Ping to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{client: rdb}, nil
}

// IncrementAndCheck atomically increments the download count and checks if it exceeds the max.
func (r *RedisClient) IncrementAndCheck(ctx context.Context, dropID string, maxDownloads int) (bool, error) {
	// The Lua Script
	// KEYS[1] = The Redis key for this drop's counter (e.g., "drop:123:downloads")
	// ARGV[1] = The maximum allowed downloads
	// Logic: Increment the key. If it's 1 (first time), set an expiry so Redis doesn't fill up with junk.
	// Then, check if the new value is greater than the max.
	script := redis.NewScript(`
		local key = KEYS[1]
		local max_downloads = tonumber(ARGV[1])
		
		local current = redis.call("INCR", key)
		
		-- If this is the first download, set the counter to expire in 24 hours to save memory
		if current == 1 then
			redis.call("EXPIRE", key, 86400)
		end
		
		if current > max_downloads then
			return 0 -- Failed / Rejected
		end
		
		return 1 -- Success / Allowed
	`)

	key := fmt.Sprintf("drop:%s:downloads", dropID)
	
	// Run the script atomically
	result, err := script.Run(ctx, r.client, []string{key}, maxDownloads).Result()
	if err != nil {
		return false, fmt.Errorf("redis script error: %w", err)
	}

	// 1 means allowed, 0 means rejected
	return result.(int64) == 1, nil
}

// Helper to get env vars
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}