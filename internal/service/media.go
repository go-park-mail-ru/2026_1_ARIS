package service

import (
	"errors"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"github.com/google/uuid"
)

type mediaService struct {
	mediaRepo         repository.MediaRepo
	postWithMediaRepo repository.PostWithMediaRepo
}

type MediaService interface {
	GetAvatarByID(avatarID uuid.UUID) (models.Media, error)
	GetMediaByPost(postID uuid.UUID) []models.Media
}

func NewMediaService(mediaRepo repository.MediaRepo, postWithMediaRepo repository.PostWithMediaRepo) MediaService {
	return &mediaService{
		mediaRepo:         mediaRepo,
		postWithMediaRepo: postWithMediaRepo,
	}
}

func (s *mediaService) GetAvatarByID(avatarID uuid.UUID) (models.Media, error) {
	media, err := s.mediaRepo.GetMediaByID(avatarID)
	if err != nil {
		return models.Media{}, err
	}

	if media.MimeType != "image" {
		return models.Media{}, errors.New("Avatar has not \"image\" MIME-type")
	}

	return media, nil
}

func (s *mediaService) GetMediaByPost(postID uuid.UUID) []models.Media {
	var medias []models.Media

	mediaIDs := s.postWithMediaRepo.GetMediaByPost(postID)

	for _, mediaID := range mediaIDs {
		media, err := s.mediaRepo.GetMediaByID(mediaID)
		if err == nil {
			medias = append(medias, media)
		}
	}

	return medias
}
