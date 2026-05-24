package usecase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"html"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/cursor"
	communitypb "github.com/go-park-mail-ru/2026_1_ARIS/proto/community"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/analytics"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidInput         = errors.New("invalid input")
	ErrPostContentRequired  = errors.New("post content required")
	ErrPostNotFound         = repository.ErrPostNotFound
	ErrCommentNotFound      = repository.ErrCommentNotFound
	ErrProfileNotFound      = errors.New("profile not found")
	ErrCommunityNotFound    = errors.New("community not found")
	ErrForbidden            = errors.New("denied")
	ErrMediaAttachmentError = errors.New("can't attach media")
	ErrCommentsDisabled     = errors.New("comments disabled")
)

const maxAttachments = 10

type Service struct {
	store           repository.Store
	userClient      userpb.UserServiceClient
	mediaClient     mediapb.MediaServiceClient
	communityClient communitypb.CommunityServiceClient
	cache           PostCache
	analytics       *analytics.Writer
	recommendation  repository.RecommendationRepo
	sessionCache    FeedSessionCache
}

type FeedSessionCache interface {
	SaveFeedSession(ctx context.Context, sessionID string, postIDs []int64, ttl time.Duration) error
	GetFeedSession(ctx context.Context, sessionID string) ([]int64, error)
}

type PostCache interface {
	GetPostLikeCount(ctx context.Context, postID int64) (int, error)
	SetPostLikeCount(ctx context.Context, postID int64, count int) error
	DeletePostLikeCount(ctx context.Context, postID int64) error
}

type Author struct {
	ID            int64
	FirstName     string
	LastName      string
	Username      string
	UserAccountID int64
	AvatarURL     *string
}

type Media struct {
	ID       int64
	UID      string
	Name     string
	MimeType string
	URL      string
}

type FeedPost struct {
	ID        int64
	Text      string
	Author    Author
	CreatedAt time.Time
	Likes     int
	Comments  int
	Reposts   int
	IsLiked   bool
	Medias    []Media
	Files     []Media
}

type FeedResult struct {
	Posts   []FeedPost
	Cursor  string
	HasMore bool
}

type MediaRequestData struct {
	MediaID  int64  `json:"mediaID"`
	MediaURL string `json:"mediaURL"`
}

type CreateInput struct {
	Text            *string
	Media           []MediaRequestData
	Files           []MediaRequestData
	AuthorProfileID *int64
	CommunityID     *int64
}

type UpdateInput = CreateInput

type PostDetails struct {
	ID          int64
	UID         uuid.UUID
	AuthorID    int64
	CommunityID *int64
	Text        *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Author      Author
	Media       []Media
	Files       []Media
	Likes       int
	Comments    int
	IsLiked     bool
}

type Comment struct {
	ID              int64
	UID             uuid.UUID
	Text            *string
	PostID          int64
	ParentCommentID *int64
	Author          Author
	CreatedAt       time.Time
	UpdatedAt       time.Time
	RepliesCount    int
	Likes           int
	IsLiked         bool
}

type communityInfo struct {
	ID        int64
	ProfileID int64
	Title     string
	Username  string
	AvatarID  *int64
}

type communityMemberInfo struct {
	Role     string
	IsActive bool
}

func New(store repository.Store, userClient userpb.UserServiceClient, mediaClient mediapb.MediaServiceClient, communityClient communitypb.CommunityServiceClient) *Service {
	return &Service{store: store, userClient: userClient, mediaClient: mediaClient, communityClient: communityClient}
}

func (s *Service) SetCache(cache PostCache) {
	s.cache = cache
}

func (s *Service) SetAnalytics(w *analytics.Writer) {
	s.analytics = w
}

func (s *Service) SetRecommendation(repo repository.RecommendationRepo) {
	s.recommendation = repo
}

func (s *Service) SetSessionCache(sc FeedSessionCache) {
	s.sessionCache = sc
}

func (s *Service) CreatePost(ctx context.Context, userAccountID int64, input CreateInput) (*PostDetails, error) {
	if userAccountID <= 0 {
		return nil, ErrInvalidInput
	}
	attachments := appendAttachmentRefs(input.Media, input.Files)
	if input.Text == nil && len(attachments) == 0 {
		return nil, ErrPostContentRequired
	}

	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	authorID, communityID, err := s.resolvePostTarget(ctx, profileID, input.AuthorProfileID, input.CommunityID)
	if err != nil {
		return nil, err
	}

	post := model.NewPost(input.Text, authorID, false, true)
	post.CommunityID = communityID
	postID, err := s.store.Posts.Save(ctx, *post)
	if err != nil {
		return nil, err
	}
	if err := s.attachMedia(ctx, postID, profileID, attachments); err != nil {
		return nil, err
	}
	post.ID = postID
	s.emitSnapshot(*post, len(attachments) > 0)
	s.emitEvent(profileID, postID, authorID, communityID, analytics.EventPostCreated)
	return s.GetPostForViewer(ctx, postID, userAccountID)
}

func (s *Service) GetMyPosts(ctx context.Context, userAccountID int64) ([]PostDetails, error) {
	if userAccountID <= 0 {
		return nil, ErrInvalidInput
	}
	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	return s.buildProfilePostList(ctx, profileID, profileID)
}

func (s *Service) GetProfilePosts(ctx context.Context, profileID, viewerUserAccountID int64) ([]PostDetails, error) {
	if profileID <= 0 {
		return nil, ErrInvalidInput
	}
	var viewerProfileID int64
	if viewerUserAccountID > 0 {
		viewerProfileID, _ = s.profileIDByUserAccount(ctx, viewerUserAccountID)
	}
	return s.buildProfilePostList(ctx, profileID, viewerProfileID)
}

func (s *Service) buildProfilePostList(ctx context.Context, profileID, viewerProfileID int64) ([]PostDetails, error) {
	posts, err := s.store.Posts.GetByAuthorID(ctx, profileID)
	if err != nil {
		return nil, err
	}
	result := make([]PostDetails, 0, len(posts))
	for _, post := range posts {
		if post.CommunityID != nil {
			continue
		}
		details, err := s.buildPostDetails(ctx, post, viewerProfileID)
		if err == nil {
			result = append(result, *details)
		}
	}
	return result, nil
}

