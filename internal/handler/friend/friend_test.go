package friend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/mocks"
	mock_service "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
)

func contextWithUserID(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func contextWithObservedLogger() (context.Context, *observer.ObservedLogs) {
	core, recorded := observer.New(zap.DebugLevel)
	l := zap.New(core)
	ctx := logger.WithLogger(context.Background(), l)
	return ctx, recorded
}

// Настройка роутера chi для тестов с параметрами
func setupChiContext(req *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

func TestFriendHandler_GetFriends_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockFriendSvc := mock_service.NewMockFriendshipService(ctrl)

	handler := NewFriendHandler(mockSessionSvc, mockUserSvc, mockFriendSvc)

	userID := int64(1)
	profileID := int64(100)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userID).
		Return(&models.Profile{ID: profileID}, nil)

	expectedFriends := []dto.FriendDTO{
		{ProfileID: 200, FirstName: "John", LastName: "Doe", Username: "johndoe"},
	}
	mockFriendSvc.EXPECT().
		GetFriends(gomock.Any(), profileID, models.FriendshipAccepted).
		Return(expectedFriends, nil)

	req := httptest.NewRequest("GET", "/friends/accepted", nil)
	req = req.WithContext(contextWithUserID(userID))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("status", "accepted")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.GetFriends(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp friendsResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Friends, 1)
	assert.Equal(t, int64(200), resp.Friends[0].ProfileID)
	assert.Equal(t, "John", resp.Friends[0].FirstName)
	assert.Equal(t, "johndoe", resp.Friends[0].Username)
}

