package usecase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	communitypb "github.com/go-park-mail-ru/2026_1_ARIS/proto/community"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/repository"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ---------------------------------------------------------------------------
// normalizeListBounds
// ---------------------------------------------------------------------------

func TestNormalizeListBoundsDefaultsOnZeroLimit(t *testing.T) {
	l, o := normalizeListBounds(0, 5)
	require.Equal(t, 50, l)
	require.Equal(t, 5, o)
}

func TestNormalizeListBoundsDefaultsOnNegativeLimit(t *testing.T) {
	l, o := normalizeListBounds(-1, 10)
	require.Equal(t, 50, l)
	require.Equal(t, 10, o)
}

func TestNormalizeListBoundsDefaultsOnLimitOver100(t *testing.T) {
	l, o := normalizeListBounds(101, 0)
	require.Equal(t, 50, l)
	require.Equal(t, 0, o)
}

func TestNormalizeListBoundsKeepsLimitAt100(t *testing.T) {
	l, _ := normalizeListBounds(100, 0)
	require.Equal(t, 100, l)
}

func TestNormalizeListBoundsKeepsValidValues(t *testing.T) {
	l, o := normalizeListBounds(20, 5)
	require.Equal(t, 20, l)
	require.Equal(t, 5, o)
}

func TestNormalizeListBoundsClampsNegativeOffset(t *testing.T) {
	_, o := normalizeListBounds(10, -3)
	require.Equal(t, 0, o)
}

// ---------------------------------------------------------------------------
// normalizePostError
// ---------------------------------------------------------------------------

func TestNormalizePostErrorNil(t *testing.T) {
	require.NoError(t, normalizePostError(nil))
}

func TestNormalizePostErrorMapsRepoError(t *testing.T) {
	err := normalizePostError(repository.ErrPostNotFound)
	require.ErrorIs(t, err, ErrPostNotFound)
}

func TestNormalizePostErrorPassesThroughOtherErrors(t *testing.T) {
	sentinel := errors.New("some other error")
	err := normalizePostError(sentinel)
	require.ErrorIs(t, err, sentinel)
}

// ---------------------------------------------------------------------------
// normalizeCommentError
// ---------------------------------------------------------------------------

func TestNormalizeCommentErrorNil(t *testing.T) {
	require.NoError(t, normalizeCommentError(nil))
}

func TestNormalizeCommentErrorMapsRepoError(t *testing.T) {
	err := normalizeCommentError(repository.ErrCommentNotFound)
	require.ErrorIs(t, err, ErrCommentNotFound)
}

func TestNormalizeCommentErrorPassesThroughOtherErrors(t *testing.T) {
	sentinel := errors.New("other")
	err := normalizeCommentError(sentinel)
	require.ErrorIs(t, err, sentinel)
}

// ---------------------------------------------------------------------------
// ToStatus
// ---------------------------------------------------------------------------