func (s *Service) GetCommunityPosts(ctx context.Context, communityID int64, viewerUserAccountID int64) ([]PostDetails, error) {
	if communityID <= 0 || viewerUserAccountID <= 0 {
		return nil, ErrInvalidInput
	}
	viewerProfileID, err := s.profileIDByUserAccount(ctx, viewerUserAccountID)
	if err != nil {
		return nil, err
	}
	posts, err := s.store.Posts.GetByCommunityID(ctx, communityID)
	if err != nil {
		return nil, err
	}
	result := make([]PostDetails, 0, len(posts))
	for _, post := range posts {
		details, err := s.buildPostDetails(ctx, post, viewerProfileID)
		if err == nil {
			result = append(result, *details)
		}
	}
	return result, nil
}

func (s *Service) GetCommunityOfficialPosts(ctx context.Context, communityID, viewerUserAccountID int64) ([]PostDetails, error) {
	if communityID <= 0 {
		return nil, ErrInvalidInput
	}
	var viewerProfileID int64
	if viewerUserAccountID > 0 {
		viewerProfileID, _ = s.profileIDByUserAccount(ctx, viewerUserAccountID)
	}
	community, err := s.communityByID(ctx, communityID)
	if err != nil {
		return nil, err
	}
	posts, err := s.store.Posts.GetByCommunityID(ctx, communityID)
	if err != nil {
		return nil, err
	}
	result := make([]PostDetails, 0, len(posts))
	for _, post := range posts {
		if post.AuthorID != community.ProfileID {
			continue
		}
		details, err := s.buildPostDetails(ctx, post, viewerProfileID)
		if err == nil {
			result = append(result, *details)
		}
	}
	return result, nil
}

func (s *Service) GetPost(ctx context.Context, postID int64) (*PostDetails, error) {
	if postID <= 0 {
		return nil, ErrInvalidInput
	}
	post, err := s.store.Posts.Get(ctx, postID)
	if err != nil {
		return nil, normalizePostError(err)
	}
	return s.buildPostDetails(ctx, *post, 0)
}

func (s *Service) GetPostForViewer(ctx context.Context, postID, userAccountID int64) (*PostDetails, error) {
	if postID <= 0 || userAccountID <= 0 {
		return nil, ErrInvalidInput
	}
	viewerProfileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	post, err := s.store.Posts.Get(ctx, postID)
	if err != nil {
		return nil, normalizePostError(err)
	}
	return s.buildPostDetails(ctx, *post, viewerProfileID)
}

func (s *Service) UpdatePost(ctx context.Context, userAccountID int64, postID int64, input UpdateInput) (*PostDetails, error) {
	if userAccountID <= 0 || postID <= 0 {
		return nil, ErrInvalidInput
	}
	attachments := appendAttachmentRefs(input.Media, input.Files)
	if input.Text == nil && len(attachments) == 0 {
		return nil, ErrPostContentRequired
	}
	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	post, err := s.store.Posts.Get(ctx, postID)
	if err != nil {
		return nil, normalizePostError(err)
	}
	if !s.canEditPost(ctx, *post, profileID) {
		return nil, ErrForbidden
	}
	if input.Text != nil {
		post.Text = input.Text
		post.UpdatedAt = time.Now()
		if err := s.store.Posts.Update(ctx, *post); err != nil {
			return nil, normalizePostError(err)
		}
	}
	if input.Media != nil || input.Files != nil {
		if err := s.store.PostMedia.DeleteByPostID(ctx, postID); err != nil {
			return nil, err
		}
		if err := s.attachMedia(ctx, postID, profileID, attachments); err != nil {
			return nil, err
		}
	}
	s.emitSnapshot(*post, len(attachments) > 0)
	s.emitEvent(profileID, postID, post.AuthorID, post.CommunityID, analytics.EventPostUpdated)
	return s.GetPostForViewer(ctx, postID, userAccountID)
}

func (s *Service) DeletePost(ctx context.Context, userAccountID, postID int64) error {
	if userAccountID <= 0 || postID <= 0 {
		return ErrInvalidInput
	}
	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return err
	}
	post, err := s.store.Posts.Get(ctx, postID)
	if err != nil {
		return normalizePostError(err)
	}
	if !s.canDeletePost(ctx, *post, profileID) {
		return ErrForbidden
	}
	deletedPost := *post
	deletedPost.IsActive = false
	s.emitSnapshot(deletedPost, false)
	s.emitEvent(profileID, postID, post.AuthorID, post.CommunityID, analytics.EventPostDeleted)
	if err := normalizePostError(s.store.Posts.Delete(ctx, postID)); err != nil {
		return err
	}
	s.deletePostLikeCount(ctx, postID)
	return nil
}

func (s *Service) LikePost(ctx context.Context, userAccountID, postID int64) (*PostDetails, error) {
	if userAccountID <= 0 || postID <= 0 {
		return nil, ErrInvalidInput
	}
	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	if _, err := s.store.Posts.Get(ctx, postID); err != nil {
		return nil, normalizePostError(err)
	}
	likedPost, err2 := s.store.Posts.Get(ctx, postID)
	existing, err := s.store.Likes.GetPostLikeByAuthor(ctx, postID, profileID)
	if err == nil {
		if !existing.IsActive {
			if err := s.store.Likes.SetActive(ctx, existing.ID, true); err != nil {
				return nil, err
			}
			s.refreshPostLikeCount(ctx, postID)
			if err2 == nil {
				s.emitEvent(profileID, postID, likedPost.AuthorID, likedPost.CommunityID, analytics.EventPostLike)
			}
		}
		return s.GetPostForViewer(ctx, postID, userAccountID)
	}
	if _, err := s.store.Likes.Save(ctx, *model.NewLikeToPost(postID, profileID)); err != nil {
		return nil, err
	}
	s.refreshPostLikeCount(ctx, postID)
	if err2 == nil {
		s.emitEvent(profileID, postID, likedPost.AuthorID, likedPost.CommunityID, analytics.EventPostLike)
	}
	return s.GetPostForViewer(ctx, postID, userAccountID)
}

func (s *Service) UnlikePost(ctx context.Context, userAccountID, postID int64) (*PostDetails, error) {
	if userAccountID <= 0 || postID <= 0 {
		return nil, ErrInvalidInput
	}
	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	if _, err := s.store.Posts.Get(ctx, postID); err != nil {
		return nil, normalizePostError(err)
	}
	unlikedPost, unlikedErr := s.store.Posts.Get(ctx, postID)
	existing, err := s.store.Likes.GetPostLikeByAuthor(ctx, postID, profileID)
	if err == nil && existing.IsActive {
		if err := s.store.Likes.SetActive(ctx, existing.ID, false); err != nil {
			return nil, err
		}
		s.refreshPostLikeCount(ctx, postID)
		if unlikedErr == nil {
			s.emitEvent(profileID, postID, unlikedPost.AuthorID, unlikedPost.CommunityID, analytics.EventPostUnlike)
		}
	}
	return s.GetPostForViewer(ctx, postID, userAccountID)
}

