package connectminio

import (
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/config"
	"github.com/stretchr/testify/require"
)

func TestInitMinio(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, err := InitMinio(config.EnvConfig{
			MinioEndpoint:     "127.0.0.1:9000",
			MinioRootUser:     "minio",
			MinioRootPassword: "password",
			MinioUseSSL:       false,
		})
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("invalid endpoint", func(t *testing.T) {
		client, err := InitMinio(config.EnvConfig{
			MinioEndpoint:     "",
			MinioRootUser:     "minio",
			MinioRootPassword: "password",
		})
		require.Error(t, err)
		require.Nil(t, client)
	})
}

