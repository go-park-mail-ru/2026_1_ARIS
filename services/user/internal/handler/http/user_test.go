package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	handlermocks "github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/handler/http/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/repository"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/repository/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/usecase"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUserHTTPProfileSettingsAndFriends(t *testing.T) {
	t.Parallel()

	t.Run("profile me", func(t *testing.T) {
		t.Parallel()
		router, repos := newUserRouter(t)
		expectCurrentUserProfile(repos, 5, 10)
		repos.userProfiles.EXPECT().GetByProfileID(gomock.Any(), int64(10)).Return(userProfile(1, 5, 10), nil)
		repos.accounts.EXPECT().Get(gomock.Any(), int64(5)).Return(userAccount(5, "neo"), nil)
		repos.profiles.EXPECT().Get(gomock.Any(), int64(10)).Return(profile(10), nil)

		rr := serveUser(t, router, http.MethodGet, "/profile/me", nil, 5)

		require.Equal(t, http.StatusOK, rr.Code)
		require.Contains(t, rr.Body.String(), `"username":"neo"`)
	})

	t.Run("get settings", func(t *testing.T) {
		t.Parallel()
		router, repos := newUserRouter(t)
		repos.settings.EXPECT().
			GetByUserID(gomock.Any(), int64(5)).
			Return(&model.UserSettings{UserAccountID: 5, Language: model.LanguageEN, Theme: model.ThemeDark}, nil)

		rr := serveUser(t, router, http.MethodGet, "/settings/", nil, 5)

		require.Equal(t, http.StatusOK, rr.Code)
		require.Contains(t, rr.Body.String(), `"language":"EN"`)
	})

	t.Run("set settings", func(t *testing.T) {
		t.Parallel()
		router, repos := newUserRouter(t)
		repos.settings.EXPECT().
			Update(gomock.Any(), int64(5), gomock.Any()).
			Return(&model.UserSettings{UserAccountID: 5, Language: model.LanguageRU, Theme: model.ThemeLight}, nil)

		rr := serveUser(t, router, http.MethodPost, "/settings/", map[string]any{"language": "RU", "theme": "light"}, 5)

		require.Equal(t, http.StatusOK, rr.Code)
		require.Contains(t, rr.Body.String(), `"theme":"light"`)
	})

	t.Run("friends", func(t *testing.T) {
		t.Parallel()
		router, repos := newUserRouter(t)
		expectCurrentUserProfile(repos, 5, 10)
		repos.friendships.EXPECT().
			GetFriends(gomock.Any(), int64(10), model.FriendshipAccepted).
			Return([]model.Friend{friend(11, "trinity")}, nil)

		rr := serveUser(t, router, http.MethodGet, "/friends/", nil, 5)

		require.Equal(t, http.StatusOK, rr.Code)
		require.Contains(t, rr.Body.String(), `"login":"trinity"`)
	})

	t.Run("edit profile", func(t *testing.T) {
		t.Parallel()
		router, repos := newUserRouter(t)
		repos.userProfiles.EXPECT().GetByUserAccountID(gomock.Any(), int64(5)).Return(userProfile(1, 5, 10), nil)
		repos.accounts.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		repos.userProfiles.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

		rr := serveUser(t, router, http.MethodPatch, "/profile/me/edit", map[string]any{"login": "neo2", "firstName": "Neo"}, 5)

		require.Equal(t, http.StatusNoContent, rr.Code)
	})
}

