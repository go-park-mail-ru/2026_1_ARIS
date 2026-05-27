package postgres

import (
	"context"
	"fmt"
	"os"

	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getConnectURL() (string, error) {
	password := os.Getenv("DB_PASSWORD")
	passwordPart := ""
	if password != "" {
		passwordPart = " password=" + password
	}

	return fmt.Sprintf("host=%s port=%s user=%s%s database=%s pool_max_conns=%s pool_max_conn_lifetime=%s pool_max_conn_idle_time=%s sslmode=%s",
			utils.EnvString("DB_HOST", "db"),
			utils.EnvString("DB_PORT", "5432"),
			os.Getenv("DB_USER"),
			passwordPart,
			utils.EnvString("DB_NAME", "ARIS-DB"),
			utils.EnvString("POOL_MAX_CONNS", "10"),
			utils.EnvString("POOL_MAX_CONN_LIFETIME", "1h"),
			utils.EnvString("POOL_MAX_CONN_IDLE_TIME", "30m"),
			utils.EnvString("SSL_MODE", "disable")),
		nil
}

func New(ctx context.Context) (*pgxpool.Pool, error) {
	confStr, err := getConnectURL()
	if err != nil {
		return nil, fmt.Errorf("failed to get db connection string: %w", err)
	}

	db, err := pgxpool.New(ctx, confStr)
	if err != nil {
		return nil, fmt.Errorf("fail to connect to db: %w", err)
	}

	err = db.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed db connection check: %w", err)
	}

	return db, nil
}