func (s *Service) GetPostComments(ctx context.Context, userAccountID, postID int64, limit, offset int) ([]Comment, error) {
	if userAccountID <= 0 || postID <= 0 {
		return nil, ErrInvalidInput
	}
	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	if _, err := s.store.Posts.Get(ctx, postID); err != nil {
		return nil, normalizePostError(err)
	}
	limit, offset = normalizeListBounds(limit, offset)
	comments, err := s.store.Comments.GetTopLevelByPostID(ctx, postID, limit, offset)
	if err != nil {
		return nil, normalizeCommentError(err)
	}
	return s.mapComments(ctx, comments, profileID), nil
}

func (s *Service) GetCommentReplies(ctx context.Context, userAccountID, postID, commentID int64, limit, offset int) ([]Comment, error) {
	if userAccountID <= 0 || postID <= 0 || commentID <= 0 {
		return nil, ErrInvalidInput
	}
	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	parent, err := s.store.Comments.Get(ctx, commentID)
	if err != nil {
		return nil, normalizeCommentError(err)
	}
	if parent.PostID != postID {
		return nil, ErrCommentNotFound
	}
	limit, offset = normalizeListBounds(limit, offset)
	comments, err := s.store.Comments.GetReplies(ctx, postID, commentID, limit, offset)
	if err != nil {
		return nil, normalizeCommentError(err)
	}
	return s.mapComments(ctx, comments, profileID), nil
}