func TestFriendHandler_GetFriends_InvalidStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := NewFriendHandler(nil, nil, nil)

	req := httptest.NewRequest("GET", "/friends/invalid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("status", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetFriends(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFriendHandler_GetFriends_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := NewFriendHandler(nil, nil, nil)

	req := httptest.NewRequest("GET", "/friends/accepted", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("status", "accepted")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetFriends(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestFriendHandler_GetUsersFriends_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockFriendSvc := mock_service.NewMockFriendshipService(ctrl)

	handler := NewFriendHandler(nil, mockUserSvc, mockFriendSvc)

	targetProfileID := int64(300)

	mockUserSvc.EXPECT().
		GetProfileByProfileID(gomock.Any(), targetProfileID).
		Return(&models.Profile{ID: targetProfileID}, nil)

	expectedFriends := []dto.FriendDTO{
		{ProfileID: 400, FirstName: "Alice", LastName: "Smith", Username: "alice"},
	}
	mockFriendSvc.EXPECT().
		GetFriends(gomock.Any(), targetProfileID, models.FriendshipAccepted).
		Return(expectedFriends, nil)

	req := httptest.NewRequest("GET", "/users/300/friends", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "300")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetUsersFriends(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp friendsResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Friends, 1)
	assert.Equal(t, int64(400), resp.Friends[0].ProfileID)
}

func TestFriendHandler_DeleteFriend_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockFriendSvc := mock_service.NewMockFriendshipService(ctrl)

	handler := NewFriendHandler(nil, mockUserSvc, mockFriendSvc)

	userID := int64(1)
	profileID := int64(100)
	friendProfileID := int64(200)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userID).
		Return(&models.Profile{ID: profileID}, nil)

	mockFriendSvc.EXPECT().
		DeleteFriend(gomock.Any(), profileID, friendProfileID).
		Return(nil)

	req := httptest.NewRequest("DELETE", "/friends/200", nil)
	req = req.WithContext(contextWithUserID(userID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userID", "200")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.DeleteFriend(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFriendHandler_RequestFriendship_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockFriendSvc := mock_service.NewMockFriendshipService(ctrl)

	handler := NewFriendHandler(nil, mockUserSvc, mockFriendSvc)

	userID := int64(1)
	profileID := int64(100)
	targetProfileID := int64(300)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userID).
		Return(&models.Profile{ID: profileID}, nil)

	mockFriendSvc.EXPECT().
		MakeFriends(gomock.Any(), profileID, targetProfileID).
		Return(nil)

	body := friendRequest{FriendID: targetProfileID}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/friends/request", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithUserID(userID))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.RequestFriendship(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFriendHandler_RequestFriendship_AlreadyExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockFriendSvc := mock_service.NewMockFriendshipService(ctrl)

	handler := NewFriendHandler(nil, mockUserSvc, mockFriendSvc)

	userID := int64(1)
	profileID := int64(100)
	targetProfileID := int64(300)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userID).
		Return(&models.Profile{ID: profileID}, nil)

	mockFriendSvc.EXPECT().
		MakeFriends(gomock.Any(), profileID, targetProfileID).
		Return(xerrors.AllreadyExists)

	body := friendRequest{FriendID: targetProfileID}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/friends/request", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithUserID(userID))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.RequestFriendship(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestFriendHandler_AcceptFriendRequest_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockFriendSvc := mock_service.NewMockFriendshipService(ctrl)

	handler := NewFriendHandler(nil, mockUserSvc, mockFriendSvc)

	userID := int64(1)
	profileID := int64(100)
	requesterProfileID := int64(200)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userID).
		Return(&models.Profile{ID: profileID}, nil)

	mockFriendSvc.EXPECT().
		AcceptFriendship(gomock.Any(), requesterProfileID, profileID).
		Return(nil)

	req := httptest.NewRequest("POST", "/friends/accept/200", nil)
	req = req.WithContext(contextWithUserID(userID))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("requesterID", "200")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.AcceptFriendRequest(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFriendHandler_DeclineFriendRequest_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockFriendSvc := mock_service.NewMockFriendshipService(ctrl)

	handler := NewFriendHandler(nil, mockUserSvc, mockFriendSvc)

	userID := int64(1)
	profileID := int64(100)
	requesterProfileID := int64(200)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userID).
		Return(&models.Profile{ID: profileID}, nil)

	mockFriendSvc.EXPECT().
		DeclineFriendship(gomock.Any(), requesterProfileID, profileID).
		Return(nil)

	req := httptest.NewRequest("POST", "/friends/decline/200", nil)
	req = req.WithContext(contextWithUserID(userID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("requesterID", "200")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.DeclineFriendRequest(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFriendHandler_RevokeFriendRequest_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockFriendSvc := mock_service.NewMockFriendshipService(ctrl)

	handler := NewFriendHandler(nil, mockUserSvc, mockFriendSvc)

	userID := int64(1)
	profileID := int64(100)
	addresseeProfileID := int64(300)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userID).
		Return(&models.Profile{ID: profileID}, nil)

	mockFriendSvc.EXPECT().
		RevokeFriendRequest(gomock.Any(), profileID, addresseeProfileID).
		Return(nil)

	req := httptest.NewRequest("DELETE", "/friends/request/300", nil)
	req = req.WithContext(contextWithUserID(userID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("addresseeID", "300")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.RevokeFriendRequest(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFriendHandler_GetIncomingFriendRequests_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockFriendSvc := mock_service.NewMockFriendshipService(ctrl)

	handler := NewFriendHandler(nil, mockUserSvc, mockFriendSvc)

	userID := int64(1)
	profileID := int64(100)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userID).
		Return(&models.Profile{ID: profileID}, nil)

	expectedFriends := []dto.FriendDTO{
		{ProfileID: 500, FirstName: "Bob", LastName: "Johnson", Username: "bob"},
	}
	mockFriendSvc.EXPECT().
		GetIncomingFriends(gomock.Any(), profileID, "pending").
		Return(expectedFriends, nil)

	req := httptest.NewRequest("GET", "/friends/requests/incoming/pending", nil)
	req = req.WithContext(contextWithUserID(userID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("status", "pending")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetIncomingFriendRequests(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp friendsResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Friends, 1)
	assert.Equal(t, int64(500), resp.Friends[0].ProfileID)
}

func TestGetOutgoingFriendRequests_EmptyStatus_DefaultsToPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mocks.NewMockUserService(ctrl)
	mockFriendshipSvc := mocks.NewMockFriendshipService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)

	handler := NewFriendHandler(mockSessionSvc, mockUserSvc, mockFriendshipSvc)

	ctx, observed := contextWithObservedLogger()
	ctx = context.WithValue(ctx, "user_id", int64(123))
	req := httptest.NewRequest(http.MethodGet, "/friends/outgoing", nil).WithContext(ctx)
	// Параметр status не устанавливаем - будет пустым
	req = setupChiContext(req, map[string]string{}) // пустая мапа, status не будет
	rec := httptest.NewRecorder()

	profile := &models.Profile{ID: 456}
	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), int64(123)).
		Return(profile, nil)

	expectedFriends := []dto.FriendDTO{{ProfileID: 1, FirstName: "33", LastName: "33", Username: "dh", Status: "pengind"}}
	mockFriendshipSvc.EXPECT().
		GetOutgoingFriends(gomock.Any(), profile.ID, string(models.FriendshipPending)).
		Return(expectedFriends, nil)

	handler.GetOutgoingFriendRequests(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp friendsResponse // предполагаем, что тип экспортирован
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, expectedFriends, resp.Friends)

	// Проверяем лог Info
	logs := observed.All()
	var found bool
	for _, entry := range logs {
		if entry.Message == "outgoing_friend_requests_returned" {
			found = true
			assert.Equal(t, zap.InfoLevel, entry.Level)
			assert.Equal(t, profile.ID, entry.ContextMap()["profile_id"])
			assert.Equal(t, string(models.FriendshipPending), entry.ContextMap()["status"])
			assert.Equal(t, int64(1), entry.ContextMap()["count"])
		}
	}
	assert.True(t, found, "info log not found")
}

func TestGetOutgoingFriendRequests_InvalidStatus_BadRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mocks.NewMockUserService(ctrl)
	mockFriendshipSvc := mocks.NewMockFriendshipService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)

	handler := NewFriendHandler(mockSessionSvc, mockUserSvc, mockFriendshipSvc)

	ctx, observed := contextWithObservedLogger()
	ctx = context.WithValue(ctx, "user_id", int64(123))
	req := httptest.NewRequest(http.MethodGet, "/friends/outgoing?status=invalid", nil).WithContext(ctx)
	req = setupChiContext(req, map[string]string{"status": "invalid"})
	rec := httptest.NewRecorder()

	handler.GetOutgoingFriendRequests(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Unknown status value")

	logs := observed.All()
	require.Equal(t, 1, len(logs))
	entry := logs[0]
	assert.Equal(t, zap.WarnLevel, entry.Level)
	assert.Equal(t, "cannot_get_outgoing_friend_requests_invalid_status", entry.Message)
	assert.Equal(t, "invalid", entry.ContextMap()["status"])
	assert.Equal(t, "/friends/outgoing", entry.ContextMap()["path"])
}

