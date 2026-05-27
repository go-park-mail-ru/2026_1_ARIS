package http

import (
	"bytes"
	"context"
	"encoding/json"
	netHTTP "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	mediamock "github.com/go-park-mail-ru/2026_1_ARIS/proto/media/mock"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/model"
	repomock "github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/repository/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/usecase"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
)

type fixedRoleProvider struct {
	role model.SupportRole
}

func (p fixedRoleProvider) GetProfileRole(context.Context, int64) (model.SupportRole, error) {
	return p.role, nil
}

func TestAuthHTTPHandlerRoutes(t *testing.T) {
	t.Parallel()

	t.Run("register step one", func(t *testing.T) {
		t.Parallel()
		router, _, users, _ := newAuthRouter(t)
		users.EXPECT().
			CheckUsernameAvailable(gomock.Any(), &userpb.CheckUsernameAvailableRequest{Username: "neo"}).
			Return(&userpb.CheckUsernameAvailableResponse{Available: true}, nil)

		rr := serveAuth(t, router, "POST", "/register/step-one", map[string]any{
			"login": " Neo ", "password1": "pass1234", "password2": "pass1234",
		}, "")

		require.Equal(t, 200, rr.Code)
	})

	t.Run("register", func(t *testing.T) {
		t.Parallel()
		router, sessions, users, media := newAuthRouter(t)
		users.EXPECT().
			CheckUsernameAvailable(gomock.Any(), &userpb.CheckUsernameAvailableRequest{Username: "neo"}).
			Return(&userpb.CheckUsernameAvailableResponse{Available: true}, nil)
		users.EXPECT().
			CreateAuthUser(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, req *userpb.CreateAuthUserRequest, _ ...grpc.CallOption) (*userpb.AuthUserResponse, error) {
				require.Equal(t, "neo", req.Username)
				require.Equal(t, "2010-05-27", req.Birthday)
				require.NoError(t, bcrypt.CompareHashAndPassword([]byte(req.PasswordHash), []byte("pass1234")))
				return &userpb.AuthUserResponse{UserAccountId: 42}, nil
			})
		expectHTTPAuthResult(sessions, users, media, 42)

		rr := serveAuth(t, router, "POST", "/register", map[string]any{
			"firstName": "Neo", "lastName": "Anderson", "login": "Neo",
			"password1": "pass1234", "password2": "pass1234", "birthday": "2010-05-27", "gender": 1,
		}, "")

		require.Equal(t, 201, rr.Code)
		require.NotEmpty(t, rr.Result().Cookies())
	})

	t.Run("login", func(t *testing.T) {
		t.Parallel()
		router, sessions, users, media := newAuthRouter(t)
		hash := passwordHash(t, "pass1234")
		users.EXPECT().
			GetCredentialsByLogin(gomock.Any(), &userpb.GetCredentialsByLoginRequest{Login: "neo"}).
			Return(&userpb.GetCredentialsByLoginResponse{UserAccountId: 42, PasswordHash: hash}, nil)
		expectHTTPAuthResult(sessions, users, media, 42)

		rr := serveAuth(t, router, "POST", "/login", map[string]any{"login": "Neo", "password": "pass1234"}, "")

		require.Equal(t, 200, rr.Code)
		require.Contains(t, rr.Header().Get("Set-Cookie"), sessionCookieName)
	})

	t.Run("me", func(t *testing.T) {
		t.Parallel()
		router, sessions, users, media := newAuthRouter(t)
		sessions.EXPECT().
			GetByID(gomock.Any(), model.SessionID("active")).
			Return(&model.Session{SessionID: "active", UserID: 42, ExpiredAt: time.Now().Add(time.Hour)}, nil)
		users.EXPECT().
			GetAuthUserByAccount(gomock.Any(), &userpb.GetAuthUserByAccountRequest{UserAccountId: 42}).
			Return(authHTTPUser(42), nil)
		media.EXPECT().
			GetMediaURL(gomock.Any(), &mediapb.GetMediaURLRequest{MediaId: 9}).
			Return(&mediapb.GetMediaURLResponse{Url: "/media/avatar.png"}, nil)

		rr := serveAuth(t, router, "GET", "/me", nil, "active")

		require.Equal(t, 200, rr.Code)
		require.Contains(t, rr.Body.String(), `"role":"admin"`)
	})

	t.Run("change password", func(t *testing.T) {
		t.Parallel()
		router, sessions, users, _ := newAuthRouter(t)
		oldHash := passwordHash(t, "oldpass1")
		sessions.EXPECT().
			GetByID(gomock.Any(), model.SessionID("active")).
			Return(&model.Session{SessionID: "active", UserID: 42, ExpiredAt: time.Now().Add(time.Hour)}, nil)
		users.EXPECT().
			GetAuthUserByAccount(gomock.Any(), &userpb.GetAuthUserByAccountRequest{UserAccountId: 42}).
			Return(authHTTPUserWithoutAvatar(42), nil)
		users.EXPECT().
			GetCredentialsByLogin(gomock.Any(), &userpb.GetCredentialsByLoginRequest{Login: "neo"}).
			Return(&userpb.GetCredentialsByLoginResponse{UserAccountId: 42, PasswordHash: oldHash}, nil)
		users.EXPECT().
			UpdatePasswordHash(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, req *userpb.UpdatePasswordHashRequest, _ ...grpc.CallOption) (*userpb.UpdatePasswordHashResponse, error) {
				require.Equal(t, int64(42), req.UserAccountId)
				require.NoError(t, bcrypt.CompareHashAndPassword([]byte(req.PasswordHash), []byte("newpass1")))
				return &userpb.UpdatePasswordHashResponse{Ok: true}, nil
			})
		sessions.EXPECT().Delete(gomock.Any(), model.SessionID("active")).Return(nil)

		rr := serveAuth(t, router, "POST", "/password", map[string]any{
			"oldPassword": "oldpass1", "newPassword1": "newpass1", "newPassword2": "newpass1",
		}, "active")

		require.Equal(t, 200, rr.Code)
		require.Contains(t, rr.Header().Get("Set-Cookie"), "Max-Age=0")
	})

	t.Run("logout", func(t *testing.T) {
		t.Parallel()
		router, sessions, _, _ := newAuthRouter(t)
		sessions.EXPECT().Delete(gomock.Any(), model.SessionID("active")).Return(nil)

		rr := serveAuth(t, router, "POST", "/logout", nil, "active")

		require.Equal(t, 200, rr.Code)
		require.Contains(t, rr.Header().Get("Set-Cookie"), "Max-Age=0")
	})
}

