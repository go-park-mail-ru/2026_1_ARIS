package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "aris")
	t.Setenv("POOL_MAX_CONNS", "10")
	t.Setenv("POOL_MAX_CONN_LIFETIME", "1h")
	t.Setenv("POOL_MAX_CONN_IDLE_TIME", "30m")
	t.Setenv("SSL_MODE", "disable")
	t.Setenv("MINIO_PORT", "9000")
	t.Setenv("MINIO_ENDPOINT", "127.0.0.1:9000")
	t.Setenv("MINIO_BUCKET_NAME", "bucket")
	t.Setenv("MINIO_ROOT_USER", "minio")
	t.Setenv("MINIO_ROOT_PASSWORD", "password")
	t.Setenv("MINIO_USE_SSL", "false")
	t.Setenv("REDIS_ADDR", "127.0.0.1:6379")
	t.Setenv("REDIS_HOST", "127.0.0.1")
	t.Setenv("REDIS_PORT", "6379")
	t.Setenv("REDIS_USER", "default")
	t.Setenv("REDIS_USER_PASSWORD", "redis-pass")
	t.Setenv("REDIS_DB", "2")
	t.Setenv("REDIS_MAX_RETRIES", "5")
	t.Setenv("REDIS_DIAL_TIMEOUT", "2s")
	t.Setenv("REDIS_TIMEOUT", "3s")

	cfg, err := NewConfig()
	require.NoError(t, err)
	require.Equal(t, "localhost", cfg.DbHost)
	require.Equal(t, "5432", cfg.DbPort)
	require.Equal(t, "postgres", cfg.DbUser)
	require.Equal(t, "secret", cfg.DbPassword)
	require.Equal(t, "aris", cfg.DbName)
	require.Equal(t, "10", cfg.DbPoolMaxConns)
	require.Equal(t, "1h", cfg.DbPoolMaxConnLifetime)
	require.Equal(t, "30m", cfg.DbPoolMaxConnIdleTime)
	require.Equal(t, "disable", cfg.DbSSLMode)
	require.Equal(t, "127.0.0.1:9000", cfg.MinioEndpoint)
	require.Equal(t, "bucket", cfg.MinioBucketName)
	require.False(t, cfg.MinioUseSSL)
	require.Equal(t, "127.0.0.1:6379", cfg.RedisAddr)
	require.Equal(t, 2, cfg.RedisDB)
	require.Equal(t, 5, cfg.RedisMaxRetries)
	require.Equal(t, 2*time.Second, cfg.RedisDialTimeout)
	require.Equal(t, 3*time.Second, cfg.RedisTimeout)
}

