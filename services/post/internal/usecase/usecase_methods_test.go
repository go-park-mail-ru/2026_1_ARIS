package usecase

import (
	"context"
	"errors"
	"testing"

	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/repository"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/repository/mocks"
	"github.com/golang/mock/gomock"
)

// newPostService creates a Service wired to fresh mocks.
func newPostService(ctrl *gomock.Controller) (
	*Service,
	*repomocks.MockPostRepo,
	*repomocks.MockPostMediaRepo,
	*repomocks.MockCommentRepo,
	*repomocks.MockLikeRepo,
	*repomocks.MockRepostRepo,
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
	svc := New(store, users, nil, nil)
	return svc, posts, postMedia, comments, likes, reposts, users
}

// expectGetPostForViewer sets up all the mocks needed for a full GetPostForViewer call.
// userAccountID → profileID via GetProfileByUserAccount, then Get post, postMedia, likes, comments.
// It also expects GetProfileSummary for the author lookup (author() method).
func expectGetPostForViewer(
	userAccountID, profileID, postID int64,
	post *model.Post,
	users *usermock.MockUserServiceClient,
	posts *repomocks.MockPostRepo,
	postMedia *repomocks.MockPostMediaRepo,
	likes *repomocks.MockLikeRepo,
	comments *repomocks.MockCommentRepo,
) {
	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)
	posts.EXPECT().Get(gomock.Any(), postID).Return(post, nil)
	users.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: post.AuthorID}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: post.AuthorID}, nil)
	postMedia.EXPECT().GetDetailedMediaByPostID(gomock.Any(), postID).Return(nil, nil)
	likes.EXPECT().GetLikeCountOnPost(gomock.Any(), postID).Return(0)
	likes.EXPECT().HasActivePostLike(gomock.Any(), postID, profileID).Return(false)
	comments.EXPECT().GetCommentCount(gomock.Any(), postID).Return(0)
}

// ---------------------------------------------------------------------------
// GetPost tests
// ---------------------------------------------------------------------------

func TestGetPost_InvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _ := newPostService(ctrl)

	_, err := svc.GetPost(context.Background(), 0)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("GetPost(0) want ErrInvalidInput, got %v", err)
	}

	_, err = svc.GetPost(context.Background(), -1)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("GetPost(-1) want ErrInvalidInput, got %v", err)
	}
}

func TestGetPost_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, _, _, _, _, _ := newPostService(ctrl)

	posts.EXPECT().Get(gomock.Any(), int64(99)).Return(nil, repository.ErrPostNotFound)

	_, err := svc.GetPost(context.Background(), 99)
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("GetPost(99) want ErrPostNotFound, got %v", err)
	}
}

func TestGetPost_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// userClient is nil so author returns Author{ID: authorID}
	posts := repomocks.NewMockPostRepo(ctrl)
	postMedia := repomocks.NewMockPostMediaRepo(ctrl)
	comments := repomocks.NewMockCommentRepo(ctrl)
	likes := repomocks.NewMockLikeRepo(ctrl)
	reposts := repomocks.NewMockRepostRepo(ctrl)
	memberships := repomocks.NewMockMembershipRepo(ctrl)

	store := repository.Store{
		Posts:       posts,
		PostMedia:   postMedia,
		Comments:    comments,
		Likes:       likes,
		Reposts:     reposts,
		Memberships: memberships,
	}
	svc := New(store, nil, nil, nil)

	post := &model.Post{ID: 5, AuthorID: 20, AllowComments: true}
	posts.EXPECT().Get(gomock.Any(), int64(5)).Return(post, nil)
	postMedia.EXPECT().GetDetailedMediaByPostID(gomock.Any(), int64(5)).Return(nil, nil)
	likes.EXPECT().GetLikeCountOnPost(gomock.Any(), int64(5)).Return(5)
	comments.EXPECT().GetCommentCount(gomock.Any(), int64(5)).Return(3)

	details, err := svc.GetPost(context.Background(), 5)
	if err != nil {
		t.Fatalf("GetPost() error = %v", err)
	}
	if details.ID != 5 {
		t.Errorf("GetPost() ID = %d, want 5", details.ID)
	}
	if details.Likes != 5 {
		t.Errorf("GetPost() Likes = %d, want 5", details.Likes)
	}
	if details.Comments != 3 {
		t.Errorf("GetPost() Comments = %d, want 3", details.Comments)
	}
}