func (s *Service) GetCommentRepliesBatch(ctx context.Context, userAccountID, postID int64, parentCommentIDs []int64, limit, offset int) (map[int64][]Comment, error) {
	if userAccountID <= 0 || postID <= 0 || len(parentCommentIDs) == 0 || len(parentCommentIDs) > 50 {
		return nil, ErrInvalidInput
	}
	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	if _, err := s.store.Posts.Get(ctx, postID); err != nil {
		return nil, normalizePostError(err)
	}
	seen := make(map[int64]struct{}, len(parentCommentIDs))
	ids := make([]int64, 0, len(parentCommentIDs))
	for _, id := range parentCommentIDs {
		if id <= 0 {
			return nil, ErrInvalidInput
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	limit, offset = normalizeListBounds(limit, offset)
	grouped, err := s.store.Comments.GetRepliesByParentIDs(ctx, postID, ids, limit, offset)
	if err != nil {
		return nil, normalizeCommentError(err)
	}
	result := make(map[int64][]Comment, len(ids))
	for _, id := range ids {
		result[id] = s.mapComments(ctx, grouped[id], profileID)
	}
	return result, nil
}

func (s *Service) CreateComment(ctx context.Context, userAccountID, postID int64, text string, parentCommentID *int64) (*Comment, error) {
	text = strings.TrimSpace(text)
	if userAccountID <= 0 || postID <= 0 || text == "" {
		return nil, ErrInvalidInput
	}
	post, err := s.store.Posts.Get(ctx, postID)
	if err != nil {
		return nil, normalizePostError(err)
	}
	if !post.AllowComments {
		return nil, ErrCommentsDisabled
	}
	if parentCommentID != nil {
		if *parentCommentID <= 0 {
			return nil, ErrInvalidInput
		}
		parent, err := s.store.Comments.Get(ctx, *parentCommentID)
		if err != nil {
			return nil, normalizeCommentError(err)
		}
		if parent.PostID != postID {
			return nil, ErrCommentNotFound
		}
	}
	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	if post.CommunityID != nil {
		member, err := s.communityMember(ctx, *post.CommunityID, profileID)
		if err != nil || !member.IsActive {
			return nil, ErrForbidden
		}
	}
	comment := model.NewComment(&text, postID, parentCommentID, profileID)
	id, err := s.store.Comments.Save(ctx, *comment)
	if err != nil {
		return nil, err
	}
	s.emitEvent(profileID, postID, post.AuthorID, post.CommunityID, analytics.EventPostComment)
	saved, err := s.store.Comments.Get(ctx, id)
	if err != nil {
		return nil, normalizeCommentError(err)
	}
	mapped := s.mapComments(ctx, []model.Comment{*saved}, profileID)
	return &mapped[0], nil
}

func (s *Service) UpdateComment(ctx context.Context, userAccountID, postID, commentID int64, text string) (*Comment, error) {
	text = strings.TrimSpace(text)
	if userAccountID <= 0 || postID <= 0 || commentID <= 0 || text == "" {
		return nil, ErrInvalidInput
	}
	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	comment, err := s.store.Comments.Get(ctx, commentID)
	if err != nil {
		return nil, normalizeCommentError(err)
	}
	if comment.PostID != postID {
		return nil, ErrCommentNotFound
	}
	if comment.AuthorID != profileID {
		return nil, ErrForbidden
	}
	comment.Text = &text
	if err := s.store.Comments.Update(ctx, *comment); err != nil {
		return nil, normalizeCommentError(err)
	}
	updated, err := s.store.Comments.Get(ctx, commentID)
	if err != nil {
		return nil, normalizeCommentError(err)
	}
	mapped := s.mapComments(ctx, []model.Comment{*updated}, profileID)
	return &mapped[0], nil
}

func (s *Service) DeleteComment(ctx context.Context, userAccountID, postID, commentID int64) error {
	if userAccountID <= 0 || postID <= 0 || commentID <= 0 {
		return ErrInvalidInput
	}
	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return err
	}
	comment, err := s.store.Comments.Get(ctx, commentID)
	if err != nil {
		return normalizeCommentError(err)
	}
	if comment.PostID != postID {
		return ErrCommentNotFound
	}
	post, err := s.store.Posts.Get(ctx, postID)
	if err != nil {
		return normalizePostError(err)
	}
	if comment.AuthorID != profileID && !s.canDeletePost(ctx, *post, profileID) {
		return ErrForbidden
	}
	return normalizeCommentError(s.store.Comments.Delete(ctx, commentID))
}

func (s *Service) LikeComment(ctx context.Context, userAccountID, postID, commentID int64) (*Comment, error) {
	if userAccountID <= 0 || postID <= 0 || commentID <= 0 {
		return nil, ErrInvalidInput
	}
	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	if _, err := s.store.Posts.Get(ctx, postID); err != nil {
		return nil, normalizePostError(err)
	}
	comment, err := s.store.Comments.Get(ctx, commentID)
	if err != nil {
		return nil, normalizeCommentError(err)
	}
	if comment.PostID != postID {
		return nil, ErrCommentNotFound
	}
	existing, err := s.store.Likes.GetCommentLikeByAuthor(ctx, commentID, profileID)
	if err == nil {
		if !existing.IsActive {
			if err := s.store.Likes.SetActive(ctx, existing.ID, true); err != nil {
				return nil, err
			}
		}
	} else {
		if _, err := s.store.Likes.Save(ctx, *model.NewLikeToComment(commentID, profileID)); err != nil {
			return nil, err
		}
	}
	refreshed, err := s.store.Comments.Get(ctx, commentID)
	if err != nil {
		return nil, normalizeCommentError(err)
	}
	mapped := s.mapComments(ctx, []model.Comment{*refreshed}, profileID)
	return &mapped[0], nil
}

func (s *Service) UnlikeComment(ctx context.Context, userAccountID, postID, commentID int64) (*Comment, error) {
	if userAccountID <= 0 || postID <= 0 || commentID <= 0 {
		return nil, ErrInvalidInput
	}
	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	if _, err := s.store.Posts.Get(ctx, postID); err != nil {
		return nil, normalizePostError(err)
	}
	comment, err := s.store.Comments.Get(ctx, commentID)
	if err != nil {
		return nil, normalizeCommentError(err)
	}
	if comment.PostID != postID {
		return nil, ErrCommentNotFound
	}
	existing, err := s.store.Likes.GetCommentLikeByAuthor(ctx, commentID, profileID)
	if err == nil && existing.IsActive {
		if err := s.store.Likes.SetActive(ctx, existing.ID, false); err != nil {
			return nil, err
		}
	}
	refreshed, err := s.store.Comments.Get(ctx, commentID)
	if err != nil {
		return nil, normalizeCommentError(err)
	}
	mapped := s.mapComments(ctx, []model.Comment{*refreshed}, profileID)
	return &mapped[0], nil
}

func (s *Service) GetFeed(ctx context.Context, userAccountID int64, rawCursor, mode string, limit int) (FeedResult, error) {
	return s.getFeed(ctx, userAccountID, rawCursor, mode, limit, false)
}

func (s *Service) GetPublicFeed(ctx context.Context, rawCursor string, limit int) (FeedResult, error) {
	return s.getFeed(ctx, 0, rawCursor, "by-time", limit, true)
}

func (s *Service) RecordFeedEvents(ctx context.Context, userAccountID int64, events []analytics.PostEvent) {
	if s.analytics == nil || userAccountID <= 0 {
		return
	}
	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil || profileID <= 0 {
		return
	}

	postIDs := make([]int64, 0, len(events))
	for _, e := range events {
		postIDs = append(postIDs, e.PostID)
	}
	postMeta := make(map[int64]struct {
		authorID    int64
		communityID *int64
	})
	if posts, err := s.store.Posts.GetByIDs(ctx, postIDs); err == nil {
		for _, p := range posts {
			postMeta[p.ID] = struct {
				authorID    int64
				communityID *int64
			}{p.AuthorID, p.CommunityID}
		}
	}

	now := time.Now()
	for _, e := range events {
		e.ProfileID = profileID
		e.EventTime = now
		if meta, ok := postMeta[e.PostID]; ok {
			e.AuthorProfileID = meta.authorID
			e.CommunityID = meta.communityID
		}
		switch e.Type {
		case analytics.EventPostView, analytics.EventPostHide, analytics.EventPostReport:
			s.analytics.WriteEvent(e)
		}
	}
}

func (s *Service) getFeed(ctx context.Context, userAccountID int64, rawCursor, mode string, limit int, publicOnly bool) (FeedResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if mode == "for-you" && s.recommendation != nil && userAccountID > 0 {
		return s.getRecommendedFeed(ctx, userAccountID, rawCursor, limit)
	}
	return s.getChronologicalFeed(ctx, userAccountID, rawCursor, limit, publicOnly)
}

func (s *Service) getChronologicalFeed(ctx context.Context, userAccountID int64, rawCursor string, limit int, publicOnly bool) (FeedResult, error) {
	var friendIDs []int64
	var viewerProfileID int64

	if !publicOnly && userAccountID > 0 {
		resp, err := s.userClient.GetFriendProfileIDs(ctx, &userpb.GetFriendProfileIDsRequest{UserAccountId: userAccountID})
		if err == nil && resp != nil {
			friendIDs = resp.ProfileIds
		}
		viewerProfileID, _ = s.profileIDByUserAccount(ctx, userAccountID)
	}

	if !publicOnly && len(friendIDs) == 0 {
		return FeedResult{Posts: []FeedPost{}}, nil
	}

	var beforeTime *time.Time
	var beforeID *int64
	var legacyCursor *cursor.Cursor

	if rawCursor != "" {
		decoded, err := cursor.Decode(rawCursor)
		if err != nil {
			return FeedResult{}, ErrInvalidInput
		}
		legacyCursor = &decoded
	}

	if legacyCursor != nil {
		t := legacyCursor.CreatedAt
		beforeTime = &t
		// legacy cursor has uuid, not int64; use a very large ID so we get all posts at this timestamp
		id := int64(math.MaxInt64)
		beforeID = &id
	}

	fetchLimit := limit + 1
	var posts []model.Post
	var err error
	if publicOnly {
		posts, err = s.store.Posts.GetFeedPage(ctx, nil, beforeTime, beforeID, fetchLimit, true)
	} else {
		posts, err = s.store.Posts.GetFeedPage(ctx, friendIDs, beforeTime, beforeID, fetchLimit, false)
	}
	if err != nil {
		return FeedResult{}, err
	}

	hasMore := len(posts) > limit
	if hasMore {
		posts = posts[:limit]
	}

	var nextCursor string
	if hasMore && len(posts) > 0 {
		last := posts[len(posts)-1]
		nextCursor = cursor.Encode(cursor.Cursor{CreatedAt: last.CreatedAt, ID: last.Uid})
	}

	feedPosts := s.buildFeedBatch(ctx, posts, viewerProfileID)
	return FeedResult{Posts: feedPosts, Cursor: nextCursor, HasMore: hasMore}, nil
}

func (s *Service) getRecommendedFeed(ctx context.Context, userAccountID int64, rawCursor string, limit int) (FeedResult, error) {
	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return FeedResult{Posts: []FeedPost{}}, nil
	}

	var sessionID string
	offset := 0

	if rawCursor != "" {
		sc, decErr := decodeSessionCursor(rawCursor)
		if decErr == nil {
			sessionID = sc.SessionID
			offset = sc.Offset
		}
	}

	var candidateIDs []int64
	if sessionID != "" && s.sessionCache != nil {
		candidateIDs, _ = s.sessionCache.GetFeedSession(ctx, sessionID)
	}

	if len(candidateIDs) == 0 {
		candidateIDs, err = s.buildCandidates(ctx, profileID, userAccountID)
		if err != nil || len(candidateIDs) == 0 {
			return s.getChronologicalFeed(ctx, userAccountID, "", limit, false)
		}
		sessionID = uuid.New().String()
		offset = 0
		if s.sessionCache != nil {
			_ = s.sessionCache.SaveFeedSession(ctx, sessionID, candidateIDs, 10*time.Minute)
		}
	}

	if offset >= len(candidateIDs) {
		return FeedResult{Posts: []FeedPost{}, Cursor: "", HasMore: false}, nil
	}

	end := offset + limit
	if end > len(candidateIDs) {
		end = len(candidateIDs)
	}
	pageIDs := candidateIDs[offset:end]
	hasMore := end < len(candidateIDs)

	posts, err := s.store.Posts.GetByIDs(ctx, pageIDs)
	if err != nil {
		return FeedResult{}, err
	}

	postMap := make(map[int64]model.Post, len(posts))
	for _, p := range posts {
		postMap[p.ID] = p
	}
	orderedPosts := make([]model.Post, 0, len(pageIDs))
	for _, id := range pageIDs {
		if p, ok := postMap[id]; ok {
			orderedPosts = append(orderedPosts, p)
		}
	}

	var nextCursor string
	if hasMore {
		nextCursor = encodeSessionCursor(sessionCursor{SessionID: sessionID, Offset: end})
	}

	feedPosts := s.buildFeedBatch(ctx, orderedPosts, profileID)
	return FeedResult{Posts: feedPosts, Cursor: nextCursor, HasMore: hasMore}, nil
}

