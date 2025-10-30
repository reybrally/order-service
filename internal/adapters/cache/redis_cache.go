package cache

import (
	"context"
	"encoding/json"
	"time"

	redis "github.com/redis/go-redis/v9"
	domain "github.com/reybrally/order-service/internal/domain/order"
)

type RedisCache struct {
	rdb    *redis.Client
	prefix string
	ttl    time.Duration
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	Prefix   string
	TTL      time.Duration
}

func NewRedisCache(cfg RedisConfig) *RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return &RedisCache{
		rdb:    rdb,
		prefix: cfg.Prefix,
		ttl:    cfg.TTL,
	}
}

func (c *RedisCache) makeKey(id string) string {
	return c.prefix + id
}

func (c *RedisCache) Set(id string, o domain.Order) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	data, err := json.Marshal(o)
	if err != nil {
		return err
	}

	return c.rdb.Set(ctx, c.makeKey(id), data, c.ttl).Err()
}

func (c *RedisCache) Get(id string) (domain.Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	b, err := c.rdb.Get(ctx, c.makeKey(id)).Bytes()
	if err != nil {
		return domain.Order{}, err
	}

	var o domain.Order
	if err := json.Unmarshal(b, &o); err != nil {
		return domain.Order{}, err
	}
	return o, nil
}

func (c *RedisCache) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return c.rdb.Del(ctx, c.makeKey(id)).Err()
}

func (c *RedisCache) Close() error {
	return c.rdb.Close()
}
