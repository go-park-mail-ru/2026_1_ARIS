package connectredis

import (
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/config"
	"github.com/redis/go-redis/v9"
)

func GetRedisConnection(conf config.EnvConfig) *redis.Options {
	return &redis.Options{
		Addr: conf.RedisAddr,
		//Username:     conf.RedisUsername,
		Password:     conf.RedisPassword,
		DB:           conf.RedisDB,
		MaxRetries:   conf.RedisMaxRetries,
		DialTimeout:  conf.RedisDialTimeout,
		ReadTimeout:  conf.RedisTimeout,
		WriteTimeout: conf.RedisTimeout,
	}
}
