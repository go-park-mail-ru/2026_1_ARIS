package clickhouse

import (
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
)

func New() (driver.Conn, error) {
	return clickhouse.Open(&clickhouse.Options{
		Addr: []string{utils.EnvString("CLICKHOUSE_ADDR", "localhost:9000")},
		Auth: clickhouse.Auth{
			Database: utils.EnvString("CLICKHOUSE_DB", "aris"),
			Username: utils.EnvString("CLICKHOUSE_USER", "default"),
			Password: utils.EnvString("CLICKHOUSE_PASSWORD", ""),
		},
		Settings: clickhouse.Settings{
			"async_insert":          utils.EnvInt("CLICKHOUSE_ASYNC_INSERT", 1),
			"wait_for_async_insert": utils.EnvInt("CLICKHOUSE_WAIT_FOR_ASYNC_INSERT", 0),
			"max_insert_block_size": utils.EnvInt("CLICKHOUSE_MAX_INSERT_BLOCK_SIZE", 100000),
		},
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})
}
