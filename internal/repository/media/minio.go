package media

import (
	"context"
	"fmt"
	"io"
	"os"

	xminio "github.com/go-park-mail-ru/2026_1_ARIS/pkg/minio"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
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
	objectName := xminio.GenerateMediaName(mediaUUID, size, extension)

	uploadInto, err := client.client.PutObject(ctx, bucketName, objectName, reader, size, opts)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	return fmt.Sprintf("http://%s:%s/%s/%s", os.Getenv("MINIO_PUBLIC_HOST"), os.Getenv("MINIO_PORT"), uploadInto.Bucket, uploadInto.Key), nil
}
