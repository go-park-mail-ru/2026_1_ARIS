package service

import (
	"context"
	"errors"
	"html"
	"sort"
	"strings"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/post/repository"
	legacycommunity "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/community"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/cursor"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidInput         = errors.New("invalid input")
	ErrPostContentRequired  = errors.New(xerrors.PostContentRequired)
	ErrPostNotFound         = xerrors.PostNotFound
	ErrProfileNotFound      = xerrors.ProfileNotFound
	ErrCommunityNotFound    = legacycommunity.ErrCommunityNotFound
	ErrForbidden            = errors.New("denied")
	ErrMediaAttachmentError = errors.New("can't attach media")
)

type Service struct {
	store       repository.Store
	userClient  userpb.UserServiceClient
	mediaClient mediapb.MediaServiceClient
}

type CreateInput struct {
	Text            *string
	Media           []dto.MediaRequestData
	AuthorProfileID *int64
	CommunityID     *int64
}

type UpdateInput = CreateInput

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
	MimeType string
	URL      string
}

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
	Likes       int
	IsLiked     bool
}

type FeedPost struct {
	ID        uuid.UUID
	Text      string
	Author    Author
	CreatedAt time.Time
	Likes     int
	Comments  int
	Reposts   int
	Medias    []Media
}

type FeedResult struct {
	Posts   []FeedPost
	Cursor  string
	HasMore bool
}

func New(store repository.Store, userClient userpb.UserServiceClient, mediaClient mediapb.MediaServiceClient) *Service {
	return &Service{store: store, userClient: userClient, mediaClient: mediaClient}
}

