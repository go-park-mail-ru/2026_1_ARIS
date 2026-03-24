package connectdb

import (
	"fmt"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/config"
)

func GetConnectURL(env config.EnvConfig) (string, error) {
	if env.DbPassword == "" {
		return fmt.Sprintf("host=%s port=%s user=%s database=%s pool_max_conns=%s pool_max_conn_lifetime=%s pool_max_conn_idle_time=%s sslmode=%s", env.DbHost, env.DbPort, env.DbUser, env.DbName, env.DbPoolMaxConns, env.DbPoolMaxConnLifetime, env.DbPoolMaxConnIdleTime, env.DbSSLMode), nil
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s database=%s pool_max_conns=%s pool_max_conn_lifetime=%s pool_max_conn_idle_time=%s sslmode=%s", env.DbHost, env.DbPort, env.DbUser, env.DbPassword, env.DbName, env.DbPoolMaxConns, env.DbPoolMaxConnLifetime, env.DbPoolMaxConnIdleTime, env.DbSSLMode), nil
}