// ---------------------------------------------------------------------------
// GetPostForViewer tests
// ---------------------------------------------------------------------------

func TestGetPostForViewer_InvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _ := newPostService(ctrl)
	ctx := context.Background()

	cases := [][2]int64{{0, 1}, {1, 0}, {0, 0}, {-1, 5}, {5, -1}}
	for _, tc := range cases {
		_, err := svc.GetPostForViewer(ctx, tc[0], tc[1])
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("GetPostForViewer(%d,%d) want ErrInvalidInput, got %v", tc[0], tc[1], err)
		}
	}
}

func TestGetPostForViewer_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, postMedia, comments, likes, _, users := newPostService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 10
		profileID     int64 = 20
		postID        int64 = 5
	)

	post := &model.Post{ID: postID, AuthorID: profileID, AllowComments: true}

	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)

	posts.EXPECT().Get(gomock.Any(), postID).Return(post, nil)

	// GetProfileSummary called by author() inside buildPostDetails
	users.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: profileID}, nil)

	postMedia.EXPECT().GetDetailedMediaByPostID(gomock.Any(), postID).Return(nil, nil)
	likes.EXPECT().GetLikeCountOnPost(gomock.Any(), postID).Return(0)
	likes.EXPECT().HasActivePostLike(gomock.Any(), postID, profileID).Return(false)
	comments.EXPECT().GetCommentCount(gomock.Any(), postID).Return(0)

	details, err := svc.GetPostForViewer(ctx, postID, userAccountID)
	if err != nil {
		t.Fatalf("GetPostForViewer() error = %v", err)
	}
	if details.ID != postID {
		t.Errorf("GetPostForViewer() ID = %d, want %d", details.ID, postID)
	}
}

// ---------------------------------------------------------------------------
// DeletePost tests
// ---------------------------------------------------------------------------

func TestDeletePost_InvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _ := newPostService(ctrl)
	ctx := context.Background()

	cases := [][2]int64{{0, 1}, {1, 0}, {-1, 5}}
	for _, tc := range cases {
		err := svc.DeletePost(ctx, tc[0], tc[1])
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("DeletePost(%d,%d) want ErrInvalidInput, got %v", tc[0], tc[1], err)
		}
	}
}

func TestDeletePost_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, _, _, _, _, users := newPostService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 10
		profileID     int64 = 20
		postID        int64 = 5
	)

	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)

	post := &model.Post{ID: postID, AuthorID: profileID}
	posts.EXPECT().Get(gomock.Any(), postID).Return(post, nil)
	posts.EXPECT().Delete(gomock.Any(), postID).Return(nil)

	err := svc.DeletePost(ctx, userAccountID, postID)
	if err != nil {
		t.Fatalf("DeletePost() error = %v", err)
	}
}

func TestDeletePost_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, _, _, _, _, users := newPostService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 10
		profileID     int64 = 20
		otherAuthor   int64 = 99
		postID        int64 = 5
	)

	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)

	// Post authored by someone else, no community — canDeletePost returns false
	post := &model.Post{ID: postID, AuthorID: otherAuthor}
	posts.EXPECT().Get(gomock.Any(), postID).Return(post, nil)

	err := svc.DeletePost(ctx, userAccountID, postID)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("DeletePost() want ErrForbidden, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// LikePost tests
// ---------------------------------------------------------------------------

func TestLikePost_InvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _ := newPostService(ctrl)
	ctx := context.Background()

	cases := [][2]int64{{0, 1}, {1, 0}, {0, 0}}
	for _, tc := range cases {
		_, err := svc.LikePost(ctx, tc[0], tc[1])
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("LikePost(%d,%d) want ErrInvalidInput, got %v", tc[0], tc[1], err)
		}
	}
}