func (s *Service) buildCandidates(ctx context.Context, profileID, userAccountID int64) ([]int64, error) {
	resp, err := s.userClient.GetFriendProfileIDs(ctx, &userpb.GetFriendProfileIDsRequest{UserAccountId: userAccountID})
	var friendIDs []int64
	if err == nil && resp != nil {
		friendIDs = resp.ProfileIds
	}

	candidateIDs, err := s.recommendation.GetForYouCandidates(ctx, repository.ForYouInput{
		ProfileID:       profileID,
		FriendIDs:       friendIDs,
		Limit:           200,
		ExcludeAuthorID: profileID,
	})
	if err != nil {
		return nil, err
	}
	if len(candidateIDs) == 0 {
		return nil, nil
	}

	stats, _ := s.recommendation.GetPostStats(ctx, candidateIDs)
	authorAffinity, _ := s.recommendation.GetAuthorAffinity(ctx, profileID, 30)
	communityAffinity, _ := s.recommendation.GetCommunityAffinity(ctx, profileID, 30)

	posts, _ := s.store.Posts.GetByIDs(ctx, candidateIDs)
	postMeta := make(map[int64]struct {
		authorID    int64
		communityID int64
	}, len(posts))
	for _, p := range posts {
		var cid int64
		if p.CommunityID != nil {
			cid = *p.CommunityID
		}
		postMeta[p.ID] = struct {
			authorID    int64
			communityID int64
		}{p.AuthorID, cid}
	}

	friendSet := make(map[int64]struct{}, len(friendIDs))
	for _, id := range friendIDs {
		friendSet[id] = struct{}{}
	}

	memberCommunityIDs, _ := s.store.Memberships.GetMemberCommunityIDs(ctx, profileID)
	memberCommunitySet := make(map[int64]struct{}, len(memberCommunityIDs))
	for _, cid := range memberCommunityIDs {
		memberCommunitySet[cid] = struct{}{}
	}

	type scoredPost struct {
		id    int64
		score float64
	}
	scored := make([]scoredPost, 0, len(candidateIDs))
	for _, id := range candidateIDs {
		meta := postMeta[id]
		_, isFriend := friendSet[meta.authorID]
		_, isMemberCommunity := memberCommunitySet[meta.communityID]
		if meta.communityID == 0 {
			isMemberCommunity = false
		}
		scored = append(scored, scoredPost{
			id: id,
			score: repository.ScorePost(
				stats[id],
				authorAffinity[meta.authorID],
				communityAffinity[meta.communityID],
				isFriend,
				isMemberCommunity,
			),
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	result := make([]int64, 0, len(scored))
	for _, sp := range scored {
		result = append(result, sp.id)
	}
	return result, nil
}

func (s *Service) buildFeedBatch(ctx context.Context, posts []model.Post, viewerProfileID int64) []FeedPost {
	if len(posts) == 0 {
		return []FeedPost{}
	}
	ids := make([]int64, 0, len(posts))
	for _, p := range posts {
		ids = append(ids, p.ID)
	}

	type batchData struct {
		likeCounts    map[int64]int
		isLiked       map[int64]bool
		commentCounts map[int64]int
		repostCounts  map[int64]int
		mediaByPost   map[int64][]model.AttachedMedia
	}

	var bd batchData
	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		bd.likeCounts, _ = s.store.Likes.GetPostLikeCountsBatch(egCtx, ids)
		return nil
	})
	eg.Go(func() error {
		bd.isLiked, _ = s.store.Likes.GetViewerPostLikesBatch(egCtx, ids, viewerProfileID)
		return nil
	})
	eg.Go(func() error {
		bd.commentCounts, _ = s.store.Comments.GetCommentCountsBatch(egCtx, ids)
		return nil
	})
	eg.Go(func() error {
		bd.repostCounts, _ = s.store.Reposts.GetRepostCountsBatch(egCtx, ids)
		return nil
	})
	eg.Go(func() error {
		bd.mediaByPost, _ = s.store.PostMedia.GetDetailedMediaByPostIDs(egCtx, ids)
		return nil
	})
	_ = eg.Wait()

	if bd.likeCounts == nil {
		bd.likeCounts = map[int64]int{}
	}
	if bd.isLiked == nil {
		bd.isLiked = map[int64]bool{}
	}
	if bd.commentCounts == nil {
		bd.commentCounts = map[int64]int{}
	}
	if bd.repostCounts == nil {
		bd.repostCounts = map[int64]int{}
	}
	if bd.mediaByPost == nil {
		bd.mediaByPost = map[int64][]model.AttachedMedia{}
	}

	result := make([]FeedPost, 0, len(posts))
	for _, post := range posts {
		author, err := s.author(ctx, post.AuthorID)
		if err != nil {
			continue
		}
		text := ""
		if post.Text != nil {
			text = html.EscapeString(*post.Text)
		}
		attached := make([]Media, 0, len(bd.mediaByPost[post.ID]))
		for _, item := range bd.mediaByPost[post.ID] {
			attached = append(attached, Media{
				ID:       item.MediaID,
				UID:      item.UID.String(),
				Name:     item.Name,
				MimeType: item.MimeType,
				URL:      s.absoluteMediaURL(ctx, item.MediaID, item.Link),
			})
		}
		media, files := splitMedia(attached)

		likeCount := bd.likeCounts[post.ID]
		if s.cache != nil {
			if cached, err := s.cache.GetPostLikeCount(ctx, post.ID); err == nil {
				likeCount = cached
			}
		}

		result = append(result, FeedPost{
			ID:        post.ID,
			Text:      text,
			Author:    author,
			CreatedAt: post.CreatedAt,
			Likes:     likeCount,
			IsLiked:   bd.isLiked[post.ID],
			Comments:  bd.commentCounts[post.ID],
			Reposts:   bd.repostCounts[post.ID],
			Medias:    media,
			Files:     files,
		})
	}
	return result
}

type sessionCursor struct {
	SessionID string `json:"s"`
	Offset    int    `json:"o"`
}

func encodeSessionCursor(sc sessionCursor) string {
	b, _ := json.Marshal(sc)
	return "v2s:" + base64.StdEncoding.EncodeToString(b)
}

func decodeSessionCursor(raw string) (sessionCursor, error) {
	if !strings.HasPrefix(raw, "v2s:") {
		return sessionCursor{}, errors.New("not a session cursor")
	}
	b, err := base64.StdEncoding.DecodeString(raw[4:])
	if err != nil {
		return sessionCursor{}, err
	}
	var sc sessionCursor
	return sc, json.Unmarshal(b, &sc)
}


func (s *Service) buildFeedPost(ctx context.Context, post model.Post, viewerProfileID int64) (FeedPost, error) {
	author, err := s.author(ctx, post.AuthorID)
	if err != nil {
		return FeedPost{}, err
	}

	text := ""
	if post.Text != nil {
		text = html.EscapeString(*post.Text)
	}

	media, files := splitMedia(s.postMedia(ctx, post.ID))
	return FeedPost{
		ID:        post.ID,
		Text:      text,
		Author:    author,
		CreatedAt: post.CreatedAt,
		Likes:     s.postLikeCount(ctx, post.ID),
		IsLiked:   viewerProfileID > 0 && s.store.Likes.HasActivePostLike(ctx, post.ID, viewerProfileID),
		Comments:  s.store.Comments.GetCommentCount(ctx, post.ID),
		Reposts:   s.store.Reposts.GetRepostCount(ctx, post.ID),
		Medias:    media,
		Files:     files,
	}, nil
}

func (s *Service) buildPostDetails(ctx context.Context, post model.Post, viewerProfileID int64) (*PostDetails, error) {
	author, err := s.author(ctx, post.AuthorID)
	if err != nil {
		return nil, err
	}
	media, files := splitMedia(s.postMedia(ctx, post.ID))
	return &PostDetails{
		ID:          post.ID,
		UID:         post.Uid,
		AuthorID:    post.AuthorID,
		CommunityID: post.CommunityID,
		Text:        post.Text,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
		Author:      author,
		Media:       media,
		Files:       files,
		Likes:       s.postLikeCount(ctx, post.ID),
		Comments:    s.store.Comments.GetCommentCount(ctx, post.ID),
		IsLiked:     viewerProfileID > 0 && s.store.Likes.HasActivePostLike(ctx, post.ID, viewerProfileID),
	}, nil
}

func (s *Service) postLikeCount(ctx context.Context, postID int64) int {
	if s.cache != nil {
		if count, err := s.cache.GetPostLikeCount(ctx, postID); err == nil {
			return count
		}
	}
	count := s.store.Likes.GetLikeCountOnPost(ctx, postID)
	if s.cache != nil {
		_ = s.cache.SetPostLikeCount(ctx, postID, count)
	}
	return count
}

func (s *Service) refreshPostLikeCount(ctx context.Context, postID int64) {
	if s.cache == nil {
		return
	}
	_ = s.cache.SetPostLikeCount(ctx, postID, s.store.Likes.GetLikeCountOnPost(ctx, postID))
}

func (s *Service) deletePostLikeCount(ctx context.Context, postID int64) {
	if s.cache == nil {
		return
	}
	_ = s.cache.DeletePostLikeCount(ctx, postID)
}

func (s *Service) emitSnapshot(post model.Post, hasMedia bool) {
	if s.analytics == nil {
		return
	}
	s.analytics.WriteSnapshot(analytics.PostSnapshot{
		PostID:          post.ID,
		AuthorProfileID: post.AuthorID,
		CommunityID:     post.CommunityID,
		IsPublicDemo:    post.IsPublicDemo,
		IsActive:        post.IsActive,
		AllowComments:   post.AllowComments,
		CreatedAt:       post.CreatedAt,
		UpdatedAt:       post.UpdatedAt,
		TextLength:      uint16(len(derefStr(post.Text))),
		HasMedia:        hasMedia,
	})
}

func (s *Service) emitEvent(profileID, postID, authorProfileID int64, communityID *int64, t analytics.EventType) {
	if s.analytics == nil {
		return
	}
	s.analytics.WriteEvent(analytics.PostEvent{
		EventTime:       time.Now(),
		ProfileID:       profileID,
		PostID:          postID,
		AuthorProfileID: authorProfileID,
		CommunityID:     communityID,
		Type:            t,
	})
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (s *Service) mapComments(ctx context.Context, comments []model.Comment, viewerProfileID int64) []Comment {
	if len(comments) == 0 {
		return []Comment{}
	}
	commentIDs := make([]int64, 0, len(comments))
	for _, c := range comments {
		commentIDs = append(commentIDs, c.ID)
	}
	likeCounts, _ := s.store.Likes.GetCommentLikeCountBatch(ctx, commentIDs)
	viewerLikes, _ := s.store.Likes.GetCommentViewerLikesBatch(ctx, commentIDs, viewerProfileID)
	result := make([]Comment, 0, len(comments))
	for _, comment := range comments {
		result = append(result, s.mapComment(ctx, comment, likeCounts[comment.ID], viewerLikes[comment.ID]))
	}
	return result
}

func (s *Service) mapComment(ctx context.Context, comment model.Comment, likes int, isLiked bool) Comment {
	text := comment.Text
	if text != nil {
		escaped := html.EscapeString(html.UnescapeString(*text))
		text = &escaped
	}
	author, err := s.author(ctx, comment.AuthorID)
	if err != nil {
		author = Author{ID: comment.AuthorID}
	}
	return Comment{
		ID:              comment.ID,
		UID:             comment.Uid,
		Text:            text,
		PostID:          comment.PostID,
		ParentCommentID: comment.ParentCommentID,
		Author:          author,
		CreatedAt:       comment.CreatedAt,
		UpdatedAt:       comment.UpdatedAt,
		RepliesCount:    comment.RepliesCount,
		Likes:           likes,
		IsLiked:         isLiked,
	}
}

func (s *Service) author(ctx context.Context, profileID int64) (Author, error) {
	if s.userClient == nil {
		return Author{ID: profileID}, nil
	}

	resp, err := s.userClient.GetProfileSummary(ctx, &userpb.GetProfileSummaryRequest{ProfileId: profileID})
	if err != nil {
		community, communityErr := s.communityByProfileID(ctx, profileID)
		if communityErr == nil {
			author := Author{ID: profileID, FirstName: community.Title, Username: community.Username}
			if community.AvatarID != nil {
				author.AvatarURL = s.mediaURL(ctx, *community.AvatarID)
			}
			return author, nil
		}
		return Author{}, err
	}

	author := Author{
		ID:            resp.GetProfileId(),
		FirstName:     resp.GetFirstName(),
		LastName:      resp.GetLastName(),
		Username:      resp.GetUsername(),
		UserAccountID: resp.GetUserAccountId(),
	}
	if resp.AvatarId != nil {
		author.AvatarURL = s.mediaURL(ctx, resp.GetAvatarId())
	}
	return author, nil
}

func (s *Service) postMedia(ctx context.Context, postID int64) []Media {
	attached, err := s.store.PostMedia.GetDetailedMediaByPostID(ctx, postID)
	if err != nil {
		return nil
	}
	items := make([]Media, 0, len(attached))
	for _, item := range attached {
		items = append(items, Media{
			ID:       item.MediaID,
			UID:      item.UID.String(),
			Name:     item.Name,
			MimeType: item.MimeType,
			URL:      s.absoluteMediaURL(ctx, item.MediaID, item.Link),
		})
	}
	return items
}

func (s *Service) media(ctx context.Context, mediaID int64) *Media {
	if s.mediaClient == nil || mediaID <= 0 {
		return nil
	}
	resp, err := s.mediaClient.GetMedia(ctx, &mediapb.GetMediaRequest{MediaId: mediaID})
	if err != nil || resp == nil {
		return nil
	}

	url := resp.GetUrl()
	if !strings.HasPrefix(url, "http") {
		mediaURL, err := s.mediaClient.GetMediaURL(ctx, &mediapb.GetMediaURLRequest{MediaId: resp.GetMediaId()})
		if err != nil || mediaURL == nil {
			return nil
		}
		url = mediaURL.GetUrl()
	}

	return &Media{ID: resp.GetMediaId(), UID: resp.GetUid(), MimeType: resp.GetMimeType(), URL: url}
}

func (s *Service) absoluteMediaURL(ctx context.Context, mediaID int64, raw string) string {
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") || s.mediaClient == nil {
		return raw
	}
	resp, err := s.mediaClient.GetMediaURL(ctx, &mediapb.GetMediaURLRequest{MediaId: mediaID})
	if err != nil || resp == nil || strings.TrimSpace(resp.GetUrl()) == "" {
		return raw
	}
	return resp.GetUrl()
}

func (s *Service) mediaURL(ctx context.Context, mediaID int64) *string {
	if s.mediaClient == nil || mediaID <= 0 {
		return nil
	}
	resp, err := s.mediaClient.GetMediaURL(ctx, &mediapb.GetMediaURLRequest{MediaId: mediaID})
	if err != nil || resp == nil || strings.TrimSpace(resp.GetUrl()) == "" {
		return nil
	}
	url := resp.GetUrl()
	return &url
}

func (s *Service) profileIDByUserAccount(ctx context.Context, userAccountID int64) (int64, error) {
	if s.userClient == nil {
		return 0, ErrProfileNotFound
	}
	resp, err := s.userClient.GetProfileByUserAccount(ctx, &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return 0, ErrProfileNotFound
		}
		return 0, err
	}
	if resp.GetProfileId() <= 0 {
		return 0, ErrProfileNotFound
	}
	return resp.GetProfileId(), nil
}

