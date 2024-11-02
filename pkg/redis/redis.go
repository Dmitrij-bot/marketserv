package redis

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
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
