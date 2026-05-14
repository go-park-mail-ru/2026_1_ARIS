package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetConnectURL_WithoutPassword(t *testing.T) {
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "testuser")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_NAME", "testdb")
	t.Setenv("POOL_MAX_CONNS", "10")
	t.Setenv("POOL_MAX_CONN_LIFETIME", "1h")
	t.Setenv("POOL_MAX_CONN_IDLE_TIME", "30m")
	t.Setenv("SSL_MODE", "disable")

	expected := "host=localhost port=5432 user=testuser database=testdb pool_max_conns=10 pool_max_conn_lifetime=1h pool_max_conn_idle_time=30m sslmode=disable"

	url, err := getConnectURL()

	assert.NoError(t, err)
	assert.Equal(t, expected, url)
}

func TestGetConnectURL_WithPassword(t *testing.T) {
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "testuser")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "testdb")
	t.Setenv("POOL_MAX_CONNS", "10")
	t.Setenv("POOL_MAX_CONN_LIFETIME", "1h")
	t.Setenv("POOL_MAX_CONN_IDLE_TIME", "30m")
	t.Setenv("SSL_MODE", "require")

	expected := "host=localhost port=5432 user=testuser password=secret database=testdb pool_max_conns=10 pool_max_conn_lifetime=1h pool_max_conn_idle_time=30m sslmode=require"

	url, err := getConnectURL()

	assert.NoError(t, err)
	assert.Equal(t, expected, url)
}
