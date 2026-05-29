package usecase

import (
	"context"
	"testing"
	"time"

	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/model"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

func TestCreatePostTextOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, postMedia, comments, likes, _, users := newPostService(ctrl)
	ctx := context.Background()
	text := "created"
	now := time.Now()

	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: int64(10)}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 20}, nil)
	posts.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, post model.Post) (int64, error) {
		if post.AuthorID != 20 || post.Text == nil || *post.Text != text || !post.AllowComments {
			t.Fatalf("unexpected saved post: %+v", post)
		}
		return 99, nil
	})
	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: int64(10)}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 20}, nil)
	posts.EXPECT().Get(gomock.Any(), int64(99)).Return(&model.Post{ID: 99, Uid: uuid.New(), Text: &text, AuthorID: 20, AllowComments: true, CreatedAt: now, UpdatedAt: now}, nil)
	users.EXPECT().GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: int64(20)}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: 20, Username: "neo"}, nil)
	postMedia.EXPECT().GetDetailedMediaByPostID(gomock.Any(), int64(99)).Return(nil, nil)
	likes.EXPECT().GetLikeCountOnPost(gomock.Any(), int64(99)).Return(2)
	likes.EXPECT().HasActivePostLike(gomock.Any(), int64(99), int64(20)).Return(false)
	comments.EXPECT().GetCommentCount(gomock.Any(), int64(99)).Return(3)

	got, err := svc.CreatePost(ctx, 10, CreateInput{Text: &text})
	if err != nil {
		t.Fatalf("CreatePost() error = %v", err)
	}
	if got.ID != 99 || got.AuthorID != 20 || got.Likes != 2 || got.Comments != 3 {
		t.Fatalf("unexpected post details: %+v", got)
	}
}

func TestGetMyPostsSkipsCommunityPosts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, postMedia, comments, likes, _, users := newPostService(ctrl)
	ctx := context.Background()
	text := "mine"
	communityID := int64(7)

	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: int64(10)}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 20}, nil)
	posts.EXPECT().GetByAuthorID(gomock.Any(), int64(20)).Return([]model.Post{
		{ID: 1, Uid: uuid.New(), Text: &text, AuthorID: 20, AllowComments: true},
		{ID: 2, Uid: uuid.New(), Text: &text, AuthorID: 20, CommunityID: &communityID, AllowComments: true},
	}, nil)
	users.EXPECT().GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: int64(20)}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: 20, Username: "neo"}, nil)
	postMedia.EXPECT().GetDetailedMediaByPostID(gomock.Any(), int64(1)).Return(nil, nil)
	likes.EXPECT().GetLikeCountOnPost(gomock.Any(), int64(1)).Return(1)
	likes.EXPECT().HasActivePostLike(gomock.Any(), int64(1), int64(20)).Return(true)
	comments.EXPECT().GetCommentCount(gomock.Any(), int64(1)).Return(0)

	got, err := svc.GetMyPosts(ctx, 10)
	if err != nil {
		t.Fatalf("GetMyPosts() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != 1 || !got[0].IsLiked {
		t.Fatalf("unexpected posts: %+v", got)
	}
}

func TestUpdatePostOwnText(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, postMedia, comments, likes, _, users := newPostService(ctrl)
	ctx := context.Background()
	oldText := "old"
	newText := "new"

	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: int64(10)}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 20}, nil)
	posts.EXPECT().Get(gomock.Any(), int64(1)).Return(&model.Post{ID: 1, Uid: uuid.New(), Text: &oldText, AuthorID: 20, AllowComments: true}, nil)
	posts.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, post model.Post) error {
		if post.Text == nil || *post.Text != newText {
			t.Fatalf("unexpected updated post: %+v", post)
		}
		return nil
	})
	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: int64(10)}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 20}, nil)
	posts.EXPECT().Get(gomock.Any(), int64(1)).Return(&model.Post{ID: 1, Uid: uuid.New(), Text: &newText, AuthorID: 20, AllowComments: true}, nil)
	users.EXPECT().GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: int64(20)}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: 20, Username: "neo"}, nil)
	postMedia.EXPECT().GetDetailedMediaByPostID(gomock.Any(), int64(1)).Return(nil, nil)
	likes.EXPECT().GetLikeCountOnPost(gomock.Any(), int64(1)).Return(0)
	likes.EXPECT().HasActivePostLike(gomock.Any(), int64(1), int64(20)).Return(false)
	comments.EXPECT().GetCommentCount(gomock.Any(), int64(1)).Return(0)

	got, err := svc.UpdatePost(ctx, 10, 1, UpdateInput{Text: &newText})
	if err != nil {
		t.Fatalf("UpdatePost() error = %v", err)
	}
	if got.Text == nil || *got.Text != newText {
		t.Fatalf("unexpected details: %+v", got)
	}
}
