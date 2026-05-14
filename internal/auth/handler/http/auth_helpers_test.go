package http

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	authrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/auth/repository"
	authservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/auth/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	profilemock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile/mock"
	sessionmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/session/mock"
	accountmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account/mock"
	userprofilemock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type fakeRoleProvider struct {
	role models.SupportRole
	err  error
}

func (f fakeRoleProvider) GetProfileRole(context.Context, int64) (models.SupportRole, error) {
	return f.role, f.err
}

func TestAuthHandlerHelpers(t *testing.T) {
	handler := New(nil, true, fakeRoleProvider{role: models.SupportRoleAdmin})
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	rec := httptest.NewRecorder()
	expiresAt := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	handler.setSessionCookie(rec, models.SessionID("sid"), expiresAt)
	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	require.Equal(t, "sid", cookies[0].Value)
	require.True(t, cookies[0].Secure)
	require.Equal(t, expiresAt, cookies[0].Expires)

	require.Equal(t, models.Male, parseGender(1))
	require.Equal(t, models.Female, parseGender(2))
	require.Equal(t, "", derefString(nil))
	email := "neo@example.test"
	require.Equal(t, email, derefString(&email))

	user := authservice.User{
		UserAccountID: 10,
		ProfileID:     20,
		FirstName:     "Neo",
		LastName:      "Anderson",
		Login:         "neo",
		Email:         &email,
		CreatedAt:     expiresAt,
	}
	resp := handler.mapUser(context.Background(), user)
	require.Equal(t, int64(20), resp.ID)
	require.Equal(t, string(models.SupportRoleAdmin), resp.Role)
	require.NotEmpty(t, resp.CreatedAt)

	handler = New(nil, false, fakeRoleProvider{err: errors.New("no role")})
	require.Equal(t, models.SupportRoleUser, handler.supportRole(context.Background(), 20))
	require.Equal(t, models.SupportRoleUser, New(nil, false).supportRole(context.Background(), 20))
}

func TestAuthDecodeAndErrorResponses(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"login":"neo"}`))
	rec := httptest.NewRecorder()
	var login loginRequest
	require.True(t, decodeJSON(rec, req, &login))
	require.Equal(t, "neo", login.Login)

	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{`))
	rec = httptest.NewRecorder()
	require.False(t, decodeJSON(rec, req, &login))
	require.Equal(t, http.StatusBadRequest, rec.Code)

	for _, tc := range []struct {
		err  error
		code int
	}{
		{authservice.ErrLoginAlreadyExists, http.StatusConflict},
		{authservice.ErrInvalidCredentials, http.StatusUnauthorized},
		{authservice.ErrSessionNotFound, http.StatusUnauthorized},
		{authservice.ErrInvalidInput, http.StatusBadRequest},
		{authservice.ErrInvalidBirthday, http.StatusBadRequest},
		{authservice.ErrTooYoung, http.StatusBadRequest},
		{errors.New("boom"), http.StatusInternalServerError},
	} {
		rec = httptest.NewRecorder()
		writeServiceError(rec, tc.err)
		require.Equal(t, tc.code, rec.Code)
	}

	rec = httptest.NewRecorder()
	writeJSON(rec, http.StatusAccepted, map[string]bool{"ok": true})
	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

type authHTTPMocks struct {
	accounts     *accountmock.MockUserAccountRepo
	profiles     *profilemock.MockProfileRepo
	userProfiles *userprofilemock.MockUserProfileRepo
	sessions     *sessionmock.MockSessionRepo
	handler      *Handler
}

func newAuthHTTPMocks(t *testing.T) (*gomock.Controller, authHTTPMocks) {
	t.Helper()

	ctrl := gomock.NewController(t)
	m := authHTTPMocks{
		accounts:     accountmock.NewMockUserAccountRepo(ctrl),
		profiles:     profilemock.NewMockProfileRepo(ctrl),
		userProfiles: userprofilemock.NewMockUserProfileRepo(ctrl),
		sessions:     sessionmock.NewMockSessionRepo(ctrl),
	}
	auth := authservice.New(authrepo.NewStore(m.accounts, m.profiles, m.userProfiles, m.sessions))
	m.handler = New(auth, true, fakeRoleProvider{role: models.SupportRoleAdmin})
	return ctrl, m
}

func expectAuthHTTPUser(m authHTTPMocks, accountID, profileID int64) {
	m.accounts.EXPECT().Get(gomock.Any(), accountID).Return(&models.UserAccount{ID: accountID, Username: "neo"}, nil)
	m.userProfiles.EXPECT().GetByUserAccountID(gomock.Any(), accountID).Return(&models.UserProfile{
		UserAccountID: accountID, ProfileID: profileID, FirstName: "Neo", LastName: "Anderson",
	}, nil)
	m.profiles.EXPECT().GetByUserAccountID(gomock.Any(), accountID).Return(&models.Profile{ID: profileID}, nil)
}

func TestAuthHandlerEndpoints(t *testing.T) {
	ctrl, m := newAuthHTTPMocks(t)
	defer ctrl.Finish()

	m.accounts.EXPECT().GetByUsername(gomock.Any(), "neo").Return(nil, errors.New("not found"))
	rec := httptest.NewRecorder()
	m.handler.RegisterStepOne(rec, httptest.NewRequest(http.MethodPost, "/register/step-one", bytes.NewBufferString(`{"login":"neo","password1":"secret","password2":"secret"}`)))
	require.Equal(t, http.StatusOK, rec.Code)

	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err)
	accountID := int64(10)
	profileID := int64(20)
	m.accounts.EXPECT().GetByUsername(gomock.Any(), "neo").Return(&models.UserAccount{ID: accountID, Username: "neo", PasswordHash: string(hash)}, nil)
	m.sessions.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
	expectAuthHTTPUser(m, accountID, profileID)
	rec = httptest.NewRecorder()
	m.handler.Login(rec, httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"login":"neo","password":"secret"}`)))
	require.Equal(t, http.StatusOK, rec.Code)
	require.NotEmpty(t, rec.Result().Cookies())

	m.sessions.EXPECT().GetByID(gomock.Any(), models.SessionID("sid")).Return(&models.Session{
		SessionID: "sid", UserID: accountID, CreatedAt: time.Now(), ExpiredAt: time.Now().Add(time.Hour),
	}, nil)
	expectAuthHTTPUser(m, accountID, profileID)
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid"})
	rec = httptest.NewRecorder()
	m.handler.Me(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"role":"admin"`)

	m.sessions.EXPECT().Delete(gomock.Any(), models.SessionID("sid")).Return(nil)
	req = httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sid"})
	rec = httptest.NewRecorder()
	m.handler.Logout(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}
