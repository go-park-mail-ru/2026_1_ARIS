package media

//go:generate mockgen -destination=../mocks/s3_repo_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/media S3Repo

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	xminio "github.com/go-park-mail-ru/2026_1_ARIS/pkg/minio"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

type MinioClient struct {
	client *minio.Client
	// logger
}

func NewMinioClient(client *minio.Client) S3Repo {
	return &MinioClient{
		client: client,
	}
}

type S3Repo interface {
	Save(ctx context.Context, bucketName string, reader io.Reader, mediaUUID uuid.UUID, size int64, extension string, opts minio.PutObjectOptions) (string, error)
}

func (client *MinioClient) Save(ctx context.Context, bucketName string, reader io.Reader, mediaUUID uuid.UUID, size int64, extension string, opts minio.PutObjectOptions) (string, error) {
	logger := logger.FromContext(ctx)
	objectName := xminio.GenerateMediaName(mediaUUID, size, extension)

	start := time.Now()
	uploadInto, err := client.client.PutObject(ctx, bucketName, objectName, reader, size, opts)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "GetUserByID"),
			zap.Duration("duration_ms", time.Since(start)),
		)
	}
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	return fmt.Sprintf("http://arisnet.ru/%s/%s",
		uploadInto.Bucket,
		uploadInto.Key,
	), nil
}
