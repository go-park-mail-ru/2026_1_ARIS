package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/post/repository"
	commentmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/comment/mock"
	communitymock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/community/mock"
	likemock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/like/mock"
	postmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/post/mock"
	repostmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/repost/mock"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	mediamock "github.com/go-park-mail-ru/2026_1_ARIS/proto/media/mock"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type postMocks struct {
	posts         *postmock.MockPostRepo
	postWithMedia *postmock.MockPostWithMediaRepo
	comments      *commentmock.MockCommentRepo
	likes         *likemock.MockLikeRepo
	reposts       *repostmock.MockRepostRepo
	communities   *communitymock.MockCommunityRepo
	userClient    *usermock.MockUserServiceClient
	mediaClient   *mediamock.MockMediaServiceClient
	service       *Service
}

func newPostMocks(t *testing.T) (*gomock.Controller, postMocks) {
	t.Helper()

	ctrl := gomock.NewController(t)
	m := postMocks{
		posts:         postmock.NewMockPostRepo(ctrl),
		postWithMedia: postmock.NewMockPostWithMediaRepo(ctrl),
		comments:      commentmock.NewMockCommentRepo(ctrl),
		likes:         likemock.NewMockLikeRepo(ctrl),
		reposts:       repostmock.NewMockRepostRepo(ctrl),
		communities:   communitymock.NewMockCommunityRepo(ctrl),
		userClient:    usermock.NewMockUserServiceClient(ctrl),
		mediaClient:   mediamock.NewMockMediaServiceClient(ctrl),
	}
	m.service = New(repository.NewStore(m.posts, m.postWithMedia, m.comments, m.likes, m.reposts, m.communities), m.userClient, m.mediaClient)
	return ctrl, m
}

