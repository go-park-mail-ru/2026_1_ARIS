package minioclient

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

type MinioClient struct {
	client *minio.Client
	// logger
}

func NewMinioClient(client *minio.Client) *MinioClient {
	return &MinioClient{
		client: client,
	}
}

func (client *MinioClient) Save(ctx context.Context, bucketName string, reader io.Reader, mediaUUID uuid.UUID, size int64, extension string, opts minio.PutObjectOptions) (string, error) {
	objectName := GenerageMedaName(mediaUUID, size, extension)

	uploadInto, err := client.client.PutObject(ctx, bucketName, objectName, reader, size, opts)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	return fmt.Sprintf("http://localhost:9000/%s/%s", uploadInto.Bucket, uploadInto.Key), nil
}
