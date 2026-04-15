package media

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestMediaServiceGetAvatarByID(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mrepo := repomocks.NewMockMediaRepo(ctrl)
	svc := NewMediaService(mrepo, repomocks.NewMockPostWithMediaRepo(ctrl), repomocks.NewMockS3Repo(ctrl))

	_, err := svc.GetAvatarByID(context.Background(), nil)
	require.ErrorContains(t, err, "nil")

	id := int64(1)
	mrepo.EXPECT().Get(gomock.Any(), int64(1)).Return(&models.Media{MimeType: "text/plain"}, nil)
	_, err = svc.GetAvatarByID(context.Background(), &id)
	require.ErrorContains(t, err, "image")

	mrepo.EXPECT().Get(gomock.Any(), int64(2)).Return(&models.Media{MimeType: "image/png"}, nil)
	got, err := svc.GetAvatarByID(context.Background(), func() *int64 { v := int64(2); return &v }())
	require.NoError(t, err)
	require.Equal(t, "image/png", got.MimeType)
}

func TestMediaServiceGetMediasByPostID(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mrepo := repomocks.NewMockMediaRepo(ctrl)
	pwm := repomocks.NewMockPostWithMediaRepo(ctrl)
	svc := NewMediaService(mrepo, pwm, repomocks.NewMockS3Repo(ctrl))

	pwm.EXPECT().GetMediaByPostID(gomock.Any(), int64(7)).Return([]int64{1, 2})
	mrepo.EXPECT().Get(gomock.Any(), int64(1)).Return(&models.Media{ID: 1, MimeType: "image/png"}, nil)
	mrepo.EXPECT().Get(gomock.Any(), int64(2)).Return(nil, errors.New("skip"))
	got := svc.GetMediasByPostID(context.Background(), 7)
	require.Len(t, got, 1)
	require.Equal(t, int64(1), got[0].ID)
}

func TestMediaServiceSave(t *testing.T) {
	t.Parallel()
	require.NoError(t, os.Setenv("MINIO_BUCKET_NAME", "test-bucket"))

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mrepo := repomocks.NewMockMediaRepo(ctrl)
	s3 := repomocks.NewMockS3Repo(ctrl)
	svc := NewMediaService(mrepo, repomocks.NewMockPostWithMediaRepo(ctrl), s3)

	// minimal PNG header for mimetype detection
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52}
	f, err := os.CreateTemp("", "media-test-*.bin")
	require.NoError(t, err)
	defer os.Remove(f.Name())
	_, err = f.Write(png)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	file, err := os.Open(f.Name())
	require.NoError(t, err)
	defer file.Close()

	s3.EXPECT().Save(gomock.Any(), "test-bucket", gomock.Any(), gomock.Any(), int64(len(png)), gomock.Any(), gomock.Any()).Return("https://link/obj", nil)
	mrepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(int64(42), nil)

	id, link, err := svc.Save(context.Background(), "pic", int64(len(png)), file, "avatar", 9)
	require.NoError(t, err)
	require.Equal(t, int64(42), id)
	require.Equal(t, "https://link/obj", link)
}

func TestMediaServiceSaveUnsupportedType(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	svc := NewMediaService(repomocks.NewMockMediaRepo(ctrl), repomocks.NewMockPostWithMediaRepo(ctrl), repomocks.NewMockS3Repo(ctrl))
	f, err := os.CreateTemp("", "plain-*.txt")
	require.NoError(t, err)
	defer os.Remove(f.Name())
	_, err = f.WriteString("plain text")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	file, err := os.Open(f.Name())
	require.NoError(t, err)
	defer file.Close()
	_, _, err = svc.Save(context.Background(), "f", 10, file, "post", 1)
	require.ErrorIs(t, err, xerrors.UnsupportedContentType)
}
