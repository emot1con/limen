package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	client *redis.Client
	prefix string
}

func NewRedis(client *redis.Client, prefix string) Cache {
	return &redisCache{
		client: client,
		prefix: prefix,
	}
}

func (c *redisCache) Get(ctx context.Context, key string, dest any) (bool, error) {
	raw, err := c.client.Get(ctx, c.prefix+key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return false, err
	}
	return true, nil
}

func (c *redisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.prefix+key, raw, jitter(ttl)).Err()
}

func (c *redisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	prefixedKeys := make([]string, len(keys))
	for i, k := range keys {
		prefixedKeys[i] = c.prefix + k
	}
	return c.client.Del(ctx, prefixedKeys...).Err()
}

func (c *redisCache) GetVersion(ctx context.Context, group string) (int64, error) {
	val, err := c.client.Get(ctx, c.prefix+group+":ver").Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

func (c *redisCache) BumpVersion(ctx context.Context, group string) (int64, error) {
	return c.client.Incr(ctx, c.prefix+group+":ver").Result()
}
