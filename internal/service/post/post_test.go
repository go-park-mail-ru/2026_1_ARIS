package post

import (
	"context"
	"errors"
	"testing"
	"time"

	hdto "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/cursor"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPostServiceCRUDAndCounts(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	postRepo := repomocks.NewMockPostRepo(ctrl)
	pwm := repomocks.NewMockPostWithMediaRepo(ctrl)
	prof := repomocks.NewMockProfileRepo(ctrl)
	com := repomocks.NewMockCommentRepo(ctrl)
	rep := repomocks.NewMockRepostRepo(ctrl)
	like := repomocks.NewMockLikeRepo(ctrl)
	svc := NewPostService(postRepo, pwm, prof, com, rep, like)

	p := &models.Post{ID: 1}
	postRepo.EXPECT().Get(gomock.Any(), int64(1)).Return(p, nil)
	got, err := svc.Get(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, p, got)

	postRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(int64(5), nil)
	id, err := svc.Save(context.Background(), models.Post{})
	require.NoError(t, err)
	require.Equal(t, int64(5), id)

	postRepo.EXPECT().Delete(gomock.Any(), int64(3)).Return(nil)
	require.NoError(t, svc.Delete(context.Background(), 3))

	postRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	require.NoError(t, svc.Update(context.Background(), models.Post{ID: 1}))

	postRepo.EXPECT().GetByAuthorID(gomock.Any(), int64(9)).Return([]models.Post{{ID: 1}}, nil)
	list, err := svc.GetByAuthorID(context.Background(), 9)
	require.NoError(t, err)
	require.Len(t, list, 1)

	like.EXPECT().GetLikeCountOnPost(gomock.Any(), int64(1)).Return(3)
	require.Equal(t, 3, svc.GetLikeCount(context.Background(), 1))
	com.EXPECT().GetCommentCount(gomock.Any(), int64(1)).Return(4)
	require.Equal(t, 4, svc.GetCommentCount(context.Background(), 1))
	rep.EXPECT().GetRepostCount(gomock.Any(), int64(1)).Return(2)
	require.Equal(t, 2, svc.GetRepostCount(context.Background(), 1))
}

func TestPostServiceGetPostAuthor(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	postRepo := repomocks.NewMockPostRepo(ctrl)
	prof := repomocks.NewMockProfileRepo(ctrl)
	svc := NewPostService(postRepo, repomocks.NewMockPostWithMediaRepo(ctrl), prof, repomocks.NewMockCommentRepo(ctrl), repomocks.NewMockRepostRepo(ctrl), repomocks.NewMockLikeRepo(ctrl))

	postRepo.EXPECT().Get(gomock.Any(), int64(1)).Return(nil, xerrors.PostNotFound)
	_, err := svc.GetPostAuthor(context.Background(), 1)
	require.Error(t, err)

	postRepo.EXPECT().Get(gomock.Any(), int64(2)).Return(&models.Post{AuthorID: 10}, nil)
	prof.EXPECT().Get(gomock.Any(), int64(10)).Return(nil, errors.New("nf"))
	_, err = svc.GetPostAuthor(context.Background(), 2)
	require.Error(t, err)

	want := &models.Profile{ID: 10}
	postRepo.EXPECT().Get(gomock.Any(), int64(3)).Return(&models.Post{AuthorID: 10}, nil)
	prof.EXPECT().Get(gomock.Any(), int64(10)).Return(want, nil)
	got, err := svc.GetPostAuthor(context.Background(), 3)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestPostServiceAttachReplaceMedia(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	postRepo := repomocks.NewMockPostRepo(ctrl)
	pwm := repomocks.NewMockPostWithMediaRepo(ctrl)
	svc := NewPostService(postRepo, pwm, repomocks.NewMockProfileRepo(ctrl), repomocks.NewMockCommentRepo(ctrl), repomocks.NewMockRepostRepo(ctrl), repomocks.NewMockLikeRepo(ctrl))

	pwm.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
	pwm.EXPECT().Save(gomock.Any(), gomock.Any()).Return(errors.New("dup"))
	res := svc.AttachMedia(context.Background(), 1, []hdto.MediaRequestData{{MediaID: 1}, {MediaID: 2}})
	require.Len(t, res.Errs, 1)

	pwm.EXPECT().DeleteByPostID(gomock.Any(), int64(1)).Return(errors.New("del"))
	res2 := svc.ReplaceMedia(context.Background(), 1, []hdto.MediaRequestData{{MediaID: 1}})
	require.Len(t, res2.Errs, 1)

	pwm.EXPECT().DeleteByPostID(gomock.Any(), int64(2)).Return(nil)
	got := svc.ReplaceMedia(context.Background(), 2, nil)
	require.Empty(t, got.Errs)

	pwm.EXPECT().DeleteByPostID(gomock.Any(), int64(3)).Return(nil)
	pwm.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
	got3 := svc.ReplaceMedia(context.Background(), 3, []hdto.MediaRequestData{{MediaID: 5}})
	require.Empty(t, got3.Errs)
}

func TestPostServiceFeeds(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	postRepo := repomocks.NewMockPostRepo(ctrl)
	svc := NewPostService(postRepo, repomocks.NewMockPostWithMediaRepo(ctrl), repomocks.NewMockProfileRepo(ctrl), repomocks.NewMockCommentRepo(ctrl), repomocks.NewMockRepostRepo(ctrl), repomocks.NewMockLikeRepo(ctrl)).(*postService)

	_, err := svc.GetFeed(context.Background(), "not-valid-base64!!!", 10)
	require.Error(t, err)

	postRepo.EXPECT().GetAll(gomock.Any()).Return(nil, errors.New("db"))
	_, err = svc.GetFeed(context.Background(), "", 10)
	require.Error(t, err)

	now := time.Now()
	posts := []models.Post{
		{ID: 1, Uid: uuid.New(), IsPublicDemo: false, CreatedAt: now.Add(-3 * time.Hour)},
		{ID: 2, Uid: uuid.New(), IsPublicDemo: false, CreatedAt: now.Add(-2 * time.Hour)},
		{ID: 3, Uid: uuid.New(), IsPublicDemo: false, CreatedAt: now.Add(-1 * time.Hour)},
	}
	postRepo.EXPECT().GetAll(gomock.Any()).Return(posts, nil)
	res, err := svc.GetFeed(context.Background(), "", 2)
	require.NoError(t, err)
	require.True(t, res.HasMore)
	require.Len(t, res.Posts, 2)

	demoPosts := []models.Post{
		{ID: 1, Uid: uuid.New(), IsPublicDemo: true, CreatedAt: now},
	}
	postRepo.EXPECT().GetAll(gomock.Any()).Return(demoPosts, nil)
	pub, err := svc.GetPublicFeed(context.Background(), "", 5)
	require.NoError(t, err)
	require.False(t, pub.HasMore)

	pop, err := svc.GetPopularPosts(context.Background())
	require.NoError(t, err)
	require.Nil(t, pop)
	pubPop, err := svc.GetPublicPopularPosts(context.Background())
	require.NoError(t, err)
	require.Nil(t, pubPop)
}

func TestPostServiceGetCursoredPostsBranches(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	postRepo := repomocks.NewMockPostRepo(ctrl)
	svc := NewPostService(postRepo, repomocks.NewMockPostWithMediaRepo(ctrl), repomocks.NewMockProfileRepo(ctrl), repomocks.NewMockCommentRepo(ctrl), repomocks.NewMockRepostRepo(ctrl), repomocks.NewMockLikeRepo(ctrl)).(*postService)

	now := time.Now()
	posts := []models.Post{
		{ID: 1, Uid: uuid.New(), IsPublicDemo: false, CreatedAt: now.Add(-3 * time.Hour)},
		{ID: 2, Uid: uuid.New(), IsPublicDemo: true, CreatedAt: now.Add(-2 * time.Hour)},
		{ID: 3, Uid: uuid.New(), IsPublicDemo: false, CreatedAt: now.Add(-1 * time.Hour)},
	}

	postRepo.EXPECT().GetAll(gomock.Any()).Return(posts, nil)
	got, err := svc.getCursoredPosts(context.Background(), FeedParams{Limit: 10})
	require.NoError(t, err)
	require.Len(t, got, 2)

	cursorAfterFirst := FeedParams{
		Limit: 1,
		Cursor: &cursor.Cursor{
			CreatedAt: now.Add(-3 * time.Hour),
			ID:        uuid.New(),
		},
	}
	postRepo.EXPECT().GetAll(gomock.Any()).Return(posts, nil)
	got, err = svc.getCursoredPosts(context.Background(), cursorAfterFirst)
	require.NoError(t, err)
	require.NotEmpty(t, got)

	postRepo.EXPECT().GetAll(gomock.Any()).Return(posts, nil)
	_, err = svc.getCursoredPosts(context.Background(), FeedParams{
		Limit: 1,
		Cursor: &cursor.Cursor{
			CreatedAt: now.Add(10 * time.Hour),
			ID:        uuid.New(),
		},
	})
	require.Error(t, err)
}

func TestPostServiceGetCursoredPublicPostsBranches(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	postRepo := repomocks.NewMockPostRepo(ctrl)
	svc := NewPostService(postRepo, repomocks.NewMockPostWithMediaRepo(ctrl), repomocks.NewMockProfileRepo(ctrl), repomocks.NewMockCommentRepo(ctrl), repomocks.NewMockRepostRepo(ctrl), repomocks.NewMockLikeRepo(ctrl)).(*postService)

	now := time.Now()
	publicPosts := []models.Post{
		{ID: 1, Uid: uuid.New(), IsPublicDemo: true, CreatedAt: now.Add(-3 * time.Hour)},
		{ID: 2, Uid: uuid.New(), IsPublicDemo: false, CreatedAt: now.Add(-2 * time.Hour)},
		{ID: 3, Uid: uuid.New(), IsPublicDemo: true, CreatedAt: now.Add(-1 * time.Hour)},
	}

	postRepo.EXPECT().GetAll(gomock.Any()).Return(publicPosts, nil)
	got, err := svc.getCursoredPublicPosts(context.Background(), FeedParams{Limit: 10})
	require.NoError(t, err)
	require.Len(t, got, 2)

	postRepo.EXPECT().GetAll(gomock.Any()).Return(publicPosts, nil)
	got, err = svc.getCursoredPublicPosts(context.Background(), FeedParams{
		Limit: 1,
		Cursor: &cursor.Cursor{
			CreatedAt: now.Add(-3 * time.Hour),
			ID:        uuid.New(),
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, got)

	postRepo.EXPECT().GetAll(gomock.Any()).Return(publicPosts, nil)
	_, err = svc.getCursoredPublicPosts(context.Background(), FeedParams{
		Limit: 1,
		Cursor: &cursor.Cursor{
			CreatedAt: now.Add(10 * time.Hour),
			ID:        uuid.New(),
		},
	})
	require.Error(t, err)
}
