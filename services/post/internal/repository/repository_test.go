package repository

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/model"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type feedFriendsQueryMatcher struct{}

func (feedFriendsQueryMatcher) Matches(value any) bool {
	query, ok := value.(string)
	if !ok {
		return false
	}
	normalized := strings.ToLower(query)
	return strings.Contains(normalized, "author_id = any") && !strings.Contains(normalized, "community")
}

func (feedFriendsQueryMatcher) String() string {
	return "friends feed SQL without community posts"
}

func TestGetFeedPageWithAuthorsDoesNotIncludeCommunityPosts(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	db := repomocks.NewMockDB(ctrl)
	dbErr := errors.New("stop after query assertion")
	authorIDs := []int64{10, 11}

	db.EXPECT().
		Query(gomock.Any(), feedFriendsQueryMatcher{}, false, authorIDs, 21).
		Return(nil, dbErr)

	store := NewStore(db)
	_, err := store.Posts.GetFeedPage(context.Background(), authorIDs, nil, nil, 21, false)
	require.ErrorIs(t, err, dbErr)
}

func TestPostRepositoriesReturnDBErrors(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	db := repomocks.NewMockDB(ctrl)
	row := repomocks.NewMockRow(ctrl)
	dbErr := errors.New("db down")

	row.EXPECT().Scan(gomock.Any()).Return(dbErr).AnyTimes()
	db.EXPECT().Begin(gomock.Any()).Return(nil, dbErr).AnyTimes()
	db.EXPECT().QueryRow(gomock.Any(), gomock.Any(), gomock.Any()).Return(row).AnyTimes()
	db.EXPECT().Exec(gomock.Any(), gomock.Any(), gomock.Any()).Return(pgconn.CommandTag{}, dbErr).AnyTimes()
	db.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbErr).AnyTimes()

	ctx := context.Background()
	store := NewStore(db)
	post := model.Post{ID: 1, AuthorID: 10, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	comment := model.Comment{ID: 1, PostID: 1, AuthorID: 10, CreatedAt: time.Now(), UpdatedAt: time.Now()}

	_, err := store.Memberships.GetMemberCommunityIDs(ctx, 10)
	require.Error(t, err)

	_, err = store.Posts.Save(ctx, post)
	require.Error(t, err)
	require.Error(t, store.Posts.Delete(ctx, 1))
	require.Error(t, store.Posts.Update(ctx, post))
	_, err = store.Posts.Get(ctx, 1)
	require.Error(t, err)
	_, err = store.Posts.GetAll(ctx)
	require.Error(t, err)
	_, err = store.Posts.GetByAuthorID(ctx, 10)
	require.Error(t, err)
	_, err = store.Posts.GetByCommunityID(ctx, 5)
	require.Error(t, err)
	_, err = store.Posts.GetByIDs(ctx, []int64{1, 2})
	require.Error(t, err)
	_, err = store.Posts.GetFeedPage(ctx, []int64{10}, nil, nil, 10, false)
	require.Error(t, err)
	posts, err := store.Posts.GetByIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, posts)

	require.Empty(t, store.PostMedia.GetMediaByPostID(ctx, 1))
	_, err = store.PostMedia.GetDetailedMediaByPostID(ctx, 1)
	require.Error(t, err)
	_, err = store.PostMedia.GetDetailedMediaByPostIDs(ctx, []int64{1, 2})
	require.Error(t, err)
	_, err = store.PostMedia.GetMediaAuthorID(ctx, 1)
	require.Error(t, err)
	require.Error(t, store.PostMedia.Save(ctx, model.PostWithMedia{PostID: 1, MediaID: 2}))
	require.Error(t, store.PostMedia.DeleteByPostID(ctx, 1))
	details, err := store.PostMedia.GetDetailedMediaByPostIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, details)

	require.Zero(t, store.Comments.GetCommentCount(ctx, 1))
	_, err = store.Comments.GetTopLevelByPostID(ctx, 1, 10, 0)
	require.Error(t, err)
	_, err = store.Comments.GetReplies(ctx, 1, 2, 10, 0)
	require.Error(t, err)
	_, err = store.Comments.GetRepliesByParentIDs(ctx, 1, []int64{2, 3}, 10, 0)
	require.Error(t, err)
	_, err = store.Comments.Get(ctx, 1)
	require.Error(t, err)
	_, err = store.Comments.Save(ctx, comment)
	require.Error(t, err)
	require.Error(t, store.Comments.Update(ctx, comment))
	require.Error(t, store.Comments.Delete(ctx, 1))
	_, err = store.Comments.GetCommentCountsBatch(ctx, []int64{1, 2})
	require.NoError(t, err)
	replies, err := store.Comments.GetRepliesByParentIDs(ctx, 1, nil, 10, 0)
	require.NoError(t, err)
	require.Empty(t, replies)
	counts, err := store.Comments.GetCommentCountsBatch(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, counts)

	_, err = store.Likes.Save(ctx, model.Like{PostID: &post.ID, AuthorID: 10})
	require.Error(t, err)
	require.Zero(t, store.Likes.GetLikeCountOnPost(ctx, 1))
	_, err = store.Likes.GetPostLikeByAuthor(ctx, 1, 10)
	require.Error(t, err)
	require.Error(t, store.Likes.SetActive(ctx, 1, true))
	require.False(t, store.Likes.HasActivePostLike(ctx, 1, 10))
	_, err = store.Likes.GetCommentLikeByAuthor(ctx, 1, 10)
	require.Error(t, err)
	_, err = store.Likes.GetCommentLikeCountBatch(ctx, []int64{1})
	require.NoError(t, err)
	_, err = store.Likes.GetCommentViewerLikesBatch(ctx, []int64{1}, 10)
	require.NoError(t, err)
	_, err = store.Likes.GetPostLikeCountsBatch(ctx, []int64{1})
	require.NoError(t, err)
	_, err = store.Likes.GetViewerPostLikesBatch(ctx, []int64{1}, 10)
	require.NoError(t, err)

	require.Zero(t, store.Reposts.GetRepostCount(ctx, 1))
	_, err = store.Reposts.GetRepostCountsBatch(ctx, []int64{1})
	require.NoError(t, err)
}