func (s *Service) resolvePostTarget(ctx context.Context, actorProfileID int64, requestedAuthorID, requestedCommunityID *int64) (int64, *int64, error) {
	if requestedCommunityID != nil {
		if *requestedCommunityID <= 0 {
			return 0, nil, ErrInvalidInput
		}
		community, err := s.communityByID(ctx, *requestedCommunityID)
		if err != nil {
			return 0, nil, ErrForbidden
		}
		if requestedAuthorID == nil || *requestedAuthorID == actorProfileID {
			if !s.canPostAsMember(ctx, community.ID, actorProfileID) {
				return 0, nil, ErrForbidden
			}
			return actorProfileID, &community.ID, nil
		}
		if *requestedAuthorID == community.ProfileID && s.canPostAsCommunity(ctx, community.ProfileID, actorProfileID) {
			return community.ProfileID, &community.ID, nil
		}
		return 0, nil, ErrForbidden
	}
	if requestedAuthorID == nil || *requestedAuthorID == actorProfileID {
		return actorProfileID, nil, nil
	}
	if *requestedAuthorID <= 0 {
		return 0, nil, ErrInvalidInput
	}
	community, err := s.communityByProfileID(ctx, *requestedAuthorID)
	if err == nil && s.canPostAsCommunity(ctx, *requestedAuthorID, actorProfileID) {
		return *requestedAuthorID, &community.ID, nil
	}
	return 0, nil, ErrForbidden
}

