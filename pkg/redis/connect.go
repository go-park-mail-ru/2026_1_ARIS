package redis

import (
	"fmt"
	"os"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"github.com/redis/go-redis/v9"
)

func getRedisOpts() *redis.Options {
	dialTimeout, _ := time.ParseDuration(utils.EnvString("REDIS_DIAL_TIMEOUT", "3s"))
	timeout, _ := time.ParseDuration(utils.EnvString("REDIS_TIMEOUT", "3s"))

	return &redis.Options{
		Addr:         fmt.Sprintf("%s:%s", utils.EnvString("REDIS_HOST", "redis"), utils.EnvString("REDIS_PORT", "6379")),
		Password:     os.Getenv("REDIS_USER_PASSWORD"),
		DB:           utils.EnvInt("REDIS_DB", 0),
		MaxRetries:   utils.EnvInt("REDIS_MAX_RETRIES", 3),
		DialTimeout:  dialTimeout,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	}
}
