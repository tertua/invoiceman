package cache

import (
	"github.com/tertua/tupay/pkg/configs"

	"github.com/redis/go-redis/v9"
)

// RedisConnection func for connect to Redis server.
func RedisConnection() (*redis.Client, error) {
	cfg := configs.Get().Redis

	// Set Redis options.
	options := &redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DBNumber,
	}

	return redis.NewClient(options), nil
}