func (s *Service) canEditPost(ctx context.Context, post model.Post, actorProfileID int64) bool {
	if post.AuthorID == actorProfileID {
		return true
	}
	if post.CommunityID == nil {
		return false
	}
	community, err := s.communityByID(ctx, *post.CommunityID)
	if err != nil || community.ProfileID != post.AuthorID {
		return false
	}
	member, err := s.communityMember(ctx, community.ID, actorProfileID)
	if err != nil || member == nil || !member.IsActive {
		return false
	}
	role := normalizeCommunityRole(member.Role)
	return role == "owner" || role == "admin" || role == "moderator"
}

func (s *Service) canDeletePost(ctx context.Context, post model.Post, actorProfileID int64) bool {
	if post.AuthorID == actorProfileID {
		return true
	}
	if post.CommunityID == nil {
		return false
	}
	community, err := s.communityByID(ctx, *post.CommunityID)
	if err != nil {
		return false
	}
	member, err := s.communityMember(ctx, community.ID, actorProfileID)
	if err != nil || member == nil || !member.IsActive {
		return false
	}
	role := normalizeCommunityRole(member.Role)
	return role == "owner" || role == "admin" || role == "moderator"
}

func (s *Service) canPostAsCommunity(ctx context.Context, communityProfileID, actorProfileID int64) bool {
	if s.communityClient == nil {
		return false
	}
	resp, err := s.communityClient.CanPostByProfile(ctx, &communitypb.CanPostByProfileRequest{
		CommunityProfileId: communityProfileID,
		ActorProfileId:     actorProfileID,
	})
	if err != nil || resp == nil {
		return false
	}
	return resp.GetOk()
}