func TestGetOutgoingFriendRequests_MissingUserID_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mocks.NewMockUserService(ctrl)
	mockFriendshipSvc := mocks.NewMockFriendshipService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)

	handler := NewFriendHandler(mockSessionSvc, mockUserSvc, mockFriendshipSvc)

	ctx, observed := contextWithObservedLogger()
	// user_id не добавляем
	req := httptest.NewRequest(http.MethodGet, "/friends/outgoing", nil).WithContext(ctx)
	req = setupChiContext(req, map[string]string{"status": "pending"})
	rec := httptest.NewRecorder()

	handler.GetOutgoingFriendRequests(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), xerrors.InvalidCtxUserAccountValue)

	logs := observed.All()
	require.Equal(t, 1, len(logs))
	entry := logs[0]
	assert.Equal(t, zap.WarnLevel, entry.Level)
	assert.Equal(t, "cannot_get_outgoing_friend_requests_missing_user", entry.Message)
	assert.Equal(t, "/friends/outgoing", entry.ContextMap()["path"])
}

func TestGetOutgoingFriendRequests_ProfileNotFound_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mocks.NewMockUserService(ctrl)
	mockFriendshipSvc := mocks.NewMockFriendshipService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)

	handler := NewFriendHandler(mockSessionSvc, mockUserSvc, mockFriendshipSvc)

	ctx, observed := contextWithObservedLogger()
	ctx = context.WithValue(ctx, "user_id", int64(123))
	req := httptest.NewRequest(http.MethodGet, "/friends/outgoing", nil).WithContext(ctx)
	req = setupChiContext(req, map[string]string{"status": "accepted"})
	rec := httptest.NewRecorder()

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), int64(123)).
		Return(nil, xerrors.ProfileNotFound)

	handler.GetOutgoingFriendRequests(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "Profile not found")

	logs := observed.All()
	require.Equal(t, 1, len(logs))
	entry := logs[0]
	assert.Equal(t, zap.WarnLevel, entry.Level)
	assert.Equal(t, "cannot_get_outgoing_friend_requests_profile_not_found", entry.Message)
	assert.Equal(t, int64(123), entry.ContextMap()["userAccount_id"])
}

