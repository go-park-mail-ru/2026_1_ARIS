package connectdb

import (
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/config"
	"github.com/stretchr/testify/assert"
)

func TestGetConnectURL_WithoutPassword(t *testing.T) {
	env := config.EnvConfig{
		DbHost:                "localhost",
		DbPort:                "5432",
		DbUser:                "testuser",
		DbPassword:            "", // пустой пароль
		DbName:                "testdb",
		DbPoolMaxConns:        "10",
		DbPoolMaxConnLifetime: "1h",
		DbPoolMaxConnIdleTime: "30m",
		DbSSLMode:             "disable",
	}

	expected := "host=localhost port=5432 user=testuser database=testdb pool_max_conns=10 pool_max_conn_lifetime=1h pool_max_conn_idle_time=30m sslmode=disable"

	url, err := GetConnectURL(env)

	assert.NoError(t, err)
	assert.Equal(t, expected, url)
}

func TestGetConnectURL_WithPassword(t *testing.T) {
	env := config.EnvConfig{
		DbHost:                "localhost",
		DbPort:                "5432",
		DbUser:                "testuser",
		DbPassword:            "secret",
		DbName:                "testdb",
		DbPoolMaxConns:        "10",
		DbPoolMaxConnLifetime: "1h",
		DbPoolMaxConnIdleTime: "30m",
		DbSSLMode:             "require",
	}

	expected := "host=localhost port=5432 user=testuser password=secret database=testdb pool_max_conns=10 pool_max_conn_lifetime=1h pool_max_conn_idle_time=30m sslmode=require"

	url, err := GetConnectURL(env)

	assert.NoError(t, err)
	assert.Equal(t, expected, url)
}
