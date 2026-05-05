package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/community/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCommunityMappingHelpers(t *testing.T) {
	bio := "<bio>"
	role := models.Owner
	avatarID := int64(7)
	coverID := int64(8)
	avatarURL := "https://cdn.test/a.png"
	coverURL := "https://cdn.test/c.png"
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	resp := mapDetails(&service.Details{
		Community: models.Community{
			ID: 1, Uid: uuid.New(), ProfileID: 2, Username: "<team>", Title: "<Team>", Bio: &bio,
			Type: models.PublicGroup, CoverMediaID: &coverID, IsActive: true, CreatedAt: now, UpdatedAt: now,
		},
		AvatarID: &avatarID, AvatarURL: &avatarURL, CoverURL: &coverURL,
		Membership: service.Membership{IsMember: true, Role: &role},
		Permission: service.Permissions{CanEditCommunity: true, CanDeleteCommunity: true, CanPost: true},
	})
	require.Equal(t, "&lt;team&gt;", resp.Community.Username)
	require.Equal(t, "&lt;Team&gt;", resp.Community.Title)
	require.Equal(t, "&lt;bio&gt;", *resp.Community.Bio)
	require.True(t, resp.Membership.IsMember)
	require.True(t, resp.Permissions.CanPost)

	member := mapMember(service.MemberDetails{
		ProfileID: 2, UserAccountID: 10, FirstName: "<Neo>", LastName: "A", Username: "<neo>",
		AvatarID: &avatarID, AvatarURL: &avatarURL, Role: models.Admin, IsSelf: true, JoinedAt: now.Format(time.RFC3339),
	})
	require.Equal(t, "&lt;Neo&gt;", member.FirstName)
	require.Equal(t, "&lt;neo&gt;", member.Username)
	require.True(t, member.IsSelf)
	require.Nil(t, escapePtr(nil))
	require.Equal(t, "&lt;bio&gt;", *escapePtr(&bio))
}

func TestCommunityHTTPHelpersAndErrors(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?limit=bad&includeBlocked=true", nil)
	require.Equal(t, 20, parseIntQuery(req, "limit", 20))
	require.True(t, parseBoolQuery(req, "includeBlocked"))

	id, ok := parseID(rec, "10")
	require.True(t, ok)
	require.Equal(t, int64(10), id)

	rec = httptest.NewRecorder()
	_, ok = parseID(rec, "bad")
	require.False(t, ok)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	userReq := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(context.WithValue(context.Background(), "user_id", int64(10)))
	rec = httptest.NewRecorder()
	userID, ok := userIDFromContext(rec, userReq)
	require.True(t, ok)
	require.Equal(t, int64(10), userID)
	require.Equal(t, int64(10), *optionalUserID(userReq))
	require.Nil(t, optionalUserID(httptest.NewRequest(http.MethodGet, "/", nil)))

	rec = httptest.NewRecorder()
	_, ok = userIDFromContext(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.False(t, ok)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	for _, tc := range []struct {
		err  error
		code int
	}{
		{service.ErrInvalidInput, http.StatusBadRequest},
		{service.ErrNothingToUpdate, http.StatusBadRequest},
		{service.ErrAlreadyExists, http.StatusConflict},
		{service.ErrCommunityNotFound, http.StatusNotFound},
		{service.ErrCommunityMemberNotFound, http.StatusNotFound},
		{service.ErrProfileNotFound, http.StatusNotFound},
		{service.ErrForbidden, http.StatusForbidden},
		{errors.New("boom"), http.StatusInternalServerError},
	} {
		rec = httptest.NewRecorder()
		writeServiceError(rec, tc.err)
		require.Equal(t, tc.code, rec.Code)
	}
}
