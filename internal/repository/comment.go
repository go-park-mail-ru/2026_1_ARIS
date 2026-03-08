package repository

import (
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
)

type inmemoryCommentRepo struct {
	comments map[uuid.UUID]models.Comment
}

type CommentRepo interface {
	GetCommentCount(postID uuid.UUID) int
	Save(comment models.Comment) error
}

func NewCommentRepo() CommentRepo {
	repo := inmemoryCommentRepo{}
	repo.comments = make(map[uuid.UUID]models.Comment)
	return &repo
}

func (r *inmemoryCommentRepo) GetCommentCount(postID uuid.UUID) int {
	commentsCount := 0

	for _, c := range r.comments {
		if c.TargetPostID == postID {
			commentsCount++
		}
	}

	return commentsCount
}

func (r *inmemoryCommentRepo) Save(comment models.Comment) error {
	_, ok := r.comments[comment.ID]
	if !ok {
		r.comments[comment.ID] = comment
	}
	return nil
}
