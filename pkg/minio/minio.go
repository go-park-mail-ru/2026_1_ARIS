package minio

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/config"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
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

func GenerateMediaName(mediaUUID uuid.UUID, mediaSize int64, extension string) string {
	return fmt.Sprintf("%s-%d%s", mediaUUID.String(), mediaSize, extension)
}

func New(ctx context.Context, envConf config.EnvConfig, logger *zap.Logger) (*minio.Client, error) {
	// создание MinIO клиента
	minioClient, err := InitMinio(envConf)
	if err != nil {
		return nil, fmt.Errorf("fail to initialize MinIO: %w", err)
	}

	// Проверка на существование бакета
	exists, err := minioClient.BucketExists(ctx, envConf.MinioBucketName)
	if err != nil {
		return nil, fmt.Errorf("fail to chech MinIO bucket existition: %w", err)
	}

	logger.Info("Successfully connected to MinIO")

	// Если бакета нет - его нужно создать
	if !exists {
		err := minioClient.MakeBucket(ctx, envConf.MinioBucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("fail to create MinIO buchet: %w", err)
		}
		logger.Info(("MinIO bucket created"))
	}

	// Устанавливаем политику доступа к файлам
	policy := map[string]any{
		"Version": "2012-10-17",
		"Statement": []any{
			map[string]any{
				"Effect": "Allow",
				"Principal": map[string]string{
					"AWS": "*",
				},
				"Action": []string{
					"s3:GetBucketLocation",
					"s3:ListBucket",
				},
				"Resource": "arn:aws:s3:::" + envConf.MinioBucketName,
			},
			map[string]any{
				"Effect": "Allow",
				"Principal": map[string]string{
					"AWS": "*",
				},
				"Action":   "s3:GetObject",
				"Resource": "arn:aws:s3:::" + envConf.MinioBucketName + "/*",
			},
		},
	}

	rawPolicy, err := json.Marshal(policy)
	if err != nil {
		return nil, fmt.Errorf("fail to marshal MinIO policy: %w", err)
	}

	err = minioClient.SetBucketPolicy(ctx, envConf.MinioBucketName, string(rawPolicy))
	if err != nil {
		return nil, fmt.Errorf("fail to set MinIO bucket policy: %w", err)
	}

	return minioClient, nil
}
