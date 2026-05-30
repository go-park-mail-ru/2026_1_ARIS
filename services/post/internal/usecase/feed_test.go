package usecase

import (
	"context"
	"testing"
	"time"

	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/repository"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

func newFeedPostService(ctrl *gomock.Controller) (
	*Service,
	*repomocks.MockPostRepo,
	*repomocks.MockPostMediaRepo,
	*repomocks.MockCommentRepo,
	*repomocks.MockLikeRepo,
	*repomocks.MockRepostRepo,
	*repomocks.MockMembershipRepo,
	*usermock.MockUserServiceClient,
) {
	posts := repomocks.NewMockPostRepo(ctrl)
	postMedia := repomocks.NewMockPostMediaRepo(ctrl)
	comments := repomocks.NewMockCommentRepo(ctrl)
	likes := repomocks.NewMockLikeRepo(ctrl)
	reposts := repomocks.NewMockRepostRepo(ctrl)
	memberships := repomocks.NewMockMembershipRepo(ctrl)
	users := usermock.NewMockUserServiceClient(ctrl)
	store := repository.Store{
		Posts:       posts,
		PostMedia:   postMedia,
		Comments:    comments,
		Likes:       likes,
		Reposts:     reposts,
		Memberships: memberships,
	}
	return New(store, users, nil, nil), posts, postMedia, comments, likes, reposts, memberships, users
}

func TestGetPublicFeedBuildsChronologicalBatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, postMedia, comments, likes, reposts, _, users := newFeedPostService(ctrl)
	ctx := context.Background()
	text := "<public>"
	now := time.Now()
	page := []model.Post{
		{ID: 10, Text: &text, AuthorID: 20, IsPublicDemo: true, CreatedAt: now},
		{ID: 9, Text: &text, AuthorID: 21, IsPublicDemo: true, CreatedAt: now.Add(-time.Minute)},
	}
	ids := []int64{10}

	posts.EXPECT().GetFeedPage(gomock.Any(), nil, nil, nil, 2, true).Return(page, nil)
	likes.EXPECT().GetPostLikeCountsBatch(gomock.Any(), ids).Return(map[int64]int{10: 3, 9: 1}, nil)
	likes.EXPECT().GetViewerPostLikesBatch(gomock.Any(), ids, int64(0)).Return(map[int64]bool{}, nil)
	comments.EXPECT().GetCommentCountsBatch(gomock.Any(), ids).Return(map[int64]int{10: 4}, nil)
	reposts.EXPECT().GetRepostCountsBatch(gomock.Any(), ids).Return(map[int64]int{10: 5}, nil)
	postMedia.EXPECT().GetDetailedMediaByPostIDs(gomock.Any(), ids).Return(map[int64][]model.AttachedMedia{
		10: {{MediaID: 1, UID: uuid.New(), Name: "image", MimeType: "image/png", Link: "/media/1"}},
	}, nil)
	users.EXPECT().GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: int64(20)}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: 20, Username: "author"}, nil)
	feed, err := svc.GetPublicFeed(ctx, "", 1)
	if err != nil {
		t.Fatalf("GetPublicFeed() error = %v", err)
	}
	if !feed.HasMore || feed.Cursor == "" || len(feed.Posts) != 1 {
		t.Fatalf("unexpected feed paging: %+v", feed)
	}
	if feed.Posts[0].Text != "&lt;public&gt;" || feed.Posts[0].Likes != 3 || len(feed.Posts[0].Medias) != 1 {
		t.Fatalf("unexpected mapped post: %+v", feed.Posts[0])
	}
}

func TestGetFeedWithoutFriendsReturnsEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, _, _, _, _, _, users := newFeedPostService(ctrl)
	ctx := context.Background()

	users.EXPECT().GetFriendProfileIDs(gomock.Any(), &userpb.GetFriendProfileIDsRequest{UserAccountId: int64(5)}).
		Return(&userpb.GetFriendProfileIDsResponse{}, nil)
	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: int64(5)}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 50}, nil)
	posts.EXPECT().GetFeedPage(gomock.Any(), []int64{-1}, nil, nil, 21, false).Return([]model.Post{}, nil)

	feed, err := svc.GetFeed(ctx, 5, "", "by-time", 20)
	if err != nil {
		t.Fatalf("GetFeed() error = %v", err)
	}
	if len(feed.Posts) != 0 || feed.HasMore || feed.Cursor != "" {
		t.Fatalf("expected empty feed, got %+v", feed)
	}
}
