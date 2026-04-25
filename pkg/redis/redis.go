package redis

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/config"
	"github.com/redis/go-redis/v9"
)

func InitRedis(ctx context.Context, conf config.EnvConfig) (*redis.Client, error) {
	opt := GetRedisConnection(conf)

	client := redis.NewClient(opt)

	err := client.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}

	return client, nil
}
