package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

var ErrCacheMiss = errors.New("cache miss")

type CacheConfig struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Prefix       string
	DefaultTTL   time.Duration
}

func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     100,
		MinIdleConns: 10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		Prefix:       "biometrics:",
		DefaultTTL:   5 * time.Minute,
	}
}

type RedisCache struct {
	client  *redis.Client
	config  CacheConfig
	logger  *zap.Logger
	ctx     context.Context
	metrics *CacheMetrics
}

func NewRedisCache(config CacheConfig, logger *zap.Logger) (*RedisCache, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	ctx := context.Background()

	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	cache := &RedisCache{
		client:  client,
		config:  config,
		logger:  logger,
		ctx:     ctx,
		metrics: NewCacheMetrics(),
	}

	logger.Info("Redis cache initialized",
		zap.String("addr", config.Addr),
		zap.Int("pool_size", config.PoolSize),
	)

	return cache, nil
}

func (c *RedisCache) key(suffix string) string {
	return c.config.Prefix + suffix
}

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	start := time.Now()

	data, err := c.client.Get(ctx, c.key(key)).Bytes()

	c.metrics.RecordGet(time.Since(start), err == nil)

	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		c.logger.Error("Cache get error", zap.String("key", key), zap.Error(err))
		return nil, err
	}

	return data, nil
}

func (c *RedisCache) GetJSON(ctx context.Context, key string, value interface{}) error {
	data, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	if data == nil {
		return ErrCacheMiss
	}
	return json.Unmarshal(data, value)
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl == 0 {
		ttl = c.config.DefaultTTL
	}

	start := time.Now()
	err := c.client.Set(ctx, c.key(key), value, ttl).Err()
	c.metrics.RecordSet(time.Since(start), err == nil)

	if err != nil {
		c.logger.Error("Cache set error", zap.String("key", key), zap.Error(err))
		return err
	}

	return nil
}

func (c *RedisCache) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return c.Set(ctx, key, data, ttl)
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	start := time.Now()
	err := c.client.Del(ctx, c.key(key)).Err()
	c.metrics.RecordDelete(time.Since(start))

	if err != nil {
		c.logger.Error("Cache delete error", zap.String("key", key), zap.Error(err))
		return err
	}

	return nil
}

func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, c.key(key)).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

func (c *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, c.key(key)).Result()
}

func (c *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, c.key(key), ttl).Err()
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}

func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *RedisCache) Flush(ctx context.Context) error {
	return c.client.FlushAll(ctx).Err()
}

func (c *RedisCache) GetClient() *redis.Client {
	return c.client
}

func (c *RedisCache) GetMetrics() *CacheMetrics {
	return c.metrics
}
