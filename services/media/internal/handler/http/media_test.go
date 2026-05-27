package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/repository"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/repository/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/usecase"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMediaHTTPFileURLAndRedirect(t *testing.T) {
	t.Parallel()

	t.Run("url", func(t *testing.T) {
		t.Parallel()
		router, media := newMediaRouter(t)
		media.EXPECT().
			Get(gomock.Any(), int64(10)).
			Return(&model.Media{ID: 10, Uid: uuid.New(), Link: "/media/file.png"}, nil)

		rr := serveMedia(router, http.MethodGet, "/10/url")

		require.Equal(t, http.StatusOK, rr.Code)
		require.Contains(t, rr.Body.String(), `"mediaID":10`)
		require.Contains(t, rr.Body.String(), `arisnet.ru/media/file.png`)
	})

	t.Run("redirect", func(t *testing.T) {
		t.Parallel()
		router, media := newMediaRouter(t)
		media.EXPECT().
			Get(gomock.Any(), int64(10)).
			Return(&model.Media{ID: 10, Uid: uuid.New(), Link: "https://cdn.test/file.png"}, nil)

		rr := serveMedia(router, http.MethodGet, "/10")

		require.Equal(t, http.StatusTemporaryRedirect, rr.Code)
		require.Equal(t, "https://cdn.test/file.png", rr.Header().Get("Location"))
	})
}

func TestMediaHTTPErrorsAndMapping(t *testing.T) {
	t.Parallel()

	router, media := newMediaRouter(t)

	rr := serveMedia(router, http.MethodGet, "/bad/url")
	require.Equal(t, http.StatusBadRequest, rr.Code)

	media.EXPECT().Get(gomock.Any(), int64(404)).Return(nil, repository.ErrMediaNotFound)
	rr = serveMedia(router, http.MethodGet, "/404/url")
	require.Equal(t, http.StatusNotFound, rr.Code)

	files := mapSavedFiles([]usecase.SavedFile{{Index: 1, ID: 2, URL: "/m/2.png"}})
	require.Len(t, files, 1)
	require.Equal(t, int64(2), files[0].MediaID)
	errs := mapFileErrors([]usecase.FileError{{Index: 3, Error: "bad"}})
	require.Len(t, errs, 1)
	require.Equal(t, "bad", errs[0].Error)
}

func newMediaRouter(t *testing.T) (*chi.Mux, *repomocks.MockMediaRepo) {
	t.Helper()
	ctrl := gomock.NewController(t)
	media := repomocks.NewMockMediaRepo(ctrl)
	svc := usecase.New(repository.Store{Media: media}, userpb.NewUserServiceClient(nil))
	router := chi.NewRouter()
	New(svc).RegisterRoutes(router)
	return router, media
}

func serveMedia(router *chi.Mux, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}
