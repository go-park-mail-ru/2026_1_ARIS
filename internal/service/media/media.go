package media

//go:generate mockgen -destination=../mocks/media_service_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/service/media MediaService

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/media"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/post"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

type mediaService struct {
	mediaRepo         media.MediaRepo
	postWithMediaRepo post.PostWithMediaRepo
	minioClient       media.S3Repo
}

type MediaService interface {
	GetAvatarByID(ctx context.Context, avatarID *int64) (*models.Media, error)
	GetMediasByPostID(ctx context.Context, postID int64) []models.Media
	Save(ctx context.Context, name string, size int64, fileReader multipart.File, contentType string, authorID int64) (int64, string, error)
}

func NewMediaService(mediaRepo media.MediaRepo, postWithMediaRepo post.PostWithMediaRepo, minioClient media.S3Repo) MediaService {
	return &mediaService{
		mediaRepo:         mediaRepo,
		postWithMediaRepo: postWithMediaRepo,
		minioClient:       minioClient,
	}
}

func (s *mediaService) GetAvatarByID(ctx context.Context, avatarID *int64) (*models.Media, error) {
	if avatarID == nil {
		return nil, errors.New("avatar is nil")
	}

	media, err := s.mediaRepo.Get(ctx, *avatarID)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(media.MimeType, "image") {
		return nil, errors.New("Avatar has not \"image\" MIME-type")
	}

	return media, nil
}

func (s *mediaService) GetMediasByPostID(ctx context.Context, postID int64) []models.Media {
	var medias []models.Media

	mediaIDs := s.postWithMediaRepo.GetMediaByPostID(ctx, postID)

	for _, mediaID := range mediaIDs {
		media, err := s.mediaRepo.Get(ctx, mediaID)
		if err == nil {
			medias = append(medias, *media)
		}
	}

	return medias
}

func (s *mediaService) Save(ctx context.Context, name string, size int64, fileReader multipart.File, contentType string, authorID int64) (int64, string, error) {
	mediaUUID := uuid.New()

	mType, extension, err := GetMimeType(fileReader)

	if !isAllowed(mType, contentType) {
		return 0, "", xerrors.UnsupportedContentType
	}

	fileReader.Seek(0, io.SeekStart)

	opts := minio.PutObjectOptions{
		ContentType: mType,
	}

	mediaLink, err := s.minioClient.Save(ctx, os.Getenv("MINIO_BUCKET_NAME"), fileReader, mediaUUID, size, extension, opts)
	if err != nil {
		return 0, "", err
	}

	createdMedia := models.NewMedia(name, extension, mediaUUID, nil, mType, mediaLink, authorID)

	mediaID, err := s.mediaRepo.Save(ctx, *createdMedia)
	if err != nil {
		return 0, "", err
	}

	return mediaID, mediaLink, nil
}

func GetMimeType(file io.Reader) (string, string, error) {
	mType, err := mimetype.DetectReader(file)
	if err == nil {
		return mType.String(), mType.Extension(), nil
	}

	return mType.String(), "", err
}

var allowedByContext = map[string][]string{
	"post": {"image/"},
	//"post_attachment": {},
	"support": {"image/"},
	"avatar":  {"image/"},
	//"message": {},
}

func isAllowed(mimeType string, allowed string) bool {
	for _, prefix := range allowedByContext[allowed] {
		if strings.HasPrefix(mimeType, prefix) {
			return true
		}
	}
	return false
}
