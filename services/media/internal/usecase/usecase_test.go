package usecase

import (
	"context"
	"io"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/repository"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
)

type fakeMediaRepo struct {
	media *model.Media
	err   error
}

func (r fakeMediaRepo) Get(context.Context, int64) (*model.Media, error) {
	return r.media, r.err
}

func (r fakeMediaRepo) Save(context.Context, model.Media) (int64, error) {
	return 0, nil
}

func (r fakeMediaRepo) GetLink(context.Context, int64) (string, error) {
	return "", nil
}

func (r fakeMediaRepo) UpdateLink(context.Context, int64, string) error {
	return nil
}

type fakeS3Repo struct{}

func (fakeS3Repo) Save(context.Context, string, io.Reader, uuid.UUID, int64, string, minio.PutObjectOptions) (string, error) {
	return "", nil
}

func TestGetFileURLKeepsAbsoluteMediaLinks(t *testing.T) {
	svc := New(repository.NewStore(fakeMediaRepo{
		media: &model.Media{ID: 42, Link: "https://cdn.example/avatar.jpg"},
	}, fakeS3Repo{}, ""), nil)

	url, err := svc.GetFileURL(context.Background(), 42)

	require.NoError(t, err)
	require.Equal(t, "https://cdn.example/avatar.jpg", url)
}

func TestGetFileURLAddsEndpointForRelativeMediaLinks(t *testing.T) {
	t.Setenv("APP_ENDPOINT", "http://localhost:8080")
	svc := New(repository.NewStore(fakeMediaRepo{
		media: &model.Media{ID: 43, Link: "/media/avatar.jpg"},
	}, fakeS3Repo{}, ""), nil)

	url, err := svc.GetFileURL(context.Background(), 43)

	require.NoError(t, err)
	require.Equal(t, "http://localhost:8080/media/avatar.jpg", url)
}