func TestLikePost_NewLike_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, postMedia, comments, likes, _, users := newPostService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 10
		profileID     int64 = 20
		postID        int64 = 5
	)

	// profileIDByUserAccount called twice: once for LikePost, once for GetPostForViewer
	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil).
		Times(2)

	post := &model.Post{ID: postID, AuthorID: profileID, AllowComments: true}
	posts.EXPECT().Get(gomock.Any(), postID).Return(post, nil).Times(2)

	// No existing like
	likes.EXPECT().GetPostLikeByAuthor(gomock.Any(), postID, profileID).Return(nil, errors.New("not found"))
	likes.EXPECT().Save(gomock.Any(), gomock.Any()).Return(int64(1), nil)

	// GetPostForViewer at the end — author() calls GetProfileSummary
	users.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: profileID}, nil)
	postMedia.EXPECT().GetDetailedMediaByPostID(gomock.Any(), postID).Return(nil, nil)
	likes.EXPECT().GetLikeCountOnPost(gomock.Any(), postID).Return(1)
	likes.EXPECT().HasActivePostLike(gomock.Any(), postID, profileID).Return(true)
	comments.EXPECT().GetCommentCount(gomock.Any(), postID).Return(0)

	details, err := svc.LikePost(ctx, userAccountID, postID)
	if err != nil {
		t.Fatalf("LikePost() error = %v", err)
	}
	if details == nil {
		t.Fatal("LikePost() returned nil details")
	}
}

func TestLikePost_AlreadyLiked_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, postMedia, comments, likes, _, users := newPostService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 10
		profileID     int64 = 20
		postID        int64 = 5
	)

	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil).
		Times(2)

	post := &model.Post{ID: postID, AuthorID: profileID, AllowComments: true}
	posts.EXPECT().Get(gomock.Any(), postID).Return(post, nil).Times(2)

	// Already active like — no SetActive called
	existingLike := &model.Like{ID: 100, IsActive: true}
	likes.EXPECT().GetPostLikeByAuthor(gomock.Any(), postID, profileID).Return(existingLike, nil)

	// GetPostForViewer at the end
	users.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: profileID}, nil)
	postMedia.EXPECT().GetDetailedMediaByPostID(gomock.Any(), postID).Return(nil, nil)
	likes.EXPECT().GetLikeCountOnPost(gomock.Any(), postID).Return(1)
	likes.EXPECT().HasActivePostLike(gomock.Any(), postID, profileID).Return(true)
	comments.EXPECT().GetCommentCount(gomock.Any(), postID).Return(0)

	details, err := svc.LikePost(ctx, userAccountID, postID)
	if err != nil {
		t.Fatalf("LikePost() error = %v", err)
	}
	if details == nil {
		t.Fatal("LikePost() returned nil details")
	}
}

func TestLikePost_Reactivate_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, postMedia, comments, likes, _, users := newPostService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 10
		profileID     int64 = 20
		postID        int64 = 5
		likeID        int64 = 100
	)

	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil).
		Times(2)

	post := &model.Post{ID: postID, AuthorID: profileID, AllowComments: true}
	posts.EXPECT().Get(gomock.Any(), postID).Return(post, nil).Times(2)

	// Inactive like — should be reactivated
	existingLike := &model.Like{ID: likeID, IsActive: false}
	likes.EXPECT().GetPostLikeByAuthor(gomock.Any(), postID, profileID).Return(existingLike, nil)
	likes.EXPECT().SetActive(gomock.Any(), likeID, true).Return(nil)

	// GetPostForViewer at the end
	users.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: profileID}, nil)
	postMedia.EXPECT().GetDetailedMediaByPostID(gomock.Any(), postID).Return(nil, nil)
	likes.EXPECT().GetLikeCountOnPost(gomock.Any(), postID).Return(1)
	likes.EXPECT().HasActivePostLike(gomock.Any(), postID, profileID).Return(true)
	comments.EXPECT().GetCommentCount(gomock.Any(), postID).Return(0)

	details, err := svc.LikePost(ctx, userAccountID, postID)
	if err != nil {
		t.Fatalf("LikePost() error = %v", err)
	}
	if details == nil {
		t.Fatal("LikePost() returned nil details")
	}
}

// ---------------------------------------------------------------------------
// UnlikePost tests
// ---------------------------------------------------------------------------

func TestUnlikePost_InvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _ := newPostService(ctrl)
	ctx := context.Background()

	cases := [][2]int64{{0, 1}, {1, 0}, {0, 0}}
	for _, tc := range cases {
		_, err := svc.UnlikePost(ctx, tc[0], tc[1])
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("UnlikePost(%d,%d) want ErrInvalidInput, got %v", tc[0], tc[1], err)
		}
	}
}