func TestAuthHTTPHandlerOAuthAndHelpers(t *testing.T) {
	t.Parallel()

	router, _, _, _ := newAuthRouter(t)

	rr := serveAuth(t, router, "GET", "/vkid/login", nil, "")
	require.Equal(t, 302, rr.Code)
	require.Contains(t, rr.Header().Get("Location"), "client_id=client")
	require.Contains(t, rr.Header().Get("Set-Cookie"), vkidStateCookieName)

	rr = serveAuth(t, router, "GET", "/vkid/callback?state=missing&code=code", nil, "")
	require.Equal(t, 302, rr.Code)
	require.Contains(t, rr.Header().Get("Location"), "error=invalid_state")

	require.Equal(t, model.Male, parseGender(1))
	require.Equal(t, model.Female, parseGender(2))
	require.Equal(t, "/", sanitizeReturnTo("https://evil.test", "/ok"))
	require.Equal(t, "/ok", sanitizeReturnTo("", "/ok"))
	require.Equal(t, "/safe", sanitizeReturnTo("/safe", "/ok"))
	base, query := splitURLQuery("https://id.test/auth?foo=bar")
	require.Equal(t, "https://id.test/auth", base)
	require.Equal(t, "bar", query.Get("foo"))
	require.NotEmpty(t, codeChallenge("verifier"))
	token, err := randomURLToken(8)
	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func newAuthRouter(t *testing.T) (*chi.Mux, *repomock.MockSessionRepo, *usermock.MockUserServiceClient, *mediamock.MockMediaServiceClient) {
	t.Helper()
	ctrl := gomock.NewController(t)
	sessions := repomock.NewMockSessionRepo(ctrl)
	users := usermock.NewMockUserServiceClient(ctrl)
	media := mediamock.NewMockMediaServiceClient(ctrl)
	svc := usecase.New(sessions, users, media)
	handler := New(svc, false, fixedRoleProvider{role: model.SupportRoleAdmin})
	handler.ConfigureVKID(VKIDConfig{
		ClientID:            "client",
		AuthorizeURL:        "https://id.test/auth?existing=1",
		RedirectURI:         "https://api.test/api/auth/vkid/callback",
		FrontendSuccessPath: "/feed",
		FrontendErrorPath:   "/login",
	})
	router := chi.NewRouter()
	handler.RegisterRoutes(router)
	return router, sessions, users, media
}

func serveAuth(t *testing.T, router *chi.Mux, method, path string, body any, sessionID string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if sessionID != "" {
		req.AddCookie(&netHTTP.Cookie{Name: sessionCookieName, Value: sessionID})
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func expectHTTPAuthResult(sessions *repomock.MockSessionRepo, users *usermock.MockUserServiceClient, media *mediamock.MockMediaServiceClient, accountID int64) {
	sessions.EXPECT().
		Save(gomock.Any(), gomock.AssignableToTypeOf(model.Session{})).
		Return(nil)
	users.EXPECT().
		GetAuthUserByAccount(gomock.Any(), &userpb.GetAuthUserByAccountRequest{UserAccountId: accountID}).
		Return(authHTTPUser(accountID), nil)
	media.EXPECT().
		GetMediaURL(gomock.Any(), &mediapb.GetMediaURLRequest{MediaId: 9}).
		Return(&mediapb.GetMediaURLResponse{Url: "/media/avatar.png"}, nil)
}

func authHTTPUser(accountID int64) *userpb.AuthUserResponse {
	avatarID := int64(9)
	email := "neo@example.com"
	return &userpb.AuthUserResponse{
		UserAccountId: accountID,
		UserProfileId: 20,
		ProfileId:     30,
		Login:         "neo",
		Email:         &email,
		FirstName:     "Neo",
		LastName:      "Anderson",
		AvatarId:      &avatarID,
		CreatedAt:     "2026-05-27T12:00:00Z",
	}
}

func authHTTPUserWithoutAvatar(accountID int64) *userpb.AuthUserResponse {
	user := authHTTPUser(accountID)
	user.AvatarId = nil
	return user
}

func passwordHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	return string(hash)
}
