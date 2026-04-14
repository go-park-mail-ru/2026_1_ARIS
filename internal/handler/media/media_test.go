package media

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	mock_service "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/mocks"
)

func contextWithUserID(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func TestMediaHandler_SaveFiles_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewMediaHandler(mockMediaSvc, mockSessionSvc, mockUserSvc)

	userID := int64(1)
	profileID := int64(100)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userID).
		Return(&models.Profile{ID: profileID}, nil)

	mockMediaSvc.EXPECT().
		Save(gomock.Any(), "test.txt", gomock.Any(), gomock.Any(), "post", profileID).
		Return(int64(123), "http://link", nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("files", "test.txt")
	part.Write([]byte("hello"))
	writer.Close()

	req := httptest.NewRequest("POST", "/media?for=post", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = req.WithContext(contextWithUserID(userID))

	w := httptest.NewRecorder()
	handler.SaveFiles(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp fileResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Files, 1)
	assert.Equal(t, int64(123), resp.Files[0].MediaID)
	assert.Equal(t, "http://link", resp.Files[0].MediaURL)
}

func TestMediaHandler_SaveFiles_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := NewMediaHandler(nil, nil, nil)

	req := httptest.NewRequest("POST", "/media?for=post", nil)
	w := httptest.NewRecorder()
	handler.SaveFiles(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMediaHandler_SaveFiles_MissingForParam(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := NewMediaHandler(nil, nil, mockUserSvc)

	userID := int64(1)
	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userID).
		Return(&models.Profile{ID: 100}, nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.CreateFormFile("files", "test.txt")
	writer.Close()

	req := httptest.NewRequest("POST", "/media", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = req.WithContext(contextWithUserID(userID))

	w := httptest.NewRecorder()
	handler.SaveFiles(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMediaHandler_SaveFiles_ProfileNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := NewMediaHandler(nil, nil, mockUserSvc)

	userID := int64(1)
	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userID).
		Return(nil, xerrors.ProfileNotFound)

	req := httptest.NewRequest("POST", "/media?for=post", nil)
	req = req.WithContext(contextWithUserID(userID))

	w := httptest.NewRecorder()
	handler.SaveFiles(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMediaHandler_SaveFiles_UnsupportedContentType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewMediaHandler(mockMediaSvc, nil, mockUserSvc)

	userID := int64(1)
	profileID := int64(100)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userID).
		Return(&models.Profile{ID: profileID}, nil)

	mockMediaSvc.EXPECT().
		Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "post", profileID).
		Return(int64(0), "", xerrors.UnsupportedContentType)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("files", "test.txt")
	part.Write([]byte("hello"))
	writer.Close()

	req := httptest.NewRequest("POST", "/media?for=post", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = req.WithContext(contextWithUserID(userID))

	w := httptest.NewRecorder()
	handler.SaveFiles(w, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
}
