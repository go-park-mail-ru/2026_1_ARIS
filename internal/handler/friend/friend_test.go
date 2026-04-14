package friend

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/dto"
	mock_service "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/mocks"
)

func contextWithUserID(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
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

	w := httptest.NewRecorder()
	handler.GetIncomingFriendRequests(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp friendsResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Friends, 1)
	assert.Equal(t, int64(500), resp.Friends[0].ProfileID)
}
