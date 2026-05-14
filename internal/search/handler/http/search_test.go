package http

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/search/repository"
	searchmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/search/repository/mock"
	searchservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/search/service"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestSearchHandlerSearch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	searchRepo := searchmock.NewMockSearchRepo(ctrl)
	searchRepo.EXPECT().SearchUsers(gomock.Any(), "neo", 2).Return([]repository.UserResult{{
		ProfileID: 10, UserAccountID: 20, Username: "<neo>", FirstName: "<Neo>", LastName: "Anderson",
	}}, nil)
	searchRepo.EXPECT().SearchCommunities(gomock.Any(), "neo", 2).Return([]repository.CommunityResult{{
		ID: 1, ProfileID: 2, Username: "<team>", Title: "<Team>",
	}}, nil)

	handler := New(searchservice.New(repository.NewStore(searchRepo), nil))
	req := httptest.NewRequest(stdhttp.MethodGet, "/api/search?q=neo&limit=2", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.Search(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	var resp response
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp.Users, 1)
	require.Equal(t, "&lt;neo&gt;", resp.Users[0].Username)
	require.Len(t, resp.Communities, 1)
	require.Equal(t, "&lt;Team&gt;", resp.Communities[0].Title)
}

func TestSearchHandlerErrorsAndHelpers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := New(searchservice.New(repository.NewStore(searchmock.NewMockSearchRepo(ctrl)), nil))
	rec := httptest.NewRecorder()
	handler.Search(rec, httptest.NewRequest(stdhttp.MethodGet, "/api/search", nil))
	require.Equal(t, stdhttp.StatusBadRequest, rec.Code)

	require.Equal(t, 10, parseLimit(httptest.NewRequest(stdhttp.MethodGet, "/?limit=bad", nil), "limit", 10))
	require.Equal(t, 3, parseLimit(httptest.NewRequest(stdhttp.MethodGet, "/?limit=3", nil), "limit", 10))

	text := "<bio>"
	require.Nil(t, escapePtr(nil))
	require.Equal(t, "&lt;bio&gt;", *escapePtr(&text))
}
