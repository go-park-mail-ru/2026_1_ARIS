package connectredis

import (
	"context"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/config"
	"github.com/stretchr/testify/require"
)

func TestGetRedisConnection(t *testing.T) {
	cfg := config.EnvConfig{
		RedisAddr:        "127.0.0.1:6379",
		RedisPassword:    "secret",
		RedisDB:          3,
		RedisMaxRetries:  4,
		RedisDialTimeout: 2 * time.Second,
		RedisTimeout:     5 * time.Second,
	}

	opts := GetRedisConnection(cfg)
	require.Equal(t, "127.0.0.1:6379", opts.Addr)
	require.Equal(t, "secret", opts.Password)
	require.Equal(t, 3, opts.DB)
	require.Equal(t, 4, opts.MaxRetries)
	require.Equal(t, 2*time.Second, opts.DialTimeout)
	require.Equal(t, 5*time.Second, opts.ReadTimeout)
	require.Equal(t, 5*time.Second, opts.WriteTimeout)
}

func TestInitRedis_Error(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	client, err := InitRedis(ctx, config.EnvConfig{
		RedisAddr:        "127.0.0.1:1",
		RedisDialTimeout: 100 * time.Millisecond,
		RedisTimeout:     100 * time.Millisecond,
	})

	require.Error(t, err)
	require.Nil(t, client)
}