func expectPostProfile(m postMocks, ctx context.Context, accountID, profileID int64) {
	m.userClient.EXPECT().
		GetProfileByUserAccount(ctx, &userpb.GetProfileByUserAccountRequest{UserAccountId: accountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)
}

func expectPostAuthor(m postMocks, ctx context.Context, profileID, accountID int64) {
	m.userClient.EXPECT().
		GetProfileSummary(ctx, &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{
			ProfileId: profileID, UserAccountId: accountID, FirstName: "Neo", LastName: "Anderson", Username: "neo",
		}, nil)
}

func expectPostDetailsDeps(m postMocks, ctx context.Context, postID, authorID, viewerProfileID int64) {
	expectPostAuthor(m, ctx, authorID, 10)
	m.postWithMedia.EXPECT().GetMediaByPostID(ctx, postID).Return(nil)
	m.likes.EXPECT().GetLikeCountOnPost(ctx, postID).Return(3)
	if viewerProfileID > 0 {
		m.likes.EXPECT().HasActivePostLike(ctx, postID, viewerProfileID).Return(true)
	}
}

func TestCreatePost(t *testing.T) {
	ctrl, m := newPostMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	text := "hello"
	postID := int64(99)
	authorID := int64(20)

	expectPostProfile(m, ctx, 10, authorID)
	m.posts.EXPECT().Save(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, post models.Post) (int64, error) {
		require.Equal(t, authorID, post.AuthorID)
		require.Equal(t, text, *post.Text)
		require.True(t, post.AllowComments)
		return postID, nil
	})
	expectPostProfile(m, ctx, 10, authorID)
	m.posts.EXPECT().Get(ctx, postID).Return(&models.Post{ID: postID, Uid: uuid.New(), Text: &text, AuthorID: authorID, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil)
	expectPostDetailsDeps(m, ctx, postID, authorID, authorID)

	details, err := m.service.CreatePost(ctx, 10, CreateInput{Text: &text})

	require.NoError(t, err)
	require.Equal(t, postID, details.ID)
	require.Equal(t, authorID, details.AuthorID)
	require.True(t, details.IsLiked)
	require.Equal(t, 3, details.Likes)
}

func TestGetFeedSortsAndMapsPosts(t *testing.T) {
	ctrl, m := newPostMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	oldText := "<old>"
	newText := "<new>"
	now := time.Now()
	posts := []models.Post{
		{ID: 1, Uid: uuid.New(), Text: &oldText, AuthorID: 20, CreatedAt: now.Add(-time.Hour), IsPublicDemo: false},
		{ID: 2, Uid: uuid.New(), Text: &newText, AuthorID: 20, CreatedAt: now, IsPublicDemo: false},
		{ID: 3, Uid: uuid.New(), Text: &newText, AuthorID: 20, CreatedAt: now, IsPublicDemo: true},
	}
	m.posts.EXPECT().GetAll(ctx).Return(posts, nil)
	expectPostAuthor(m, ctx, 20, 10)
	m.likes.EXPECT().GetLikeCountOnPost(ctx, int64(2)).Return(5)
	m.comments.EXPECT().GetCommentCount(ctx, int64(2)).Return(6)
	m.reposts.EXPECT().GetRepostCount(ctx, int64(2)).Return(7)
	m.postWithMedia.EXPECT().GetMediaByPostID(ctx, int64(2)).Return(nil)
	expectPostAuthor(m, ctx, 20, 10)
	m.likes.EXPECT().GetLikeCountOnPost(ctx, int64(1)).Return(1)
	m.comments.EXPECT().GetCommentCount(ctx, int64(1)).Return(2)
	m.reposts.EXPECT().GetRepostCount(ctx, int64(1)).Return(3)
	m.postWithMedia.EXPECT().GetMediaByPostID(ctx, int64(1)).Return(nil)

	feed, err := m.service.GetFeed(ctx, "", 10)

	require.NoError(t, err)
	require.Len(t, feed.Posts, 2)
	require.Equal(t, int64(2), feed.Posts[0].ID)
	require.Equal(t, "&lt;new&gt;", feed.Posts[0].Text)
	require.Equal(t, 5, feed.Posts[0].Likes)
	require.False(t, feed.HasMore)
}

func TestPostReadAPIs(t *testing.T) {
	ctrl, m := newPostMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	text := "hello"
	postID := int64(99)
	authorID := int64(20)
	avatarID := int64(8)
	mediaID := int64(7)
	now := time.Now()

	m.posts.EXPECT().Get(ctx, postID).Return(&models.Post{ID: postID, Uid: uuid.New(), Text: &text, AuthorID: authorID, CreatedAt: now, UpdatedAt: now}, nil)
	m.userClient.EXPECT().GetProfileSummary(ctx, &userpb.GetProfileSummaryRequest{ProfileId: authorID}).Return(&userpb.GetProfileSummaryResponse{
		ProfileId: authorID, UserAccountId: 10, FirstName: "Neo", LastName: "Anderson", Username: "neo", AvatarId: &avatarID,
	}, nil)
	m.mediaClient.EXPECT().GetMediaURL(ctx, &mediapb.GetMediaURLRequest{MediaId: avatarID}).Return(&mediapb.GetMediaURLResponse{Url: "https://cdn.test/avatar.png"}, nil)
	m.postWithMedia.EXPECT().GetMediaByPostID(ctx, postID).Return([]int64{mediaID})
	m.mediaClient.EXPECT().GetMedia(ctx, &mediapb.GetMediaRequest{MediaId: mediaID}).Return(&mediapb.GetMediaResponse{
		MediaId: mediaID, Uid: "media-uid", MimeType: "image/png", Url: "/media/7",
	}, nil)
	m.mediaClient.EXPECT().GetMediaURL(ctx, &mediapb.GetMediaURLRequest{MediaId: mediaID}).Return(&mediapb.GetMediaURLResponse{Url: "https://cdn.test/media.png"}, nil)
	m.likes.EXPECT().GetLikeCountOnPost(ctx, postID).Return(4)

	details, err := m.service.GetPost(ctx, postID)
	require.NoError(t, err)
	require.Equal(t, "https://cdn.test/avatar.png", *details.Author.AvatarURL)
	require.Len(t, details.Media, 1)
	require.Equal(t, "https://cdn.test/media.png", details.Media[0].URL)

	expectPostProfile(m, ctx, 10, authorID)
	m.posts.EXPECT().GetByAuthorID(ctx, authorID).Return([]models.Post{
		{ID: postID, Uid: uuid.New(), Text: &text, AuthorID: authorID},
		{ID: 100, Uid: uuid.New(), Text: &text, AuthorID: authorID, CommunityID: ptrInt64(1)},
	}, nil)
	expectPostDetailsDeps(m, ctx, postID, authorID, 0)

	myPosts, err := m.service.GetMyPosts(ctx, 10)
	require.NoError(t, err)
	require.Len(t, myPosts, 1)

	m.communities.EXPECT().Get(ctx, int64(1)).Return(&models.Community{ID: 1, ProfileID: authorID}, nil)
	m.posts.EXPECT().GetByCommunityID(ctx, int64(1)).Return([]models.Post{{ID: postID, Uid: uuid.New(), Text: &text, AuthorID: authorID}}, nil)
	expectPostDetailsDeps(m, ctx, postID, authorID, 0)
	communityPosts, err := m.service.GetCommunityOfficialPosts(ctx, 1)
	require.NoError(t, err)
	require.Len(t, communityPosts, 1)
}

func ptrInt64(value int64) *int64 {
	return &value
}

func TestUpdatePostReplacesTextAndMedia(t *testing.T) {
	ctrl, m := newPostMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	oldText := "old"
	newText := "new"
	postID := int64(99)
	authorID := int64(20)

	expectPostProfile(m, ctx, 10, authorID)
	m.posts.EXPECT().Get(ctx, postID).Return(&models.Post{ID: postID, Uid: uuid.New(), Text: &oldText, AuthorID: authorID}, nil)
	m.posts.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, post models.Post) error {
		require.Equal(t, newText, *post.Text)
		return nil
	})
	m.postWithMedia.EXPECT().DeleteByPostID(ctx, postID).Return(nil)
	m.mediaClient.EXPECT().GetMedia(ctx, &mediapb.GetMediaRequest{MediaId: 7}).Return(&mediapb.GetMediaResponse{MediaId: 7, Uid: "m7", MimeType: "image/png", Url: "http://cdn.test/7.png"}, nil)
	m.postWithMedia.EXPECT().Save(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, item models.PostWithMedia) error {
		require.Equal(t, postID, item.PostID)
		require.Equal(t, int64(7), item.MediaID)
		require.Equal(t, 0, item.Order)
		return nil
	})
	expectPostProfile(m, ctx, 10, authorID)
	m.posts.EXPECT().Get(ctx, postID).Return(&models.Post{ID: postID, Uid: uuid.New(), Text: &newText, AuthorID: authorID}, nil)
	expectPostDetailsDeps(m, ctx, postID, authorID, authorID)

	details, err := m.service.UpdatePost(ctx, 10, postID, UpdateInput{
		Text:  &newText,
		Media: []dto.MediaRequestData{{MediaID: 7}},
	})

	require.NoError(t, err)
	require.Equal(t, newText, *details.Text)
}

