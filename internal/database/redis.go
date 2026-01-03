package database

import (
	"aigentools-backend/config"
	"context"

	"github.com/go-redis/redis/v8"
)

var (
	RedisClient *redis.Client
	Ctx         = context.Background()
)

func ConnectRedis(cfg *config.Config) error {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     cfg.RedisFullAddr(),
		Password: cfg.RedisPassword,
		DB:       0, // use default DB
	})

	_, err := RedisClient.Ping(Ctx).Result()
	return err
}