func TestGetOutgoingFriendRequests_GetProfileError_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mocks.NewMockUserService(ctrl)
	mockFriendshipSvc := mocks.NewMockFriendshipService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)

	handler := NewFriendHandler(mockSessionSvc, mockUserSvc, mockFriendshipSvc)

	ctx, observed := contextWithObservedLogger()
	ctx = context.WithValue(ctx, "user_id", int64(123))
	req := httptest.NewRequest(http.MethodGet, "/friends/outgoing", nil).WithContext(ctx)
	req = setupChiContext(req, map[string]string{"status": "pending"})
	rec := httptest.NewRecorder()

	dbErr := errors.New("db error")
	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), int64(123)).
		Return(nil, dbErr)

	handler.GetOutgoingFriendRequests(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), xerrors.InternalServerErrorStr)

	logs := observed.All()
	require.Equal(t, 1, len(logs))
	entry := logs[0]
	assert.Equal(t, zap.ErrorLevel, entry.Level)
	assert.Equal(t, "failed_to_get_profile", entry.Message)
	assert.Equal(t, int64(123), entry.ContextMap()["userAccount_id"])
	assert.Contains(t, entry.ContextMap()["error"], dbErr.Error())
}

func TestGetOutgoingFriendRequests_GetOutgoingFriendsError_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mocks.NewMockUserService(ctrl)
	mockFriendshipSvc := mocks.NewMockFriendshipService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)

	handler := NewFriendHandler(mockSessionSvc, mockUserSvc, mockFriendshipSvc)

	ctx, observed := contextWithObservedLogger()
	ctx = context.WithValue(ctx, "user_id", int64(123))
	req := httptest.NewRequest(http.MethodGet, "/friends/outgoing", nil).WithContext(ctx)
	req = setupChiContext(req, map[string]string{"status": "accepted"})
	rec := httptest.NewRecorder()

	profile := &models.Profile{ID: 456}
	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), int64(123)).
		Return(profile, nil)

	dbErr := errors.New("query failed")
	mockFriendshipSvc.EXPECT().
		GetOutgoingFriends(gomock.Any(), profile.ID, string(models.FriendshipAccepted)).
		Return(nil, dbErr)

	handler.GetOutgoingFriendRequests(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), xerrors.InternalServerErrorStr)

	logs := observed.All()
	var found bool
	for _, entry := range logs {
		if entry.Message == "failed_to_get_outgoing_friend_requests" {
			found = true
			assert.Equal(t, zap.ErrorLevel, entry.Level)
			assert.Equal(t, profile.ID, entry.ContextMap()["profile_id"])
			assert.Equal(t, string(models.FriendshipAccepted), entry.ContextMap()["status"])
			assert.Contains(t, entry.ContextMap()["error"], dbErr.Error())
		}
	}
	assert.True(t, found, "error log not found")
}

func TestGetOutgoingFriendRequests_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mocks.NewMockUserService(ctrl)
	mockFriendshipSvc := mocks.NewMockFriendshipService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)

	handler := NewFriendHandler(mockSessionSvc, mockUserSvc, mockFriendshipSvc)

	ctx, observed := contextWithObservedLogger()
	ctx = context.WithValue(ctx, "user_id", int64(123))
	req := httptest.NewRequest(http.MethodGet, "/friends/outgoing?status=pending", nil).WithContext(ctx)
	req = setupChiContext(req, map[string]string{"status": "pending"})
	rec := httptest.NewRecorder()

	profile := &models.Profile{ID: 456}
	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), int64(123)).
		Return(profile, nil)

	friends := []dto.FriendDTO{
		{ProfileID: 1, FirstName: "dd", LastName: "ed", Username: "h", Status: "penging"},
		{ProfileID: 2, FirstName: "dd", LastName: "ed", Username: "h", Status: "penging"},
	}
	mockFriendshipSvc.EXPECT().
		GetOutgoingFriends(gomock.Any(), profile.ID, string(models.FriendshipPending)).
		Return(friends, nil)

	handler.GetOutgoingFriendRequests(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp friendsResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, friends, resp.Friends)

	logs := observed.All()
	var found bool
	for _, entry := range logs {
		if entry.Message == "outgoing_friend_requests_returned" {
			found = true
			assert.Equal(t, zap.InfoLevel, entry.Level)
			assert.Equal(t, profile.ID, entry.ContextMap()["profile_id"])
			assert.Equal(t, string(models.FriendshipPending), entry.ContextMap()["status"])
			assert.Equal(t, int64(2), entry.ContextMap()["count"])
		}
	}
	assert.True(t, found)
}
