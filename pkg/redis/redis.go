package redis

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"time"
)

type RedisDB struct {
	Client *redis.Client
	cfg    Config
}

func NewRedisDB(config Config) *RedisDB {
	return &RedisDB{cfg: config}
}

func (r *RedisDB) Start(ctx context.Context) error {
	address := fmt.Sprintf("%s:%s",
		r.cfg.Host,
		r.cfg.Port,
	)

	r.Client = redis.NewClient(&redis.Options{
		Addr: address,
	})

	if _, err := r.Client.Ping(ctx).Result(); err != nil {
		return err
	}

	return nil
}

func (r *RedisDB) Stop(ctx context.Context) error {
	return r.Client.Close()
}

func (r *RedisDB) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.Client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisDB) Get(ctx context.Context, key string) (string, error) {
	val, err := r.Client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return val, nil

}
