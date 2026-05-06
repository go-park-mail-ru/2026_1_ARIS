package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/media/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	mediamock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/media/mock"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
)

type nopFile struct {
	*bytes.Reader
}

func (f nopFile) Close() error { return nil }

func TestSaveFileStoresAllowedImage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mediaRepo := mediamock.NewMockMediaRepo(ctrl)
	s3Repo := mediamock.NewMockS3Repo(ctrl)
	userClient := usermock.NewMockUserServiceClient(ctrl)
	svc := New(repository.NewStore(mediaRepo, s3Repo, "bucket"), userClient)
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0}

	s3Repo.EXPECT().
		Save(ctx, "bucket", gomock.Any(), gomock.Any(), int64(len(png)), ".png", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ io.Reader, id uuid.UUID, _ int64, _ string, opts minio.PutObjectOptions) (string, error) {
			require.NotEqual(t, uuid.Nil, id)
			require.Equal(t, "image/png", opts.ContentType)
			return "/media/image.png", nil
		})
	mediaRepo.EXPECT().Save(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, media models.Media) (int64, error) {
		require.Equal(t, "avatar.png", media.Name)
		require.Equal(t, "image/png", media.MimeType)
		require.Equal(t, "/media/image.png", media.Link)
		require.Equal(t, int64(77), media.AuthorID)
		return int64(100), nil
	})

	saved, err := svc.saveFile(ctx, SaveFileInput{
		Name:      "avatar.png",
		Size:      int64(len(png)),
		File:      nopFile{bytes.NewReader(png)},
		FileFor:   "avatar",
		ProfileID: 77,
	})

	require.NoError(t, err)
	require.Equal(t, int64(100), saved.ID)
	require.Equal(t, "/media/image.png", saved.URL)
}

func TestSaveFilesValidatesUserAndInputs(t *testing.T) {
	t.Run("invalid account", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		svc := New(repository.Store{}, usermock.NewMockUserServiceClient(ctrl))

		files, fileErrors, err := svc.SaveFiles(context.Background(), SaveFilesInput{UserAccountID: 0})

		require.Nil(t, files)
		require.Nil(t, fileErrors)
		require.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("no files", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		svc := New(repository.Store{}, usermock.NewMockUserServiceClient(ctrl))

		files, fileErrors, err := svc.SaveFiles(context.Background(), SaveFilesInput{UserAccountID: 10})

		require.Nil(t, files)
		require.Nil(t, fileErrors)
		require.ErrorIs(t, err, ErrFilesAreRequired)
	})

	t.Run("profile lookup failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		ctx := context.Background()
		userClient := usermock.NewMockUserServiceClient(ctrl)
		svc := New(repository.Store{}, userClient)
		userClient.EXPECT().GetProfileByUserAccount(ctx, &userpb.GetProfileByUserAccountRequest{UserAccountId: 10}).Return(nil, errors.New("boom"))

		files, fileErrors, err := svc.SaveFiles(ctx, SaveFilesInput{UserAccountID: 10, FileHeaders: make([]*multipart.FileHeader, 1)})

		require.Nil(t, files)
		require.Nil(t, fileErrors)
		require.ErrorIs(t, err, ErrProfileNotFound)
	})
}

func TestGetFileAndURL(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid id", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		svc := New(repository.Store{}, usermock.NewMockUserServiceClient(ctrl))

		media, err := svc.GetFile(ctx, 0)

		require.Nil(t, media)
		require.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("not found is normalized", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mediaRepo := mediamock.NewMockMediaRepo(ctrl)
		svc := New(repository.NewStore(mediaRepo, nil, ""), usermock.NewMockUserServiceClient(ctrl))
		mediaRepo.EXPECT().Get(ctx, int64(10)).Return(nil, xerrors.MediaNotFound)

		media, err := svc.GetFile(ctx, 10)

		require.Nil(t, media)
		require.ErrorIs(t, err, ErrMediaNotFound)
	})

	t.Run("url uses app endpoint fallback", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mediaRepo := mediamock.NewMockMediaRepo(ctrl)
		svc := New(repository.NewStore(mediaRepo, nil, ""), usermock.NewMockUserServiceClient(ctrl))
		mediaRepo.EXPECT().Get(ctx, int64(10)).Return(&models.Media{ID: 10, Link: "/m/10"}, nil)

		url, err := svc.GetFileURL(ctx, 10)

		require.NoError(t, err)
		require.Equal(t, "http://localhost:8080/m/10", url)
	})
}

func TestMimePolicy(t *testing.T) {
	require.True(t, isAllowed("image/png", "avatar"))
	require.True(t, isAllowed("image/jpeg", "post"))
	require.False(t, isAllowed("application/pdf", "avatar"))
	require.False(t, isAllowed("image/png", "unknown"))
}
