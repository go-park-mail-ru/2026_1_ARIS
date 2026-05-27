package usecase

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"testing"

	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/repository"
	repomock "github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
)

func newMediaService(t *testing.T) (*Service, *repomock.MockMediaRepo, *repomock.MockS3Repo, *usermock.MockUserServiceClient) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mediaRepo := repomock.NewMockMediaRepo(ctrl)
	s3Repo := repomock.NewMockS3Repo(ctrl)
	users := usermock.NewMockUserServiceClient(ctrl)
	return New(repository.NewStore(mediaRepo, s3Repo, "media"), users), mediaRepo, s3Repo, users
}

func multipartHeader(t *testing.T, name string, data []byte) *multipart.FileHeader {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", name)
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	reader := multipart.NewReader(&body, writer.Boundary())
	form, err := reader.ReadForm(int64(body.Len()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = form.RemoveAll() })
	return form.File["file"][0]
}

func TestGetFileURLKeepsAbsoluteMediaLinks(t *testing.T) {
	svc, mediaRepo, _, _ := newMediaService(t)
	mediaRepo.EXPECT().
		Get(gomock.Any(), int64(42)).
		Return(&model.Media{ID: 42, Link: "https://cdn.example/avatar.jpg"}, nil)

	url, err := svc.GetFileURL(context.Background(), 42)

	require.NoError(t, err)
	require.Equal(t, "https://cdn.example/avatar.jpg", url)
}

func TestGetFileURLAddsEndpointForRelativeMediaLinks(t *testing.T) {
	t.Setenv("APP_ENDPOINT", "http://localhost:8080")
	svc, mediaRepo, _, _ := newMediaService(t)
	mediaRepo.EXPECT().
		Get(gomock.Any(), int64(43)).
		Return(&model.Media{ID: 43, Link: "/media/avatar.jpg"}, nil)

	url, err := svc.GetFileURL(context.Background(), 43)

	require.NoError(t, err)
	require.Equal(t, "http://localhost:8080/media/avatar.jpg", url)
}

func TestGetFileValidationAndNotFound(t *testing.T) {
	t.Run("invalid id", func(t *testing.T) {
		svc, _, _, _ := newMediaService(t)

		media, err := svc.GetFile(context.Background(), 0)

		require.Nil(t, media)
		require.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("not found", func(t *testing.T) {
		svc, mediaRepo, _, _ := newMediaService(t)
		mediaRepo.EXPECT().
			Get(gomock.Any(), int64(99)).
			Return(nil, repository.ErrMediaNotFound)

		media, err := svc.GetFile(context.Background(), 99)

		require.Nil(t, media)
		require.ErrorIs(t, err, ErrMediaNotFound)
	})
}

func TestSaveFilesValidation(t *testing.T) {
	tests := []struct {
		name  string
		input SaveFilesInput
		err   error
	}{
		{name: "invalid user", input: SaveFilesInput{UserAccountID: 0}, err: ErrInvalidInput},
		{name: "no files", input: SaveFilesInput{UserAccountID: 7}, err: ErrFilesAreRequired},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _, _ := newMediaService(t)

			saved, fileErrs, err := svc.SaveFiles(context.Background(), tc.input)

			require.Nil(t, saved)
			require.Nil(t, fileErrs)
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestSaveFilesProfileNotFound(t *testing.T) {
	svc, _, _, users := newMediaService(t)
	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: 7}).
		Return(nil, errors.New("profile missing"))

	saved, fileErrs, err := svc.SaveFiles(context.Background(), SaveFilesInput{
		UserAccountID: 7,
		FileFor:       "support",
		FileHeaders:   []*multipart.FileHeader{multipartHeader(t, "image.png", []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0})},
	})

	require.Nil(t, saved)
	require.Nil(t, fileErrs)
	require.ErrorIs(t, err, ErrProfileNotFound)
}

func TestSaveFilesSuccess(t *testing.T) {
	svc, mediaRepo, s3Repo, users := newMediaService(t)
	header := multipartHeader(t, "avatar.png", []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0})

	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: 7}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 55}, nil)
	s3Repo.EXPECT().
		Save(gomock.Any(), "media", gomock.Any(), gomock.Any(), header.Size, ".png", gomock.AssignableToTypeOf(minio.PutObjectOptions{})).
		Return("/media/avatar.png", nil)
	mediaRepo.EXPECT().
		Save(gomock.Any(), gomock.AssignableToTypeOf(model.Media{})).
		DoAndReturn(func(_ context.Context, media model.Media) (int64, error) {
			require.Equal(t, int64(55), media.AuthorID)
			require.Equal(t, "avatar.png", media.Name)
			require.Equal(t, "image/png", media.MimeType)
			require.Equal(t, "/media/avatar.png", media.Link)
			return 101, nil
		})

	saved, fileErrs, err := svc.SaveFiles(context.Background(), SaveFilesInput{
		UserAccountID: 7,
		FileFor:       "support",
		FileHeaders:   []*multipart.FileHeader{header},
	})

	require.NoError(t, err)
	require.Empty(t, fileErrs)
	require.Equal(t, []SavedFile{{ID: 101, URL: "/media/avatar.png", Index: 0}}, saved)
}

func TestSaveFilesCollectsFileErrors(t *testing.T) {
	svc, _, _, users := newMediaService(t)
	header := multipartHeader(t, "notes.txt", []byte("plain text"))
	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: 7}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 55}, nil)

	saved, fileErrs, err := svc.SaveFiles(context.Background(), SaveFilesInput{
		UserAccountID: 7,
		FileFor:       "support",
		FileHeaders:   []*multipart.FileHeader{header},
	})

	require.NoError(t, err)
	require.Empty(t, saved)
	require.Equal(t, []FileError{{Index: 0, Error: ErrUnsupportedContentType.Error()}}, fileErrs)
}