func TestUnlikePost_HasActiveLike_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, postMedia, comments, likes, _, users := newPostService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 10
		profileID     int64 = 20
		postID        int64 = 5
		likeID        int64 = 50
	)

	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil).
		Times(2)

	post := &model.Post{ID: postID, AuthorID: profileID, AllowComments: true}
	posts.EXPECT().Get(gomock.Any(), postID).Return(post, nil).Times(2)

	existingLike := &model.Like{ID: likeID, IsActive: true}
	likes.EXPECT().GetPostLikeByAuthor(gomock.Any(), postID, profileID).Return(existingLike, nil)
	likes.EXPECT().SetActive(gomock.Any(), likeID, false).Return(nil)

	// GetPostForViewer at the end
	users.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: profileID}, nil)
	postMedia.EXPECT().GetDetailedMediaByPostID(gomock.Any(), postID).Return(nil, nil)
	likes.EXPECT().GetLikeCountOnPost(gomock.Any(), postID).Return(0)
	likes.EXPECT().HasActivePostLike(gomock.Any(), postID, profileID).Return(false)
	comments.EXPECT().GetCommentCount(gomock.Any(), postID).Return(0)

	details, err := svc.UnlikePost(ctx, userAccountID, postID)
	if err != nil {
		t.Fatalf("UnlikePost() error = %v", err)
	}
	if details == nil {
		t.Fatal("UnlikePost() returned nil details")
	}
}

func TestUnlikePost_NoLike_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, postMedia, comments, likes, _, users := newPostService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 10
		profileID     int64 = 20
		postID        int64 = 5
	)

	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil).
		Times(2)

	post := &model.Post{ID: postID, AuthorID: profileID, AllowComments: true}
	posts.EXPECT().Get(gomock.Any(), postID).Return(post, nil).Times(2)

	likes.EXPECT().GetPostLikeByAuthor(gomock.Any(), postID, profileID).Return(nil, errors.New("not found"))

	// GetPostForViewer at the end
	users.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: profileID}, nil)
	postMedia.EXPECT().GetDetailedMediaByPostID(gomock.Any(), postID).Return(nil, nil)
	likes.EXPECT().GetLikeCountOnPost(gomock.Any(), postID).Return(0)
	likes.EXPECT().HasActivePostLike(gomock.Any(), postID, profileID).Return(false)
	comments.EXPECT().GetCommentCount(gomock.Any(), postID).Return(0)

	details, err := svc.UnlikePost(ctx, userAccountID, postID)
	if err != nil {
		t.Fatalf("UnlikePost() error = %v", err)
	}
	if details == nil {
		t.Fatal("UnlikePost() returned nil details")
	}
}

// ---------------------------------------------------------------------------
// GetPostComments tests
// ---------------------------------------------------------------------------

func TestGetPostComments_InvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _ := newPostService(ctrl)
	ctx := context.Background()

	cases := [][2]int64{{0, 5}, {5, 0}, {0, 0}}
	for _, tc := range cases {
		_, err := svc.GetPostComments(ctx, tc[0], tc[1], 10, 0)
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("GetPostComments(%d,%d) want ErrInvalidInput, got %v", tc[0], tc[1], err)
		}
	}
}

func TestGetPostComments_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, _, comments, likes, _, users := newPostService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 10
		profileID     int64 = 20
		postID        int64 = 5
	)

	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)

	post := &model.Post{ID: postID, AuthorID: profileID, AllowComments: true}
	posts.EXPECT().Get(gomock.Any(), postID).Return(post, nil)

	txt := "hello"
	returnedComments := []model.Comment{{ID: 1, PostID: postID, AuthorID: profileID, Text: &txt}}
	comments.EXPECT().GetTopLevelByPostID(gomock.Any(), postID, 50, 0).Return(returnedComments, nil)

	commentIDs := []int64{1}
	likes.EXPECT().GetCommentLikeCountBatch(gomock.Any(), commentIDs).Return(map[int64]int{1: 2}, nil)
	likes.EXPECT().GetCommentViewerLikesBatch(gomock.Any(), commentIDs, profileID).Return(map[int64]bool{1: false}, nil)

	// mapComment calls author() → GetProfileSummary for the comment author
	users.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: profileID}, nil)

	result, err := svc.GetPostComments(ctx, userAccountID, postID, 0, 0)
	if err != nil {
		t.Fatalf("GetPostComments() error = %v", err)
	}
	if len(result) != 1 {
		t.Errorf("GetPostComments() len = %d, want 1", len(result))
	}
}

// ---------------------------------------------------------------------------
// CreateComment tests
// ---------------------------------------------------------------------------

