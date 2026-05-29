package http

//go:generate mockgen -source=service.go -destination=mocks/service_mock.go -package=mocks

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/analytics"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/usecase"
)

type PostService interface {
	RecordFeedEvents(context.Context, int64, []analytics.PostEvent)
	GetCommunityPosts(context.Context, int64, int64) ([]usecase.PostDetails, error)
	GetCommunityOfficialPosts(context.Context, int64, int64) ([]usecase.PostDetails, error)
	GetFeed(context.Context, int64, string, string, int) (usecase.FeedResult, error)
	GetPublicFeed(context.Context, string, int) (usecase.FeedResult, error)
	CreatePost(context.Context, int64, usecase.CreateInput) (*usecase.PostDetails, error)
	GetMyPosts(context.Context, int64) ([]usecase.PostDetails, error)
	GetProfilePosts(context.Context, int64, int64) ([]usecase.PostDetails, error)
	DeletePost(context.Context, int64, int64) error
	GetPostForViewer(context.Context, int64, int64) (*usecase.PostDetails, error)
	LikePost(context.Context, int64, int64) (*usecase.PostDetails, error)
	UnlikePost(context.Context, int64, int64) (*usecase.PostDetails, error)
	UpdatePost(context.Context, int64, int64, usecase.UpdateInput) (*usecase.PostDetails, error)
	LikeComment(context.Context, int64, int64, int64) (*usecase.Comment, error)
	UnlikeComment(context.Context, int64, int64, int64) (*usecase.Comment, error)
	GetPostComments(context.Context, int64, int64, int, int) ([]usecase.Comment, error)
	GetCommentReplies(context.Context, int64, int64, int64, int, int) ([]usecase.Comment, error)
	GetCommentRepliesBatch(context.Context, int64, int64, []int64, int, int) (map[int64][]usecase.Comment, error)
	CreateComment(context.Context, int64, int64, string, *int64) (*usecase.Comment, error)
	UpdateComment(context.Context, int64, int64, int64, string) (*usecase.Comment, error)
	DeleteComment(context.Context, int64, int64, int64) error
}
