package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(ctx context.Context) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %v", err)
	}

	return client, nil
}