func TestCreateComment_InvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _ := newPostService(ctrl)
	ctx := context.Background()

	// empty text
	_, err := svc.CreateComment(ctx, 10, 5, "", nil)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("CreateComment with empty text want ErrInvalidInput, got %v", err)
	}

	// invalid userAccountID
	_, err = svc.CreateComment(ctx, 0, 5, "hello", nil)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("CreateComment with userAccountID=0 want ErrInvalidInput, got %v", err)
	}

	// invalid postID
	_, err = svc.CreateComment(ctx, 10, 0, "hello", nil)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("CreateComment with postID=0 want ErrInvalidInput, got %v", err)
	}
}

func TestCreateComment_PostNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, _, _, _, _, _ := newPostService(ctrl)
	ctx := context.Background()

	posts.EXPECT().Get(gomock.Any(), int64(5)).Return(nil, repository.ErrPostNotFound)

	_, err := svc.CreateComment(ctx, 10, 5, "hello", nil)
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("CreateComment post not found want ErrPostNotFound, got %v", err)
	}
}

func TestCreateComment_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, _, comments, likes, _, users := newPostService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 10
		profileID     int64 = 20
		postID        int64 = 5
		commentID     int64 = 99
	)

	post := &model.Post{ID: postID, AuthorID: profileID, AllowComments: true}
	posts.EXPECT().Get(gomock.Any(), postID).Return(post, nil)

	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)

	comments.EXPECT().Save(gomock.Any(), gomock.Any()).Return(commentID, nil)

	txt := "hello"
	savedComment := &model.Comment{ID: commentID, PostID: postID, AuthorID: profileID, Text: &txt}
	comments.EXPECT().Get(gomock.Any(), commentID).Return(savedComment, nil)

	commentIDs := []int64{commentID}
	likes.EXPECT().GetCommentLikeCountBatch(gomock.Any(), commentIDs).Return(map[int64]int{}, nil)
	likes.EXPECT().GetCommentViewerLikesBatch(gomock.Any(), commentIDs, profileID).Return(map[int64]bool{}, nil)

	// mapComment calls author() → GetProfileSummary for the comment author
	users.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: profileID}, nil)

	result, err := svc.CreateComment(ctx, userAccountID, postID, "hello", nil)
	if err != nil {
		t.Fatalf("CreateComment() error = %v", err)
	}
	if result == nil {
		t.Fatal("CreateComment() returned nil")
	}
	if result.ID != commentID {
		t.Errorf("CreateComment() ID = %d, want %d", result.ID, commentID)
	}
}

// ---------------------------------------------------------------------------
// UpdateComment tests
// ---------------------------------------------------------------------------

func TestUpdateComment_InvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _ := newPostService(ctrl)
	ctx := context.Background()

	// empty text
	_, err := svc.UpdateComment(ctx, 10, 5, 1, "")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("UpdateComment with empty text want ErrInvalidInput, got %v", err)
	}

	// invalid userAccountID
	_, err = svc.UpdateComment(ctx, 0, 5, 1, "hi")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("UpdateComment with userAccountID=0 want ErrInvalidInput, got %v", err)
	}

	// invalid postID
	_, err = svc.UpdateComment(ctx, 10, 0, 1, "hi")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("UpdateComment with postID=0 want ErrInvalidInput, got %v", err)
	}

	// invalid commentID
	_, err = svc.UpdateComment(ctx, 10, 5, 0, "hi")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("UpdateComment with commentID=0 want ErrInvalidInput, got %v", err)
	}
}

func TestUpdateComment_CommentNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, comments, _, _, users := newPostService(ctrl)
	ctx := context.Background()

	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: int64(10)}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: int64(20)}, nil)

	comments.EXPECT().Get(gomock.Any(), int64(1)).Return(nil, repository.ErrCommentNotFound)

	_, err := svc.UpdateComment(ctx, 10, 5, 1, "new text")
	if !errors.Is(err, ErrCommentNotFound) {
		t.Fatalf("UpdateComment not found want ErrCommentNotFound, got %v", err)
	}
}

func TestUpdateComment_ForbiddenEdit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, comments, _, _, users := newPostService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 10
		profileID     int64 = 20
		otherAuthor   int64 = 99
		postID        int64 = 5
		commentID     int64 = 1
	)

	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)

	comment := &model.Comment{ID: commentID, PostID: postID, AuthorID: otherAuthor}
	comments.EXPECT().Get(gomock.Any(), commentID).Return(comment, nil)

	_, err := svc.UpdateComment(ctx, userAccountID, postID, commentID, "new text")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("UpdateComment forbidden want ErrForbidden, got %v", err)
	}
}

