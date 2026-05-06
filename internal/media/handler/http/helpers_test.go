package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	mediarepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/media/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/media/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	mediamock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/media/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestMediaMappingAndErrors(t *testing.T) {
	files := mapSavedFiles([]service.SavedFile{{ID: 1, URL: "url", Index: 2}})
	require.Len(t, files, 1)
	require.Equal(t, int64(1), files[0].MediaID)
	require.Equal(t, 2, files[0].Index)

	errs := mapFileErrors([]service.FileError{{Index: 3, Error: "bad"}})
	require.Len(t, errs, 1)
	require.Equal(t, "bad", errs[0].Error)

	for _, tc := range []struct {
		err  error
		code int
	}{
		{service.ErrInvalidInput, http.StatusBadRequest},
		{service.ErrMediaNotFound, http.StatusNotFound},
		{errors.New("boom"), http.StatusInternalServerError},
	} {
		rec := httptest.NewRecorder()
		writeServiceError(rec, tc.err)
		require.Equal(t, tc.code, rec.Code)
	}
}

func TestParseMediaID(t *testing.T) {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "10")
	req := httptest.NewRequest(http.MethodGet, "/media/10", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	id, ok := parseMediaID(rec, req)
	require.True(t, ok)
	require.Equal(t, int64(10), id)

	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")
	req = httptest.NewRequest(http.MethodGet, "/media/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec = httptest.NewRecorder()
	_, ok = parseMediaID(rec, req)
	require.False(t, ok)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestMediaHandlerEndpoints(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mediaStore := mediamock.NewMockMediaRepo(ctrl)
	s3Store := mediamock.NewMockS3Repo(ctrl)
	handler := New(service.New(mediarepo.NewStore(mediaStore, s3Store, ""), nil))
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	mediaStore.EXPECT().Get(gomock.Any(), int64(10)).Return(&models.Media{ID: 10, Link: "/media/file.png"}, nil).Times(2)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "10")
	req := httptest.NewRequest(http.MethodGet, "/media/10/url", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	handler.GetFileURL(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"mediaURL":"http://localhost:8080/media/file.png"`)

	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", "10")
	req = httptest.NewRequest(http.MethodGet, "/media/10", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec = httptest.NewRecorder()
	handler.RedirectToFile(rec, req)
	require.Equal(t, http.StatusTemporaryRedirect, rec.Code)
	require.Equal(t, "http://localhost:8080/media/file.png", rec.Header().Get("Location"))

	rec = httptest.NewRecorder()
	handler.SaveFiles(rec, httptest.NewRequest(http.MethodPost, "/media/upload", nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
