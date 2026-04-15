package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type EnvConfig struct {
	DbHost                string `env:"DB_HOST"`
	DbPort                string `env:"DB_PORT"`
	DbUser                string `env:"DB_USER"`
	DbPassword            string `env:"DB_PASSWORD"`
	DbName                string `env:"DB_NAME"`
	DbPoolMaxConns        string `env:"POOL_MAX_CONNS"`
	DbPoolMaxConnLifetime string `env:"POOL_MAX_CONN_LIFETIME"`
	DbPoolMaxConnIdleTime string `env:"POOL_MAX_CONN_IDLE_TIME"`
	DbSSLMode             string `env:"SSL_MODE"`

	MinioPort         string `env:"MINIO_PORT"`
	MinioEndpoint     string `env:"MINIO_ENDPOINT"`
	MinioBucketName   string `env:"MINIO_BUCKET_NAME"`
	MinioRootUser     string `env:"MINIO_ROOT_USER"`
	MinioRootPassword string `env:"MINIO_ROOT_PASSWORD"`
	MinioUseSSL       bool   `env:"MINIO_USE_SSL"`

	RedisAddr        string        `env:"REDIS_ADDR"`
	RedisHost        string        `env:"REDIS_HOST"`
	RedisPort        string        `env:"REDIS_PORT"`
	RedisUsername    string        `env:"REDIS_USER"`
	RedisPassword    string        `env:"REDIS_USER_PASSWORD"`
	RedisDB          int           `env:"REDIS_DB"`
	RedisMaxRetries  int           `env:"REDIS_MAX_RETRIES"`
	RedisDialTimeout time.Duration `env:"REDIS_DIAL_TIMEOUT"`
	RedisTimeout     time.Duration `env:"REDIS_TIMEOUT"`
}

// нужно будет прокинуть логгер
func NewConfig() (EnvConfig, error) {
	var config EnvConfig

	err := env.Parse(&config)
	if err != nil {
		fmt.Println("error conf", err)
		return EnvConfig{}, err
	}

	return config, nil
}
