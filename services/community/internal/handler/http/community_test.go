package http

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/requestcontext"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	handlermocks "github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/handler/http/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/model"
	repositorymock "github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/repository/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/usecase"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

func TestHandlerList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repositorymock.NewMockCommunityRepo(ctrl)
	handler := New(usecase.New(repo, nil))

	avatarID := int64(12)
	repo.EXPECT().List(gomock.Any(), 5, 3).Return([]model.Community{
		{ID: 1, ProfileID: 10, Title: "<Title>", Username: "community", Type: model.PublicGroup, IsActive: true},
	}, nil)
	repo.EXPECT().GetAvatarID(gomock.Any(), int64(10)).Return(&avatarID, nil)

	rec := serveCommunity(handler, nil, http.MethodGet, "/communities?limit=5&offset=3", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("\\u0026lt;Title\\u0026gt;")) {
		t.Fatalf("expected escaped title in response, got %s", rec.Body.String())
	}
}

func TestHandlerGetWithOptionalUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repositorymock.NewMockCommunityRepo(ctrl)
	users := usermock.NewMockUserServiceClient(ctrl)
	handler := New(usecase.New(repo, users))

	userID := int64(7)
	role := model.Admin
	repo.EXPECT().Get(gomock.Any(), int64(2)).Return(&model.Community{ID: 2, ProfileID: 20, Title: "Title", Username: "community", Type: model.PrivateGroup, IsActive: true}, nil)
	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userID}).Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 70}, nil)
	repo.EXPECT().GetAvatarID(gomock.Any(), int64(20)).Return(nil, nil)
	repo.EXPECT().GetMember(gomock.Any(), int64(2), int64(70)).Return(&model.CommunityMember{Role: role, IsActive: true}, nil)

	rec := serveCommunity(handler, authUser(userID), http.MethodGet, "/communities/2", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"canEditCommunity":true`)) {
		t.Fatalf("expected permissions in response, got %s", rec.Body.String())
	}
}

func TestHandlerCreateRequiresUserContext(t *testing.T) {
	handler := New(usecase.New(nil, nil))

	rec := serveCommunity(handler, nil, http.MethodPost, "/communities", bytes.NewBufferString(`{"title":"Title","type":"public","username":"community"}`))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerCheckExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repositorymock.NewMockCommunityRepo(ctrl)
	handler := New(usecase.New(repo, nil))

	repo.EXPECT().ExistsByTitleOrUsername(gomock.Any(), "", "taken").Return(false, true, nil)
	rec := serveCommunity(handler, nil, http.MethodPost, "/communities/check-exists", bytes.NewBufferString(`{"title":"Taken","username":"TAKEN"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"usernameExists":true`)) {
		t.Fatalf("expected existence result, got %s", rec.Body.String())
	}
}

func TestHandlerListMembers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repositorymock.NewMockCommunityRepo(ctrl)
	users := usermock.NewMockUserServiceClient(ctrl)
	handler := New(usecase.New(repo, users))

	userID := int64(8)
	joinedAt := time.Date(2026, 5, 27, 10, 30, 0, 0, time.UTC)
	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userID}).Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 80}, nil)
	repo.EXPECT().GetMember(gomock.Any(), int64(3), int64(80)).Return(&model.CommunityMember{Role: model.Member, IsActive: true}, nil)
	repo.EXPECT().ListMembers(gomock.Any(), int64(3), false).Return([]model.CommunityMember{
		{MemberID: 80, Role: model.Member, JoinedAt: joinedAt, IsActive: true},
	}, nil)
	users.EXPECT().GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: 80}).Return(&userpb.GetProfileSummaryResponse{
		ProfileId: 80, UserAccountId: userID, FirstName: "<Ann>", LastName: "User", Username: "ann",
	}, nil)

	rec := serveCommunity(handler, authUser(userID), http.MethodGet, "/communities/3/members?includeBlocked=true", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("\\u0026lt;Ann\\u0026gt;")) {
		t.Fatalf("expected escaped member name, got %s", rec.Body.String())
	}
}

func TestCommunityHTTPEndpointsWithMockService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Now()
	role := model.Member
	details := &usecase.Details{
		Community: model.Community{
			ID:        1,
			Uid:       uuid.New(),
			ProfileID: 10,
			Title:     "Community",
			Username:  "community",
			Type:      model.PublicGroup,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		Membership: usecase.Membership{IsMember: true, Role: &role},
		Permission: usecase.Permissions{CanPost: true, CanPostAsMember: true},
	}
	member := &usecase.MemberDetails{
		ProfileID:     11,
		UserAccountID: 5,
		FirstName:     "Ann",
		LastName:      "User",
		Username:      "ann",
		Role:          model.Member,
		IsSelf:        true,
		JoinedAt:      now.Format(time.RFC3339),
	}

	svc := handlermocks.NewMockCommunityService(ctrl)
	svc.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(details, nil).AnyTimes()
	svc.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]usecase.Details{*details}, nil).AnyTimes()
	svc.EXPECT().GetDetails(gomock.Any(), gomock.Any(), gomock.Any()).Return(details, nil).AnyTimes()
	svc.EXPECT().GetDetailsByProfileID(gomock.Any(), gomock.Any(), gomock.Any()).Return(details, nil).AnyTimes()
	svc.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(details, nil).AnyTimes()
	svc.EXPECT().CheckExists(gomock.Any(), gomock.Any()).Return(&usecase.CheckExistsResult{Exists: true, UsernameExists: true, SuggestedUsername: "community1"}, nil).AnyTimes()
	svc.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	svc.EXPECT().ListMembers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]usecase.MemberDetails{*member}, nil).AnyTimes()
	svc.EXPECT().Join(gomock.Any(), gomock.Any(), gomock.Any()).Return(member, nil).AnyTimes()
	svc.EXPECT().Leave(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	svc.EXPECT().RemoveMember(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	svc.EXPECT().ChangeMemberRole(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(member, nil).AnyTimes()

	handler := New(svc)
	auth := authUser(5)
	body := func(raw string) *bytes.Buffer { return bytes.NewBufferString(raw) }
	cases := []struct {
		method string
		path   string
		body   *bytes.Buffer
	}{
		{http.MethodGet, "/communities?limit=5&offset=0", nil},
		{http.MethodPost, "/communities/check-exists", body(`{"title":"Community","username":"community"}`)},
		{http.MethodGet, "/communities/1", nil},
		{http.MethodGet, "/communities/by-profile/10", nil},
		{http.MethodGet, "/communities/1/members?includeBlocked=true", nil},
		{http.MethodPost, "/communities/1/join", nil},
		{http.MethodPost, "/communities/1/leave", nil},
		{http.MethodDelete, "/communities/1/members/11", nil},
		{http.MethodPatch, "/communities/1/members/11/role", body(`{"role":"moderator"}`)},
		{http.MethodPost, "/communities", body(`{"title":"Community","type":"public","username":"community"}`)},
		{http.MethodPatch, "/communities/1", body(`{"title":"Updated","removeAvatar":true}`)},
		{http.MethodDelete, "/communities/1", nil},
	}

	for _, tc := range cases {
		rec := serveCommunity(handler, auth, tc.method, tc.path, tc.body)
		if rec.Code == 0 || rec.Code >= 500 {
			t.Fatalf("%s %s returned %d: %s", tc.method, tc.path, rec.Code, rec.Body.String())
		}
	}
}

func TestHandlerHelpers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?limit=bad&flag=true&off=12", nil)
	if got := parseIntQuery(req, "limit", 20); got != 20 {
		t.Fatalf("parseIntQuery fallback = %d", got)
	}
	if got := parseIntQuery(req, "off", 0); got != 12 {
		t.Fatalf("parseIntQuery value = %d", got)
	}
	if !parseBoolQuery(req, "flag") || parseBoolQuery(req, "missing") {
		t.Fatal("unexpected parseBoolQuery result")
	}

	rec := httptest.NewRecorder()
	if id, ok := parseID(rec, "42"); !ok || id != 42 {
		t.Fatalf("parseID() = %d, %v", id, ok)
	}
	rec = httptest.NewRecorder()
	if _, ok := parseID(rec, "0"); ok || rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad id to write 400, got ok=%v code=%d", ok, rec.Code)
	}

	escaped := escapePtr(stringPtr("<bio>"))
	if escaped == nil || *escaped != "&lt;bio&gt;" {
		t.Fatalf("escapePtr() = %#v", escaped)
	}

	statusCases := map[error]int{
		usecase.ErrInvalidInput:        http.StatusBadRequest,
		usecase.ErrAlreadyExists:       http.StatusConflict,
		usecase.ErrCommunityNotFound:   http.StatusNotFound,
		usecase.ErrForbidden:           http.StatusForbidden,
		errors.New("unexpected error"): http.StatusInternalServerError,
	}
	for err, want := range statusCases {
		rec := httptest.NewRecorder()
		writeServiceError(rec, err)
		if rec.Code != want {
			t.Fatalf("writeServiceError(%v) = %d, want %d", err, rec.Code, want)
		}
	}
}

func serveCommunity(handler *Handler, authMiddleware func(http.Handler) http.Handler, method, target string, body *bytes.Buffer) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader(body.Bytes())
	}
	req := httptest.NewRequest(method, target, reader)
	rec := httptest.NewRecorder()
	router := chi.NewRouter()
	handler.RegisterRoutes(router, authMiddleware)
	router.ServeHTTP(rec, req)
	return rec
}

func authUser(userID int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(requestcontext.WithUserID(r.Context(), userID)))
		})
	}
}

func stringPtr(value string) *string {
	return &value
}
