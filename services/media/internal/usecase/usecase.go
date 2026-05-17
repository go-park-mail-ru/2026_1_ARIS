package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/repository"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

var (
	ErrFilesAreRequired       = errors.New("files are required")
	ErrInvalidInput           = errors.New("invalid input")
	ErrUnsupportedContentType = errors.New("unsupported content type")
	ErrMediaNotFound          = repository.ErrMediaNotFound
	ErrProfileNotFound        = errors.New("profile not found")
)

const ErrCantOpenFileStr = "failed open file"

type Service struct {
	userClient userpb.UserServiceClient
	store      repository.Store
}

func New(store repository.Store, userClient userpb.UserServiceClient) *Service {
	return &Service{store: store, userClient: userClient}
}

func (s *Service) SaveFiles(ctx context.Context, input SaveFilesInput) ([]SavedFile, []FileError, error) {
	if input.UserAccountID <= 0 {
		return nil, nil, ErrInvalidInput
	}
	if len(input.FileHeaders) == 0 {
		return nil, nil, ErrFilesAreRequired
	}

	res, err := s.userClient.GetProfileByUserAccount(ctx, &userpb.GetProfileByUserAccountRequest{UserAccountId: input.UserAccountID})
	if err != nil || res == nil {
		return nil, nil, ErrProfileNotFound
	}

	result := make([]SavedFile, 0, len(input.FileHeaders))
	fileErrors := make([]FileError, 0)

	for i, fileHeader := range input.FileHeaders {
		file, err := fileHeader.Open()
		if err != nil {
			fileErrors = append(fileErrors, FileError{Index: i, Error: ErrCantOpenFileStr})
			continue
		}

		saved, err := s.saveFile(ctx, SaveFileInput{
			Name:      fileHeader.Filename,
			Size:      fileHeader.Size,
			File:      file,
			FileFor:   input.FileFor,
			ProfileID: res.GetProfileId(),
		})
		_ = file.Close()

		if err != nil {
			fileErrors = append(fileErrors, FileError{Index: i, Error: err.Error()})
			continue
		}

		saved.Index = i
		result = append(result, *saved)
	}

	return result, fileErrors, nil
}

func (s *Service) saveFile(ctx context.Context, in SaveFileInput) (*SavedFile, error) {
	if in.Size <= 0 || in.File == nil || strings.TrimSpace(in.FileFor) == "" || in.ProfileID <= 0 {
		return nil, ErrInvalidInput
	}

	mediaUUID := uuid.New()
	if strings.TrimSpace(in.Name) == "" {
		in.Name = fmt.Sprintf("%s-%d", mediaUUID.String(), in.Size)
	}

	mimeType, extension, err := detectMimeType(in.File)
	if err != nil {
		return nil, err
	}
	if !isAllowed(mimeType, in.FileFor) {
		return nil, ErrUnsupportedContentType
	}
	if _, err := in.File.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	bucketName := s.store.BucketName
	if bucketName == "" {
		bucketName = os.Getenv("MINIO_BUCKET_NAME")
	}

	link, err := s.store.S3.Save(ctx, bucketName, in.File, mediaUUID, in.Size, extension, minio.PutObjectOptions{ContentType: mimeType})
	if err != nil {
		return nil, err
	}

	createdMedia := model.NewMedia(in.Name, extension, mediaUUID, nil, mimeType, link, in.ProfileID)
	createdMedia.Size = int(in.Size)

	mediaID, err := s.store.Media.Save(ctx, *createdMedia)
	if err != nil {
		return nil, err
	}
	return &SavedFile{ID: mediaID, URL: link}, nil
}

func (s *Service) GetFileURL(ctx context.Context, mediaID int64) (string, error) {
	if mediaID <= 0 {
		return "", ErrInvalidInput
	}
	media, err := s.GetFile(ctx, mediaID)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(media.Link, "http://") || strings.HasPrefix(media.Link, "https://") {
		return media.Link, nil
	}

	appEndpoint := os.Getenv("APP_ENDPOINT")
	if appEndpoint == "" {
		appEndpoint = "http://localhost:8080"
	}
	return fmt.Sprintf("%s%s", appEndpoint, media.Link), nil
}

func (s *Service) GetFile(ctx context.Context, mediaID int64) (*model.Media, error) {
	if mediaID <= 0 {
		return nil, ErrInvalidInput
	}

	media, err := s.store.Media.Get(ctx, mediaID)
	if err != nil {
		if errors.Is(err, repository.ErrMediaNotFound) {
			return nil, ErrMediaNotFound
		}
		return nil, err
	}
	return media, nil
}

func detectMimeType(file io.Reader) (string, string, error) {
	mimeType, err := mimetype.DetectReader(file)
	if err != nil {
		return mimeType.String(), "", err
	}
	return mimeType.String(), mimeType.Extension(), nil
}

var allowedByContext = map[string][]string{
	"avatar":  {"image/"},
	"post":    {"image/", "video/", "audio/", "application/", "text/"},
	"chat":    {"image/", "video/", "audio/", "application/", "text/"},
	"message": {"image/", "video/", "audio/", "application/", "text/"},
	"sticker": {"image/"},
	"support": {"image/"},
}

func isAllowed(mimeType string, fileFor string) bool {
	for _, prefix := range allowedByContext[fileFor] {
		if strings.HasPrefix(mimeType, prefix) {
			return true
		}
	}
	return false
}