func TestUpdateComment_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, comments, likes, _, users := newPostService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 10
		profileID     int64 = 20
		postID        int64 = 5
		commentID     int64 = 1
	)

	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)

	oldTxt := "old"
	comment := &model.Comment{ID: commentID, PostID: postID, AuthorID: profileID, Text: &oldTxt}
	comments.EXPECT().Get(gomock.Any(), commentID).Return(comment, nil)
	comments.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	newTxt := "new text"
	updated := &model.Comment{ID: commentID, PostID: postID, AuthorID: profileID, Text: &newTxt}
	comments.EXPECT().Get(gomock.Any(), commentID).Return(updated, nil)

	commentIDs := []int64{commentID}
	likes.EXPECT().GetCommentLikeCountBatch(gomock.Any(), commentIDs).Return(map[int64]int{}, nil)
	likes.EXPECT().GetCommentViewerLikesBatch(gomock.Any(), commentIDs, profileID).Return(map[int64]bool{}, nil)

	// mapComment calls author() → GetProfileSummary
	users.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: profileID}, nil)

	result, err := svc.UpdateComment(ctx, userAccountID, postID, commentID, "new text")
	if err != nil {
		t.Fatalf("UpdateComment() error = %v", err)
	}
	if result == nil {
		t.Fatal("UpdateComment() returned nil")
	}
}

// ---------------------------------------------------------------------------
// DeleteComment tests
// ---------------------------------------------------------------------------

func TestDeleteComment_InvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _ := newPostService(ctrl)
	ctx := context.Background()

	cases := [][3]int64{{0, 5, 1}, {10, 0, 1}, {10, 5, 0}}
	for _, tc := range cases {
		err := svc.DeleteComment(ctx, tc[0], tc[1], tc[2])
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("DeleteComment(%d,%d,%d) want ErrInvalidInput, got %v", tc[0], tc[1], tc[2], err)
		}
	}
}

func TestDeleteComment_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, _, comments, _, _, users := newPostService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 10
		profileID     int64 = 20
		postID        int64 = 5
		commentID     int64 = 1
	)

	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)

	comment := &model.Comment{ID: commentID, PostID: postID, AuthorID: profileID}
	comments.EXPECT().Get(gomock.Any(), commentID).Return(comment, nil)

	post := &model.Post{ID: postID, AuthorID: profileID}
	posts.EXPECT().Get(gomock.Any(), postID).Return(post, nil)

	comments.EXPECT().Delete(gomock.Any(), commentID).Return(nil)

	err := svc.DeleteComment(ctx, userAccountID, postID, commentID)
	if err != nil {
		t.Fatalf("DeleteComment() error = %v", err)
	}
}

// ---------------------------------------------------------------------------
// LikeComment tests
// ---------------------------------------------------------------------------

func TestLikeComment_InvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _ := newPostService(ctrl)
	ctx := context.Background()

	cases := [][3]int64{{0, 5, 1}, {10, 0, 1}, {10, 5, 0}}
	for _, tc := range cases {
		_, err := svc.LikeComment(ctx, tc[0], tc[1], tc[2])
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("LikeComment(%d,%d,%d) want ErrInvalidInput, got %v", tc[0], tc[1], tc[2], err)
		}
	}
}

func TestLikeComment_Success_NewLike(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, _, comments, likes, _, users := newPostService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 10
		profileID     int64 = 20
		postID        int64 = 5
		commentID     int64 = 1
	)

	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)

	post := &model.Post{ID: postID, AuthorID: profileID}
	posts.EXPECT().Get(gomock.Any(), postID).Return(post, nil)

	txt := "comment"
	comment := &model.Comment{ID: commentID, PostID: postID, AuthorID: profileID, Text: &txt}
	comments.EXPECT().Get(gomock.Any(), commentID).Return(comment, nil)

	likes.EXPECT().GetCommentLikeByAuthor(gomock.Any(), commentID, profileID).Return(nil, errors.New("not found"))
	likes.EXPECT().Save(gomock.Any(), gomock.Any()).Return(int64(200), nil)

	// mapComments calls
	comments.EXPECT().Get(gomock.Any(), commentID).Return(comment, nil)
	commentIDs := []int64{commentID}
	likes.EXPECT().GetCommentLikeCountBatch(gomock.Any(), commentIDs).Return(map[int64]int{commentID: 1}, nil)
	likes.EXPECT().GetCommentViewerLikesBatch(gomock.Any(), commentIDs, profileID).Return(map[int64]bool{commentID: true}, nil)

	// mapComment calls author() → GetProfileSummary
	users.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: profileID}, nil)

	result, err := svc.LikeComment(ctx, userAccountID, postID, commentID)
	if err != nil {
		t.Fatalf("LikeComment() error = %v", err)
	}
	if result == nil {
		t.Fatal("LikeComment() returned nil")
	}
}

