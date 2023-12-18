package redis

import (
	"github.com/go-redis/redis"
	"github.com/go-redis/redis_rate"
)

func NewRateLimiter(rdb *redis.Client) *redis_rate.Limiter {
	return redis_rate.NewLimiter(rdb)
}
