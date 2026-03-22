package media

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/media"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/post"
)

type mediaService struct {
	mediaRepo         media.MediaRepo
	postWithMediaRepo post.PostWithMediaRepo
}

type MediaService interface {
	GetAvatarByID(ctx context.Context, avatarID *int64) (*models.Media, error)
	GetMediasByPostID(ctx context.Context, postID int64) []models.Media
}

func NewMediaService(mediaRepo media.MediaRepo, postWithMediaRepo post.PostWithMediaRepo) MediaService {
	return &mediaService{
		mediaRepo:         mediaRepo,
		postWithMediaRepo: postWithMediaRepo,
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

	if media.MimeType != "image" {
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