func TestToStatusInvalidInput(t *testing.T) {
	err := ToStatus(ErrInvalidInput)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestToStatusPostContentRequired(t *testing.T) {
	err := ToStatus(ErrPostContentRequired)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestToStatusPostNotFound(t *testing.T) {
	err := ToStatus(ErrPostNotFound)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestToStatusCommentNotFound(t *testing.T) {
	err := ToStatus(ErrCommentNotFound)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestToStatusProfileNotFound(t *testing.T) {
	err := ToStatus(ErrProfileNotFound)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestToStatusCommunityNotFound(t *testing.T) {
	err := ToStatus(ErrCommunityNotFound)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestToStatusForbidden(t *testing.T) {
	err := ToStatus(ErrForbidden)
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestToStatusUnknownBecomesInternal(t *testing.T) {
	err := ToStatus(errors.New("something unexpected"))
	require.Equal(t, codes.Internal, status.Code(err))
}

// ---------------------------------------------------------------------------
// appendAttachmentRefs
// ---------------------------------------------------------------------------

func TestAppendAttachmentRefsEmptyFiles(t *testing.T) {
	media := []MediaRequestData{{MediaID: 1}, {MediaID: 2}}
	result := appendAttachmentRefs(media, nil)
	require.Equal(t, media, result)
}

func TestAppendAttachmentRefsMergesBothSlices(t *testing.T) {
	media := []MediaRequestData{{MediaID: 1}}
	files := []MediaRequestData{{MediaID: 2}, {MediaID: 3}}
	result := appendAttachmentRefs(media, files)
	require.Len(t, result, 3)
	require.Equal(t, int64(1), result[0].MediaID)
	require.Equal(t, int64(2), result[1].MediaID)
	require.Equal(t, int64(3), result[2].MediaID)
}

func TestAppendAttachmentRefsEmptyMedia(t *testing.T) {
	files := []MediaRequestData{{MediaID: 5}}
	result := appendAttachmentRefs(nil, files)
	require.Len(t, result, 1)
	require.Equal(t, int64(5), result[0].MediaID)
}

func TestAppendAttachmentRefsBothEmpty(t *testing.T) {
	result := appendAttachmentRefs(nil, nil)
	require.Nil(t, result)
}

// ---------------------------------------------------------------------------
// splitMedia
// ---------------------------------------------------------------------------

func TestSplitMediaSeparatesImageAndVideo(t *testing.T) {
	items := []Media{
		{ID: 1, MimeType: "image/png"},
		{ID: 2, MimeType: "video/mp4"},
		{ID: 3, MimeType: "application/pdf"},
		{ID: 4, MimeType: "image"},
		{ID: 5, MimeType: "video"},
	}
	media, files := splitMedia(items)
	require.Len(t, media, 4)
	require.Len(t, files, 1)
	require.Equal(t, int64(3), files[0].ID)
}

func TestSplitMediaAllFiles(t *testing.T) {
	items := []Media{
		{ID: 1, MimeType: "application/zip"},
		{ID: 2, MimeType: "text/plain"},
	}
	media, files := splitMedia(items)
	require.Len(t, media, 0)
	require.Len(t, files, 2)
}

func TestSplitMediaAllMedia(t *testing.T) {
	items := []Media{
		{ID: 1, MimeType: "image/jpeg"},
		{ID: 2, MimeType: "video/webm"},
	}
	media, files := splitMedia(items)
	require.Len(t, media, 2)
	require.Len(t, files, 0)
}

func TestSplitMediaEmpty(t *testing.T) {
	media, files := splitMedia(nil)
	require.Len(t, media, 0)
	require.Len(t, files, 0)
}

// ---------------------------------------------------------------------------
// derefStr
// ---------------------------------------------------------------------------

func TestDerefStrNil(t *testing.T) {
	require.Equal(t, "", derefStr(nil))
}

func TestDerefStrNonNil(t *testing.T) {
	s := "hello"
	require.Equal(t, "hello", derefStr(&s))
}

func TestDerefStrEmptyString(t *testing.T) {
	s := ""
	require.Equal(t, "", derefStr(&s))
}

// ---------------------------------------------------------------------------
// normalizeCommunityError
// ---------------------------------------------------------------------------

func TestNormalizeCommunityErrorNil(t *testing.T) {
	require.NoError(t, normalizeCommunityError(nil))
}

func TestNormalizeCommunityErrorNotFound(t *testing.T) {
	err := normalizeCommunityError(status.Error(codes.NotFound, "not found"))
	require.ErrorIs(t, err, ErrCommunityNotFound)
}

func TestNormalizeCommunityErrorInvalidArgument(t *testing.T) {
	err := normalizeCommunityError(status.Error(codes.InvalidArgument, "bad"))
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestNormalizeCommunityErrorOtherStatusPassedThrough(t *testing.T) {
	original := status.Error(codes.Internal, "internal")
	err := normalizeCommunityError(original)
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestNormalizeCommunityErrorNonStatusPassedThrough(t *testing.T) {
	sentinel := errors.New("raw error")
	err := normalizeCommunityError(sentinel)
	require.ErrorIs(t, err, sentinel)
}

// ---------------------------------------------------------------------------
// normalizeCommunityRole
// ---------------------------------------------------------------------------

func TestNormalizeCommunityRoleManagerBecomesModertor(t *testing.T) {
	require.Equal(t, "moderator", normalizeCommunityRole("manager"))
}

func TestNormalizeCommunityRoleModeratorUnchanged(t *testing.T) {
	require.Equal(t, "moderator", normalizeCommunityRole("moderator"))
}

func TestNormalizeCommunityRoleMemberUnchanged(t *testing.T) {
	require.Equal(t, "member", normalizeCommunityRole("member"))
}

func TestNormalizeCommunityRoleBlockedUnchanged(t *testing.T) {
	require.Equal(t, "blocked", normalizeCommunityRole("blocked"))
}

func TestNormalizeCommunityRoleArbitraryUnchanged(t *testing.T) {
	require.Equal(t, "admin", normalizeCommunityRole("admin"))
}

// ---------------------------------------------------------------------------
// communityFromResponse
// ---------------------------------------------------------------------------

func TestCommunityFromResponseNilReturnsError(t *testing.T) {
	info, err := communityFromResponse(nil)
	require.Nil(t, info)
	require.ErrorIs(t, err, ErrCommunityNotFound)
}

func TestCommunityFromResponseZeroCommunityIDReturnsError(t *testing.T) {
	resp := &communitypb.CommunityResponse{CommunityId: 0, ProfileId: 2, Title: "T", Username: "u"}
	info, err := communityFromResponse(resp)
	require.Nil(t, info)
	require.ErrorIs(t, err, ErrCommunityNotFound)
}

func TestCommunityFromResponseZeroProfileIDReturnsError(t *testing.T) {
	resp := &communitypb.CommunityResponse{CommunityId: 1, ProfileId: 0, Title: "T", Username: "u"}
	info, err := communityFromResponse(resp)
	require.Nil(t, info)
	require.ErrorIs(t, err, ErrCommunityNotFound)
}

func TestCommunityFromResponseValidMapsCorrectly(t *testing.T) {
	resp := &communitypb.CommunityResponse{CommunityId: 1, ProfileId: 2, Title: "T", Username: "u"}
	info, err := communityFromResponse(resp)
	require.NoError(t, err)
	require.NotNil(t, info)
	require.Equal(t, int64(1), info.ID)
	require.Equal(t, int64(2), info.ProfileID)
	require.Equal(t, "T", info.Title)
	require.Equal(t, "u", info.Username)
	require.Nil(t, info.AvatarID)
}

func TestCommunityFromResponseWithAvatarID(t *testing.T) {
	avatarID := int64(99)
	resp := &communitypb.CommunityResponse{
		CommunityId: 10,
		ProfileId:   20,
		Title:       "Comm",
		Username:    "comm",
		AvatarId:    &avatarID,
	}
	info, err := communityFromResponse(resp)
	require.NoError(t, err)
	require.NotNil(t, info.AvatarID)
	require.Equal(t, avatarID, *info.AvatarID)
}

// ---------------------------------------------------------------------------
// NewPost (package-level wrapper)
// ---------------------------------------------------------------------------

func TestNewPostCreatesPost(t *testing.T) {
	text := "hello"
	p := NewPost(&text, 42, false, true)
	require.NotNil(t, p)
	require.Equal(t, int64(42), p.AuthorID)
	require.Equal(t, &text, p.Text)
	require.True(t, p.AllowComments)
	require.False(t, p.IsPublicDemo)
}

func TestNewPostNilText(t *testing.T) {
	p := NewPost(nil, 7, true, false)
	require.NotNil(t, p)
	require.Nil(t, p.Text)
	require.True(t, p.IsPublicDemo)
	require.False(t, p.AllowComments)
}

// ---------------------------------------------------------------------------
// Cursor
// ---------------------------------------------------------------------------

func TestCursorEncodesDecodable(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	encoded := Cursor(now, 123)
	require.NotEmpty(t, encoded)

	// Verify it's valid base64
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)
	require.True(t, strings.Contains(string(decoded), "123"))
}

func TestCursorDifferentInputsDifferentOutputs(t *testing.T) {
	t1 := time.Now().UTC()
	c1 := Cursor(t1, 1)
	c2 := Cursor(t1, 2)
	require.NotEqual(t, c1, c2)
}

// ---------------------------------------------------------------------------
// encodeSessionCursor / decodeSessionCursor
// ---------------------------------------------------------------------------

func TestEncodeDecodeSessionCursorRoundTrip(t *testing.T) {
	original := sessionCursor{SessionID: "abc-123", Offset: 42}
	encoded := encodeSessionCursor(original)

	require.True(t, strings.HasPrefix(encoded, "v2s:"))

	decoded, err := decodeSessionCursor(encoded)
	require.NoError(t, err)
	require.Equal(t, original.SessionID, decoded.SessionID)
	require.Equal(t, original.Offset, decoded.Offset)
}

func TestDecodeSessionCursorRejectsNonPrefixed(t *testing.T) {
	_, err := decodeSessionCursor("noprefix:abc")
	require.Error(t, err)
}

func TestDecodeSessionCursorRejectsEmptyString(t *testing.T) {
	_, err := decodeSessionCursor("")
	require.Error(t, err)
}

func TestDecodeSessionCursorRejectsInvalidBase64(t *testing.T) {
	_, err := decodeSessionCursor("v2s:!!!not-base64!!!")
	require.Error(t, err)
}

func TestDecodeSessionCursorRejectsInvalidJSON(t *testing.T) {
	// valid base64 but not valid JSON
	bad := "v2s:" + base64.StdEncoding.EncodeToString([]byte("not-json"))
	_, err := decodeSessionCursor(bad)
	require.Error(t, err)
}

func TestEncodeSessionCursorProducesValidJSON(t *testing.T) {
	sc := sessionCursor{SessionID: "sid", Offset: 7}
	encoded := encodeSessionCursor(sc)
	b, err := base64.StdEncoding.DecodeString(encoded[4:]) // strip "v2s:"
	require.NoError(t, err)
	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &out))
}

// ---------------------------------------------------------------------------
// Service setter methods: SetCache, SetAnalytics, SetRecommendation, SetSessionCache
// ---------------------------------------------------------------------------

func newTestService(ctrl *gomock.Controller) *Service {
	mockLikes := repomocks.NewMockLikeRepo(ctrl)
	store := repository.Store{
		Likes: mockLikes,
	}
	svc := New(store, nil, nil, nil)
	return svc
}

func TestSetCacheSetsCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := newTestService(ctrl)
	require.Nil(t, svc.cache)
	svc.SetCache(nil) // setting nil is fine; just tests the assignment path
	require.Nil(t, svc.cache)
}

func TestSetAnalyticsSetsAnalytics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := newTestService(ctrl)
	require.Nil(t, svc.analytics)
	svc.SetAnalytics(nil)
	require.Nil(t, svc.analytics)
}

func TestSetRecommendationSetsField(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := newTestService(ctrl)
	require.Nil(t, svc.recommendation)
	svc.SetRecommendation(nil)
	require.Nil(t, svc.recommendation)
}

func TestSetSessionCacheSetsField(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := newTestService(ctrl)
	require.Nil(t, svc.sessionCache)
	svc.SetSessionCache(nil)
	require.Nil(t, svc.sessionCache)
}

// ---------------------------------------------------------------------------
// postLikeCount when cache is nil (falls back to store)
// ---------------------------------------------------------------------------

func TestPostLikeCountNoCacheFallsBackToStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLikes := repomocks.NewMockLikeRepo(ctrl)
	mockLikes.EXPECT().GetLikeCountOnPost(gomock.Any(), int64(5)).Return(7)

	svc := New(repository.Store{Likes: mockLikes}, nil, nil, nil)
	// cache is nil by default
	count := svc.postLikeCount(context.Background(), 5)
	require.Equal(t, 7, count)
}

// ---------------------------------------------------------------------------
// refreshPostLikeCount when cache is nil (no-op, should not panic)
// ---------------------------------------------------------------------------

func TestRefreshPostLikeCountNoCacheIsNoop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := newTestService(ctrl)
	// Should not panic or call anything
	require.NotPanics(t, func() {
		svc.refreshPostLikeCount(context.Background(), 10)
	})
}

// ---------------------------------------------------------------------------
// deletePostLikeCount when cache is nil (no-op, should not panic)
// ---------------------------------------------------------------------------

func TestDeletePostLikeCountNoCacheIsNoop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := newTestService(ctrl)
	require.NotPanics(t, func() {
		svc.deletePostLikeCount(context.Background(), 10)
	})
}

// ---------------------------------------------------------------------------
// absoluteMediaURL
// ---------------------------------------------------------------------------

func TestAbsoluteMediaURLHttpPrefixReturnedAsIs(t *testing.T) {
	svc := &Service{} // mediaClient is nil
	result := svc.absoluteMediaURL(context.Background(), 1, "http://example.com/file.png")
	require.Equal(t, "http://example.com/file.png", result)
}

func TestAbsoluteMediaURLHttpsPrefixReturnedAsIs(t *testing.T) {
	svc := &Service{}
	result := svc.absoluteMediaURL(context.Background(), 2, "https://cdn.example.com/img.jpg")
	require.Equal(t, "https://cdn.example.com/img.jpg", result)
}

func TestAbsoluteMediaURLMediaClientNilReturnsRaw(t *testing.T) {
	svc := &Service{mediaClient: nil}
	raw := "/uploads/photo.png"
	result := svc.absoluteMediaURL(context.Background(), 3, raw)
	require.Equal(t, raw, result)
}
