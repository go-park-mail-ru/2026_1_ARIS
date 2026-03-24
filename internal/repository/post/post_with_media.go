package post

import (
	"context"
	"slices"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

type inmemoryPostWithMediaRepo struct {
	mu             sync.RWMutex
	postWithMedias []models.PostWithMedia
}

type PostWithMediaRepo interface {
	GetMediaByPostID(ctx context.Context, postID int64) []int64
	Save(ctx context.Context, postWithMedia models.PostWithMedia) error
}

func NewPostWithMediaRepo() PostWithMediaRepo {
	return &inmemoryPostWithMediaRepo{}
}

// убрать отсюда, переложить в сервис
func (r *inmemoryPostWithMediaRepo) GetMediaByPostID(ctx context.Context, postID int64) []int64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	var mediaIDs []int64

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

func (r *inmemoryPostWithMediaRepo) Save(ctx context.Context, postWithMedia models.PostWithMedia) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.postWithMedias = append(r.postWithMedias, postWithMedia)
	return nil
}