// ---------------------------------------------------------------------------
// UnlikeComment tests
// ---------------------------------------------------------------------------

func TestUnlikeComment_InvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _ := newPostService(ctrl)
	ctx := context.Background()

	cases := [][3]int64{{0, 5, 1}, {10, 0, 1}, {10, 5, 0}}
	for _, tc := range cases {
		_, err := svc.UnlikeComment(ctx, tc[0], tc[1], tc[2])
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("UnlikeComment(%d,%d,%d) want ErrInvalidInput, got %v", tc[0], tc[1], tc[2], err)
		}
	}
}

func TestUnlikeComment_Success_ActiveLike(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, _, comments, likes, _, users := newPostService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 10
		profileID     int64 = 20
		postID        int64 = 5
		commentID     int64 = 1
		likeID        int64 = 300
	)

	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)

	post := &model.Post{ID: postID, AuthorID: profileID}
	posts.EXPECT().Get(gomock.Any(), postID).Return(post, nil)

	txt := "comment"
	comment := &model.Comment{ID: commentID, PostID: postID, AuthorID: profileID, Text: &txt}
	comments.EXPECT().Get(gomock.Any(), commentID).Return(comment, nil)

	existingLike := &model.Like{ID: likeID, IsActive: true}
	likes.EXPECT().GetCommentLikeByAuthor(gomock.Any(), commentID, profileID).Return(existingLike, nil)
	likes.EXPECT().SetActive(gomock.Any(), likeID, false).Return(nil)

	// mapComments calls
	comments.EXPECT().Get(gomock.Any(), commentID).Return(comment, nil)
	commentIDs := []int64{commentID}
	likes.EXPECT().GetCommentLikeCountBatch(gomock.Any(), commentIDs).Return(map[int64]int{commentID: 0}, nil)
	likes.EXPECT().GetCommentViewerLikesBatch(gomock.Any(), commentIDs, profileID).Return(map[int64]bool{commentID: false}, nil)

	// mapComment calls author() → GetProfileSummary
	users.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: profileID}, nil)

	result, err := svc.UnlikeComment(ctx, userAccountID, postID, commentID)
	if err != nil {
		t.Fatalf("UnlikeComment() error = %v", err)
	}
	if result == nil {
		t.Fatal("UnlikeComment() returned nil")
	}
}

func TestUnlikeComment_NoLike_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, posts, _, comments, likes, _, users := newPostService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 10
		profileID     int64 = 20
		postID        int64 = 5
		commentID     int64 = 1
	)

	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)

	post := &model.Post{ID: postID, AuthorID: profileID}
	posts.EXPECT().Get(gomock.Any(), postID).Return(post, nil)

	txt := "comment"
	comment := &model.Comment{ID: commentID, PostID: postID, AuthorID: profileID, Text: &txt}
	comments.EXPECT().Get(gomock.Any(), commentID).Return(comment, nil)

	likes.EXPECT().GetCommentLikeByAuthor(gomock.Any(), commentID, profileID).Return(nil, errors.New("not found"))

	// mapComments calls
	comments.EXPECT().Get(gomock.Any(), commentID).Return(comment, nil)
	commentIDs := []int64{commentID}
	likes.EXPECT().GetCommentLikeCountBatch(gomock.Any(), commentIDs).Return(map[int64]int{}, nil)
	likes.EXPECT().GetCommentViewerLikesBatch(gomock.Any(), commentIDs, profileID).Return(map[int64]bool{}, nil)

	// mapComment calls author() → GetProfileSummary
	users.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: profileID}, nil)

	result, err := svc.UnlikeComment(ctx, userAccountID, postID, commentID)
	if err != nil {
		t.Fatalf("UnlikeComment() error = %v", err)
	}
	if result == nil {
		t.Fatal("UnlikeComment() returned nil")
	}
}
