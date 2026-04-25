package postgres

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getConnectURL(env config.EnvConfig) (string, error) {
	if env.DbPassword == "" {
		return fmt.Sprintf("host=%s port=%s user=%s database=%s pool_max_conns=%s pool_max_conn_lifetime=%s pool_max_conn_idle_time=%s sslmode=%s", env.DbHost, env.DbPort, env.DbUser, env.DbName, env.DbPoolMaxConns, env.DbPoolMaxConnLifetime, env.DbPoolMaxConnIdleTime, env.DbSSLMode), nil
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s database=%s pool_max_conns=%s pool_max_conn_lifetime=%s pool_max_conn_idle_time=%s sslmode=%s", env.DbHost, env.DbPort, env.DbUser, env.DbPassword, env.DbName, env.DbPoolMaxConns, env.DbPoolMaxConnLifetime, env.DbPoolMaxConnIdleTime, env.DbSSLMode), nil
}

func New(ctx context.Context, envConf config.EnvConfig) (*pgxpool.Pool, error) {
	confStr, err := getConnectURL(envConf)
	if err != nil {
		return nil, fmt.Errorf("failed to get db connection string: %w", err)
	}

	db, err := pgxpool.New(ctx, confStr)
	if err != nil {
		return nil, fmt.Errorf("fail to connect to db: %w", err)
	}
	defer db.Close()

	err = db.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed db connection check: %w", err)
	}

	return db, nil
}
