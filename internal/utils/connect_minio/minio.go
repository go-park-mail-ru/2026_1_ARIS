package connectminio

import (
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// InitMinio подключается к Minio и создает бакет, если не существует
// Бакет - это контейнер для хранения объектов в Minio. Он представляет собой пространство имен, в котором можно хранить и организовывать файлы и папки.
func InitMinio(conf config.EnvConfig) (*minio.Client, error) {
	m := &minio.Client{}

	// Подключение к Minio с использованием имени пользователя и пароля
	client, err := minio.New(conf.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(conf.MinioRootUser, conf.MinioRootPassword, ""),
		Secure: conf.MinioUseSSL,
	})
	if err != nil {
		return nil, err
	}

	// Установка подключения Minio
	m = client

	return m, nil
}