func (s *Service) canPostAsMember(ctx context.Context, communityID, actorProfileID int64) bool {
	if s.communityClient == nil {
		return false
	}
	resp, err := s.communityClient.CanPostAsMember(ctx, &communitypb.CanPostAsMemberRequest{
		CommunityId:    communityID,
		ActorProfileId: actorProfileID,
	})
	if err != nil || resp == nil {
		return false
	}
	return resp.GetOk()
}

func (s *Service) communityByID(ctx context.Context, communityID int64) (*communityInfo, error) {
	if s.communityClient == nil {
		return nil, ErrCommunityNotFound
	}
	resp, err := s.communityClient.GetCommunity(ctx, &communitypb.GetCommunityRequest{CommunityId: communityID})
	if err != nil {
		return nil, normalizeCommunityError(err)
	}
	return communityFromResponse(resp)
}

func (s *Service) communityByProfileID(ctx context.Context, profileID int64) (*communityInfo, error) {
	if s.communityClient == nil {
		return nil, ErrCommunityNotFound
	}
	resp, err := s.communityClient.GetCommunityByProfile(ctx, &communitypb.GetCommunityByProfileRequest{ProfileId: profileID})
	if err != nil {
		return nil, normalizeCommunityError(err)
	}
	return communityFromResponse(resp)
}

func (s *Service) communityMember(ctx context.Context, communityID, profileID int64) (*communityMemberInfo, error) {
	if s.communityClient == nil {
		return nil, ErrCommunityNotFound
	}
	resp, err := s.communityClient.GetMember(ctx, &communitypb.GetMemberRequest{CommunityId: communityID, ProfileId: profileID})
	if err != nil {
		return nil, normalizeCommunityError(err)
	}
	return &communityMemberInfo{Role: resp.GetRole(), IsActive: resp.GetIsActive()}, nil
}

func communityFromResponse(resp *communitypb.CommunityResponse) (*communityInfo, error) {
	if resp == nil || resp.GetCommunityId() <= 0 || resp.GetProfileId() <= 0 {
		return nil, ErrCommunityNotFound
	}
	return &communityInfo{
		ID:        resp.GetCommunityId(),
		ProfileID: resp.GetProfileId(),
		Title:     resp.GetTitle(),
		Username:  resp.GetUsername(),
		AvatarID:  resp.AvatarId,
	}, nil
}

func normalizeCommunityError(err error) error {
	if err == nil {
		return nil
	}
	switch status.Code(err) {
	case codes.NotFound:
		return ErrCommunityNotFound
	case codes.InvalidArgument:
		return ErrInvalidInput
	default:
		return err
	}
}

func normalizeCommunityRole(role string) string {
	if role == "manager" {
		return "moderator"
	}
	return role
}

func (s *Service) attachMedia(ctx context.Context, postID, actorProfileID int64, medias []MediaRequestData) error {
	if len(medias) > maxAttachments {
		return ErrInvalidInput
	}
	seen := make(map[int64]struct{}, len(medias))
	for i, media := range medias {
		if media.MediaID <= 0 {
			return ErrInvalidInput
		}
		if _, ok := seen[media.MediaID]; ok {
			return ErrInvalidInput
		}
		seen[media.MediaID] = struct{}{}
		authorID, err := s.store.PostMedia.GetMediaAuthorID(ctx, media.MediaID)
		if err != nil {
			return ErrMediaAttachmentError
		}
		if authorID != actorProfileID {
			return ErrForbidden
		}
		if err := s.store.PostMedia.Save(ctx, *model.NewPostWithMedia(postID, media.MediaID, i)); err != nil {
			return ErrMediaAttachmentError
		}
	}
	return nil
}

func appendAttachmentRefs(media []MediaRequestData, files []MediaRequestData) []MediaRequestData {
	if len(files) == 0 {
		return media
	}
	result := make([]MediaRequestData, 0, len(media)+len(files))
	result = append(result, media...)
	result = append(result, files...)
	return result
}

func splitMedia(items []Media) ([]Media, []Media) {
	media := make([]Media, 0, len(items))
	files := make([]Media, 0)
	for _, item := range items {
		if isTimelineMedia(item.MimeType) {
			media = append(media, item)
			continue
		}
		files = append(files, item)
	}
	return media, files
}

func isTimelineMedia(mimeType string) bool {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	return mimeType == "image" ||
		mimeType == "video" ||
		strings.HasPrefix(mimeType, "image/") ||
		strings.HasPrefix(mimeType, "video/")
}

func normalizeListBounds(limit, offset int) (int, int) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func normalizePostError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrPostNotFound) {
		return ErrPostNotFound
	}
	return err
}

func normalizeCommentError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrCommentNotFound) {
		return ErrCommentNotFound
	}
	return err
}

func ToStatus(err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrPostContentRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, ErrPostNotFound), errors.Is(err, ErrCommentNotFound), errors.Is(err, ErrProfileNotFound), errors.Is(err, ErrCommunityNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

func NewPost(text *string, authorID int64, isPublicDemo, allowComments bool) *model.Post {
	return model.NewPost(text, authorID, isPublicDemo, allowComments)
}

func Cursor(createdAt time.Time, id uuid.UUID) string {
	return cursor.Encode(cursor.Cursor{CreatedAt: createdAt, ID: id})
}
