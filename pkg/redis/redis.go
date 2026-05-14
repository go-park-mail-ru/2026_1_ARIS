package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func InitRedis(ctx context.Context) (*redis.Client, error) {
	opt := getRedisOpts()

	client := redis.NewClient(opt)

	err := client.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}

	return client, nil
}