func TestLikeAndUnlikePost(t *testing.T) {
	ctrl, m := newPostMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	text := "liked"
	postID := int64(99)
	authorID := int64(20)
	likeID := int64(5)

	expectPostProfile(m, ctx, 10, authorID)
	m.posts.EXPECT().Get(ctx, postID).Return(&models.Post{ID: postID}, nil)
	m.likes.EXPECT().GetPostLikeByAuthor(ctx, postID, authorID).Return(&models.Like{ID: likeID, IsActive: false}, nil)
	m.likes.EXPECT().SetActive(ctx, likeID, true).Return(nil)
	expectPostProfile(m, ctx, 10, authorID)
	m.posts.EXPECT().Get(ctx, postID).Return(&models.Post{ID: postID, Uid: uuid.New(), Text: &text, AuthorID: authorID}, nil)
	expectPostDetailsDeps(m, ctx, postID, authorID, authorID)

	details, err := m.service.LikePost(ctx, 10, postID)
	require.NoError(t, err)
	require.Equal(t, postID, details.ID)

	expectPostProfile(m, ctx, 10, authorID)
	m.posts.EXPECT().Get(ctx, postID).Return(&models.Post{ID: postID}, nil)
	m.likes.EXPECT().GetPostLikeByAuthor(ctx, postID, authorID).Return(&models.Like{ID: likeID, IsActive: true}, nil)
	m.likes.EXPECT().SetActive(ctx, likeID, false).Return(nil)
	expectPostProfile(m, ctx, 10, authorID)
	m.posts.EXPECT().Get(ctx, postID).Return(&models.Post{ID: postID, Uid: uuid.New(), Text: &text, AuthorID: authorID}, nil)
	expectPostDetailsDeps(m, ctx, postID, authorID, authorID)

	details, err = m.service.UnlikePost(ctx, 10, postID)
	require.NoError(t, err)
	require.Equal(t, postID, details.ID)
}

func TestPostValidationAndStatus(t *testing.T) {
	_, m := newPostMocks(t)
	ctx := context.Background()

	details, err := m.service.CreatePost(ctx, 0, CreateInput{})
	require.Nil(t, details)
	require.ErrorIs(t, err, ErrInvalidInput)

	details, err = m.service.CreatePost(ctx, 10, CreateInput{})
	require.Nil(t, details)
	require.ErrorIs(t, err, ErrPostContentRequired)

	feed, err := m.service.GetFeed(ctx, "bad", 10)
	require.Empty(t, feed.Posts)
	require.ErrorIs(t, err, ErrInvalidInput)

	require.Equal(t, codes.InvalidArgument, status.Code(ToStatus(ErrInvalidInput)))
	require.Equal(t, codes.NotFound, status.Code(ToStatus(ErrPostNotFound)))
	require.Equal(t, codes.PermissionDenied, status.Code(ToStatus(ErrForbidden)))
	require.Equal(t, codes.Internal, status.Code(ToStatus(errors.New("boom"))))
}