func (s *Service) CreatePost(ctx context.Context, userAccountID int64, input CreateInput) (*PostDetails, error) {
	if userAccountID <= 0 {
		return nil, ErrInvalidInput
	}
	if input.Text == nil && len(input.Media) == 0 {
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

	post := models.NewPost(input.Text, authorID, false, true)
	post.CommunityID = communityID
	postID, err := s.store.Posts.Save(ctx, *post)
	if err != nil {
		return nil, err
	}

	if err := s.attachMedia(ctx, postID, input.Media); err != nil {
		return nil, err
	}

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

	return s.GetProfilePosts(ctx, profileID)
}

func (s *Service) GetProfilePosts(ctx context.Context, profileID int64) ([]PostDetails, error) {
	if profileID <= 0 {
		return nil, ErrInvalidInput
	}

	posts, err := s.store.Posts.GetByAuthorID(ctx, profileID)
	if err != nil {
		return nil, err
	}

	result := make([]PostDetails, 0, len(posts))
	for _, post := range posts {
		details, err := s.buildPostDetails(ctx, post, 0)
		if err == nil {
			result = append(result, *details)
		}
	}
	return result, nil
}

func (s *Service) GetCommunityPosts(ctx context.Context, communityID int64, viewerUserAccountID int64) ([]PostDetails, error) {
	if communityID <= 0 {
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

func (s *Service) GetCommunityOfficialPosts(ctx context.Context, communityID int64) ([]PostDetails, error) {
	if communityID <= 0 {
		return nil, ErrInvalidInput
	}
	community, err := s.store.Communities.Get(ctx, communityID)
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
		details, err := s.buildPostDetails(ctx, post, 0)
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
	if input.Text == nil && len(input.Media) == 0 {
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

	if input.Media != nil {
		if err := s.store.PostWithMedia.DeleteByPostID(ctx, postID); err != nil {
			return nil, err
		}
		if err := s.attachMedia(ctx, postID, input.Media); err != nil {
			return nil, err
		}
	}

	return s.GetPostForViewer(ctx, postID, userAccountID)
}

func (s *Service) DeletePost(ctx context.Context, userAccountID int64, postID int64) error {
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

	return normalizePostError(s.store.Posts.Delete(ctx, postID))
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
	existing, err := s.store.Likes.GetPostLikeByAuthor(ctx, postID, profileID)
	if err == nil {
		if !existing.IsActive {
			if err := s.store.Likes.SetActive(ctx, existing.ID, true); err != nil {
				return nil, err
			}
		}
		return s.GetPostForViewer(ctx, postID, userAccountID)
	}
	if _, err := s.store.Likes.Save(ctx, *models.NewLikeToPost(postID, profileID)); err != nil {
		return nil, err
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
	existing, err := s.store.Likes.GetPostLikeByAuthor(ctx, postID, profileID)
	if err == nil && existing.IsActive {
		if err := s.store.Likes.SetActive(ctx, existing.ID, false); err != nil {
			return nil, err
		}
	}
	return s.GetPostForViewer(ctx, postID, userAccountID)
}

func (s *Service) GetFeed(ctx context.Context, rawCursor string, limit int) (FeedResult, error) {
	return s.getFeed(ctx, rawCursor, limit, false)
}

func (s *Service) GetPublicFeed(ctx context.Context, rawCursor string, limit int) (FeedResult, error) {
	return s.getFeed(ctx, rawCursor, limit, true)
}

func (s *Service) getFeed(ctx context.Context, rawCursor string, limit int, publicOnly bool) (FeedResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var cur *cursor.Cursor
	if rawCursor != "" {
		decoded, err := cursor.Decode(rawCursor)
		if err != nil {
			return FeedResult{}, ErrInvalidInput
		}
		cur = &decoded
	}

	allPosts, err := s.store.Posts.GetAll(ctx)
	if err != nil {
		return FeedResult{}, err
	}

	posts := make([]models.Post, 0, len(allPosts))
	for _, post := range allPosts {
		if publicOnly != post.IsPublicDemo {
			continue
		}
		posts = append(posts, post)
	}

	sort.Slice(posts, func(i, j int) bool {
		if posts[i].CreatedAt.Equal(posts[j].CreatedAt) {
			return posts[i].ID > posts[j].ID
		}
		return posts[i].CreatedAt.After(posts[j].CreatedAt)
	})

	start := 0
	if cur != nil {
		start = len(posts)
		for i, post := range posts {
			if post.CreatedAt.Before(cur.CreatedAt) || (post.CreatedAt.Equal(cur.CreatedAt) && post.Uid.String() != cur.ID.String()) {
				start = i
				break
			}
		}
	}

	if start >= len(posts) {
		return FeedResult{Posts: []FeedPost{}}, nil
	}

	end := start + limit + 1
	if end > len(posts) {
		end = len(posts)
	}
	page := posts[start:end]
	hasMore := len(page) > limit
	if hasMore {
		page = page[:limit]
	}

	var nextCursor string
	if hasMore && len(page) > 0 {
		last := page[len(page)-1]
		nextCursor = cursor.Encode(cursor.Cursor{CreatedAt: last.CreatedAt, ID: last.Uid})
	}

	result := make([]FeedPost, 0, len(page))
	for _, post := range page {
		item, err := s.buildFeedPost(ctx, post)
		if err == nil {
			result = append(result, item)
		}
	}

	return FeedResult{Posts: result, Cursor: nextCursor, HasMore: hasMore}, nil
}

func (s *Service) buildPostDetails(ctx context.Context, post models.Post, viewerProfileID int64) (*PostDetails, error) {
	author, err := s.author(ctx, post.AuthorID)
	if err != nil {
		return nil, err
	}

	return &PostDetails{
		ID:          post.ID,
		UID:         post.Uid,
		AuthorID:    post.AuthorID,
		CommunityID: post.CommunityID,
		Text:        post.Text,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
		Author:      author,
		Media:       s.postMedia(ctx, post.ID),
		Likes:       s.store.Likes.GetLikeCountOnPost(ctx, post.ID),
		IsLiked:     viewerProfileID > 0 && s.store.Likes.HasActivePostLike(ctx, post.ID, viewerProfileID),
	}, nil
}

func (s *Service) buildFeedPost(ctx context.Context, post models.Post) (FeedPost, error) {
	author, err := s.author(ctx, post.AuthorID)
	if err != nil {
		return FeedPost{}, err
	}

	text := ""
	if post.Text != nil {
		text = html.EscapeString(*post.Text)
	}

	return FeedPost{
		ID:        post.Uid,
		Text:      text,
		Author:    author,
		CreatedAt: post.CreatedAt,
		Likes:     s.store.Likes.GetLikeCountOnPost(ctx, post.ID),
		Comments:  s.store.Comments.GetCommentCount(ctx, post.ID),
		Reposts:   s.store.Reposts.GetRepostCount(ctx, post.ID),
		Medias:    s.postMedia(ctx, post.ID),
	}, nil
}

func (s *Service) author(ctx context.Context, profileID int64) (Author, error) {
	if s.userClient == nil {
		return Author{ID: profileID}, nil
	}

	resp, err := s.userClient.GetProfileSummary(ctx, &userpb.GetProfileSummaryRequest{ProfileId: profileID})
	if err != nil {
		if s.store.Communities != nil {
			community, communityErr := s.store.Communities.GetByProfileID(ctx, profileID)
			if communityErr == nil {
				author := Author{ID: profileID, FirstName: community.Title, Username: community.Username}
				avatarID, avatarErr := s.store.Communities.GetAvatarID(ctx, profileID)
				if avatarErr == nil && avatarID != nil {
					author.AvatarURL = s.mediaURL(ctx, *avatarID)
				}
				return author, nil
			}
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

func (s *Service) resolvePostTarget(ctx context.Context, actorProfileID int64, requestedAuthorID, requestedCommunityID *int64) (int64, *int64, error) {
	if requestedCommunityID != nil {
		if *requestedCommunityID <= 0 || s.store.Communities == nil {
			return 0, nil, ErrInvalidInput
		}
		community, err := s.store.Communities.Get(ctx, *requestedCommunityID)
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
	if s.store.Communities != nil {
		community, err := s.store.Communities.GetByProfileID(ctx, *requestedAuthorID)
		if err == nil && s.canPostAsCommunity(ctx, *requestedAuthorID, actorProfileID) {
			return *requestedAuthorID, &community.ID, nil
		}
	}
	return 0, nil, ErrForbidden
}

func (s *Service) canEditPost(ctx context.Context, post models.Post, actorProfileID int64) bool {
	if post.AuthorID == actorProfileID {
		return true
	}
	if post.CommunityID == nil || s.store.Communities == nil {
		return false
	}
	community, err := s.store.Communities.Get(ctx, *post.CommunityID)
	if err != nil || community.ProfileID != post.AuthorID {
		return false
	}
	member, err := s.store.Communities.GetMember(ctx, community.ID, actorProfileID)
	if err != nil || member == nil || !member.IsActive {
		return false
	}
	role := normalizeCommunityRole(member.Role)
	return role == models.Owner
}

func (s *Service) canDeletePost(ctx context.Context, post models.Post, actorProfileID int64) bool {
	if post.AuthorID == actorProfileID {
		return true
	}
	if post.CommunityID == nil || s.store.Communities == nil {
		return false
	}
	community, err := s.store.Communities.Get(ctx, *post.CommunityID)
	if err != nil {
		return false
	}
	member, err := s.store.Communities.GetMember(ctx, community.ID, actorProfileID)
	if err != nil || member == nil || !member.IsActive {
		return false
	}
	role := normalizeCommunityRole(member.Role)
	return role == models.Owner || role == models.Admin || role == models.Moderator
}

func (s *Service) canPostAsCommunity(ctx context.Context, communityProfileID, actorProfileID int64) bool {
	if s.store.Communities == nil {
		return false
	}
	community, err := s.store.Communities.GetByProfileID(ctx, communityProfileID)
	if err != nil {
		return false
	}
	member, err := s.store.Communities.GetMember(ctx, community.ID, actorProfileID)
	if err != nil || member == nil || !member.IsActive {
		return false
	}
	role := normalizeCommunityRole(member.Role)
	return role == models.Owner || role == models.Admin || role == models.Moderator
}

func (s *Service) canPostAsMember(ctx context.Context, communityID, actorProfileID int64) bool {
	if s.store.Communities == nil {
		return false
	}
	member, err := s.store.Communities.GetMember(ctx, communityID, actorProfileID)
	if err != nil || member == nil || !member.IsActive {
		return false
	}
	role := normalizeCommunityRole(member.Role)
	return role == models.Owner || role == models.Admin || role == models.Moderator || role == models.Member
}

func normalizeCommunityRole(role models.CommunityMemberRole) models.CommunityMemberRole {
	if role == models.CommunityMemberRole("manager") {
		return models.Moderator
	}
	return role
}

func (s *Service) postMedia(ctx context.Context, postID int64) []Media {
	mediaIDs := s.store.PostWithMedia.GetMediaByPostID(ctx, postID)
	items := make([]Media, 0, len(mediaIDs))
	for _, mediaID := range mediaIDs {
		if media := s.media(ctx, mediaID); media != nil {
			items = append(items, *media)
		}
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

	var URL string

	if !strings.HasPrefix(resp.GetUrl(), "http") {
		mediaURL, err := s.mediaClient.GetMediaURL(ctx, &mediapb.GetMediaURLRequest{MediaId: resp.GetMediaId()})
		if err != nil {
			return nil
		}
		URL = mediaURL.GetUrl()
	} else {
		URL = resp.Url
	}

	return &Media{ID: resp.GetMediaId(), UID: resp.GetUid(), MimeType: resp.GetMimeType(), URL: URL}
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

func (s *Service) attachMedia(ctx context.Context, postID int64, medias []dto.MediaRequestData) error {
	for i, media := range medias {
		if media.MediaID <= 0 {
			return ErrInvalidInput
		}
		if s.mediaClient != nil {
			resp, err := s.mediaClient.GetMedia(ctx, &mediapb.GetMediaRequest{MediaId: media.MediaID})
			if err != nil || resp == nil {
				return ErrMediaAttachmentError
			}
		}
		if err := s.store.PostWithMedia.Save(ctx, *models.NewPostWithMedia(postID, media.MediaID, i)); err != nil {
			return ErrMediaAttachmentError
		}
	}
	return nil
}

func normalizePostError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, xerrors.PostNotFound) || pgxscan.NotFound(err) {
		return ErrPostNotFound
	}
	return err
}

func ToStatus(err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrPostContentRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, ErrPostNotFound), errors.Is(err, ErrProfileNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