func TestUserHTTPHelpersAndErrors(t *testing.T) {
	t.Parallel()

	router, _ := newUserRouter(t)
	rr := serveUser(t, router, http.MethodGet, "/profile/me", nil, 0)
	require.Equal(t, http.StatusUnauthorized, rr.Code)

	rr = serveUser(t, router, http.MethodGet, "/profile/0", nil, 5)
	require.Equal(t, http.StatusBadRequest, rr.Code)

	value := " <b>x</b> "
	req := normalizeOptionalEmptyFields(updateProfileRequest{Username: stringPtr(" "), Email: &value})
	require.Nil(t, req.Username)
	require.NotNil(t, req.Email)
	require.Equal(t, " &lt;b&gt;x&lt;/b&gt; ", *escapePtr(&value))
	require.Nil(t, escapePtr(nil))
	require.Nil(t, formatOptionalTime(nil))
	now := time.Now()
	require.NotNil(t, formatOptionalTime(&now))
}

func TestUserHTTPEndpointsWithMockService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Now()
	profile := &usecase.ProfileDetails{
		ProfileID:     10,
		UserAccountID: 5,
		Username:      "neo",
		FirstName:     "Neo",
		LastName:      "Anderson",
		Gender:        model.Male,
		BirthdayDate:  time.Date(1999, 3, 31, 0, 0, 0, 0, time.UTC),
		IsOnline:      true,
		LastSeenAt:    &now,
	}
	cards := []usecase.UserCard{{ID: 10, FirstName: "Neo", LastName: "Anderson", Username: "neo", AvatarLink: "/avatar.png", IsOnline: true, LastSeenAt: &now}}
	events := []usecase.LatestEvent{{UserCard: cards[0], Type: 1}}
	friends := []model.Friend{friend(11, "trinity")}
	settings := &model.UserSettings{UserAccountID: 5, Language: model.LanguageRU, Theme: model.ThemeDark}

	svc := handlermocks.NewMockUserService(ctrl)
	svc.EXPECT().GetPublicPopularUsers(gomock.Any()).Return(cards, nil).AnyTimes()
	svc.EXPECT().GetUsersFriends(gomock.Any(), gomock.Any()).Return(friends, nil).AnyTimes()
	svc.EXPECT().GetSuggestedUsers(gomock.Any(), gomock.Any()).Return(cards, nil).AnyTimes()
	svc.EXPECT().GetLatestEvents(gomock.Any()).Return(events, nil).AnyTimes()
	svc.EXPECT().GetProfileMe(gomock.Any(), gomock.Any()).Return(profile, nil).AnyTimes()
	svc.EXPECT().GetProfileByID(gomock.Any(), gomock.Any()).Return(profile, nil).AnyTimes()
	svc.EXPECT().UpdateMe(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	svc.EXPECT().GetSettings(gomock.Any(), gomock.Any()).Return(settings, nil).AnyTimes()
	svc.EXPECT().UpdateSettings(gomock.Any(), gomock.Any(), gomock.Any()).Return(settings, nil).AnyTimes()
	svc.EXPECT().GetFriends(gomock.Any(), gomock.Any(), gomock.Any()).Return(friends, nil).AnyTimes()
	svc.EXPECT().DeleteFriend(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	svc.EXPECT().RequestFriendship(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	svc.EXPECT().GetIncomingFriendRequests(gomock.Any(), gomock.Any(), gomock.Any()).Return(friends, nil).AnyTimes()
	svc.EXPECT().GetOutgoingFriendRequests(gomock.Any(), gomock.Any(), gomock.Any()).Return(friends, nil).AnyTimes()
	svc.EXPECT().AcceptFriendRequest(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	svc.EXPECT().DeclineFriendRequest(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	svc.EXPECT().RevokeFriendRequest(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	router := chi.NewRouter()
	New(svc).RegisterRoutes(router, nil)

	cases := []struct {
		method string
		path   string
		body   any
		userID int64
	}{
		{http.MethodGet, "/public/popular-users", nil, 0},
		{http.MethodGet, "/users/10/friends", nil, 0},
		{http.MethodGet, "/users/suggested", nil, 5},
		{http.MethodGet, "/users/latest-events", nil, 5},
		{http.MethodGet, "/profile/me", nil, 5},
		{http.MethodGet, "/profile/10", nil, 5},
		{http.MethodPatch, "/profile/me/edit", map[string]any{"login": "neo2", "firstName": "Neo"}, 5},
		{http.MethodGet, "/settings/", nil, 5},
		{http.MethodPost, "/settings/", map[string]any{"language": "RU", "theme": "dark"}, 5},
		{http.MethodPost, "/friends/request", map[string]any{"friendID": 11}, 5},
		{http.MethodPost, "/friends/accept/11", nil, 5},
		{http.MethodPost, "/friends/decline/11", nil, 5},
		{http.MethodDelete, "/friends/request/11", nil, 5},
		{http.MethodGet, "/friends/requests/incoming", nil, 5},
		{http.MethodGet, "/friends/requests/incoming/pending", nil, 5},
		{http.MethodGet, "/friends/requests/outgoing", nil, 5},
		{http.MethodGet, "/friends/requests/outgoing/pending", nil, 5},
		{http.MethodDelete, "/friends/11", nil, 5},
		{http.MethodGet, "/friends/accepted", nil, 5},
		{http.MethodGet, "/friends/", nil, 5},
	}

	for _, tc := range cases {
		rr := serveUser(t, router, tc.method, tc.path, tc.body, tc.userID)
		if rr.Code == 0 || rr.Code >= 500 {
			t.Fatalf("%s %s returned %d: %s", tc.method, tc.path, rr.Code, rr.Body.String())
		}
	}
}

type userRepoMocks struct {
	accounts     *repomocks.MockAccountRepo
	profiles     *repomocks.MockProfileRepo
	userProfiles *repomocks.MockUserProfileRepo
	settings     *repomocks.MockSettingsRepo
	friendships  *repomocks.MockFriendshipRepo
}

func newUserRouter(t *testing.T) (*chi.Mux, userRepoMocks) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repos := userRepoMocks{
		accounts:     repomocks.NewMockAccountRepo(ctrl),
		profiles:     repomocks.NewMockProfileRepo(ctrl),
		userProfiles: repomocks.NewMockUserProfileRepo(ctrl),
		settings:     repomocks.NewMockSettingsRepo(ctrl),
		friendships:  repomocks.NewMockFriendshipRepo(ctrl),
	}
	store := repository.Store{
		Accounts:     repos.accounts,
		Profiles:     repos.profiles,
		UserProfiles: repos.userProfiles,
		Settings:     repos.settings,
		Friendships:  repos.friendships,
	}
	router := chi.NewRouter()
	New(usecase.New(store)).RegisterRoutes(router, nil)
	return router, repos
}

func serveUser(t *testing.T, router *chi.Mux, method, path string, body any, userID int64) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if userID > 0 {
		req = req.WithContext(context.WithValue(req.Context(), "user_id", userID))
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func expectCurrentUserProfile(repos userRepoMocks, accountID, profileID int64) {
	repos.profiles.EXPECT().GetByUserAccountID(gomock.Any(), accountID).Return(profile(profileID), nil)
}

func userAccount(id int64, username string) *model.UserAccount {
	return &model.UserAccount{ID: id, Uid: uuid.New(), Username: username, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

func profile(id int64) *model.Profile {
	return &model.Profile{ID: id, Uid: uuid.New(), IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

func userProfile(id, accountID, profileID int64) *model.UserProfile {
	return &model.UserProfile{ID: id, Uid: uuid.New(), UserAccountID: accountID, ProfileID: profileID, FirstName: "Neo", LastName: "Anderson", Gender: model.Male, BirthdayDate: time.Date(2010, 5, 27, 0, 0, 0, 0, time.UTC), IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

func friend(profileID int64, username string) model.Friend {
	now := time.Now()
	return model.Friend{ProfileID: profileID, FirstName: "Trinity", LastName: "Matrix", Username: username, Status: model.FriendshipAccepted, CreatedAt: now, UpdatedAt: now}
}

func stringPtr(value string) *string {
	return &value
}
