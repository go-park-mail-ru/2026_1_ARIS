package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/user/service"
	"github.com/stretchr/testify/require"
)

func TestUserMappingHelpers(t *testing.T) {
	bio := "<bio>"
	town := "<town>"
	avatar := "https://cdn.test/a.png"
	institution := "<uni>"
	group := "101"
	company := "<corp>"
	job := "dev"
	birthday := time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)

	resp := mapProfile(&service.ProfileDetails{
		ProfileID: 20, UserAccountID: 10, Username: "<neo>", FirstName: "<Neo>", LastName: "Anderson",
		Bio: &bio, ImageLink: &avatar, Gender: models.Male, BirthdayDate: birthday, NativeTown: &town,
		Town: &town, Education: []service.Education{{Institution: &institution, Group: &group}, {}},
		Work: []service.Work{{Company: &company, JobTitle: &job}, {}},
	})
	require.Equal(t, "&lt;neo&gt;", resp.Username)
	require.Equal(t, "&lt;Neo&gt;", resp.FirstName)
	require.Equal(t, "&lt;bio&gt;", *resp.Bio)
	require.Equal(t, "2000-01-02", resp.BirthdayDate)
	require.Len(t, resp.Education, 1)
	require.Len(t, resp.Work, 1)

	cards := mapUserCards([]service.UserCard{{ID: 1, FirstName: "A", LastName: "B", Username: "ab", AvatarLink: "url"}})
	require.Len(t, cards, 1)
	require.Equal(t, "1", cards[0].Id)

	require.Nil(t, escapePtr(nil))
	require.Equal(t, "&lt;town&gt;", *escapePtr(&town))

	empty := ""
	email := " "
	phone := "\t"
	req := normalizeOptionalEmptyFields(dto.UpdateFullProfileRequestDTO{Username: &empty, Email: &email, Phone: &phone})
	require.Nil(t, req.Username)
	require.Nil(t, req.Email)
	require.Nil(t, req.Phone)
}

func TestUserHTTPHelpersAndErrors(t *testing.T) {
	rec := httptest.NewRecorder()
	id, ok := parseID(rec, "10", "bad id")
	require.True(t, ok)
	require.Equal(t, int64(10), id)

	rec = httptest.NewRecorder()
	_, ok = parseID(rec, "bad", "bad id")
	require.False(t, ok)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	userID, ok := userIDFromContext(rec, httptest.NewRequest(http.MethodGet, "/", nil).WithContext(context.WithValue(context.Background(), "user_id", int64(10))))
	require.True(t, ok)
	require.Equal(t, int64(10), userID)

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
		{service.ErrProfileNotFound, http.StatusNotFound},
		{service.ErrUserProfileNotFound, http.StatusNotFound},
		{service.ErrUserAccountNotFound, http.StatusNotFound},
		{errors.New("boom"), http.StatusInternalServerError},
	} {
		rec = httptest.NewRecorder()
		writeServiceError(rec, tc.err)
		require.Equal(t, tc.code, rec.Code)
	}
}
