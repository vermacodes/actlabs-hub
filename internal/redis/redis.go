package redis

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(ctx context.Context) (*redis.Client, error) {
	// Get Redis hostname from environment variable, default to localhost
	redisHostname := os.Getenv("REDIS_HOSTNAME")
	if redisHostname == "" {
		redisHostname = "localhost"
	}

	// Get Redis port from environment variable, default to 6379
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	// Get Redis password from environment variable
	redisPassword := os.Getenv("REDIS_PASSWORD")

	// Get Redis DB from environment variable, default to 0
	redisDB := 0
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		if db, err := strconv.Atoi(dbStr); err == nil {
			redisDB = db
		}
	}

	addr := redisHostname + ":" + redisPort

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: redisPassword,
		DB:       redisDB,
	})

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %v", err)
	}

	return client, nil
}
