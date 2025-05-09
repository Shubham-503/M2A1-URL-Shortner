package cache

import (
	"M2A1-URL-Shortner/models"
	"context"
	"encoding/json"
	"os"

	"github.com/redis/go-redis/v9"
)

type RedisURLCache interface {
	Set(key string, value models.URLShortener) error
	Get(key string) (models.URLShortener, error)
	Delete(key string) error
	Close() error
}

// RedisStore is an implementation of URLCache using Redis.
type RedisStore struct {
	Client *redis.Client
	Ctx    context.Context
}

// NewRedisStore initializes a new RedisStore instance.
func NewRedisStore(addr, password string, db int) (*RedisStore, error) {
	// Create a Redis client.
	REDIS_USERNAME := os.Getenv("REDIS_USER")

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: REDIS_USERNAME, // e.g., "localhost:6379"
		Password: password,       // leave empty if no password
		DB:       db,             // use default DB 0 or specify another one
	})

	ctx := context.Background()
	// Ping Redis to ensure connectivity.
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development" // default to development if not set
	}

	if appEnv == "development" || appEnv == "staging" {
		// Set eviction policy to LRU (allkeys-lru) explicitly.
		if err := rdb.ConfigSet(ctx, "maxmemory-policy", "allkeys-lru").Err(); err != nil {
			return nil, err
		}
	}
	return &RedisStore{
		Client: rdb,
		Ctx:    ctx,
	}, nil
}

// Set stores a value in Redis with no expiration (0 means persist indefinitely).
func (r *RedisStore) Set(key string, value models.URLShortener) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.Client.Set(r.Ctx, key, data, 0).Err()
}

// Get retrieves a value from Redis.
func (r *RedisStore) Get(key string) (models.URLShortener, error) {
	var result models.URLShortener
	data, err := r.Client.Get(r.Ctx, key).Result()
	if err != nil {
		return result, err
	}
	err = json.Unmarshal([]byte(data), &result)
	return result, err
}

// Delete removes a value from Redis.
func (r *RedisStore) Delete(key string) error {
	return r.Client.Del(r.Ctx, key).Err()
}

// Close closes the Redis C.
func (r *RedisStore) Close() error {
	return r.Client.Close()
}
