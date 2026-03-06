package repository

import (
	"slices"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
)

type inmemoryPostWithMediaRepo struct {
	postWithMedias []models.PostWithMedia
}

type PostWithMediaRepo interface {
	GetMediaByPost(postID uuid.UUID) []uuid.UUID
	Save(post models.Post, media models.Media, order int) error
}

func NewPostWithMediaRepo() *inmemoryPostWithMediaRepo {
	return &inmemoryPostWithMediaRepo{}
}

func (r *inmemoryPostWithMediaRepo) GetMediaByPost(postID uuid.UUID) []uuid.UUID {
	var mediaIDs []uuid.UUID

	slices.SortFunc(r.postWithMedias, func(i, j models.PostWithMedia) int {
		if i.Order < j.Order {
			return -1
		} else if i.Order > j.Order {
			return 1
		}
		return 0
	})

	for _, p := range r.postWithMedias {
		if p.PostID == postID {
			mediaIDs = append(mediaIDs, p.MediaID)
		}
	}

	return mediaIDs
}

func (r *inmemoryPostWithMediaRepo) Save(post models.Post, media models.Media, order int) error {
	postWithMedia := models.NewPostWithMedia(post, media, order)
	r.postWithMedias = append(r.postWithMedias, postWithMedia)
	return nil
}
