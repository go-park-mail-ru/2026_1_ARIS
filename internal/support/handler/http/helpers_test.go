package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	tickets "github.com/go-park-mail-ru/2026_1_ARIS/internal/support/service"
	"github.com/stretchr/testify/require"
)

func TestTicketParsingHelpers(t *testing.T) {
	status, err := parseTicketStatusString("in_progress")
	require.NoError(t, err)
	require.Equal(t, models.TicketStatusInProgress, status)
	status, err = parseTicketStatusString("2")
	require.NoError(t, err)
	require.Equal(t, models.TicketStatusWaitingUser, status)
	_, err = parseTicketStatusString("bad")
	require.Error(t, err)
	_, err = parseTicketStatusString("99")
	require.Error(t, err)

	status, err = parseTicketStatusJSON([]byte(`"closed"`))
	require.NoError(t, err)
	require.Equal(t, models.TicketStatusClosed, status)
	status, err = parseTicketStatusJSON([]byte(`1`))
	require.NoError(t, err)
	require.Equal(t, models.TicketStatusInProgress, status)
	_, err = parseTicketStatusJSON([]byte(`false`))
	require.Error(t, err)

	category, err := parseTicketCategoryString("feature_request")
	require.NoError(t, err)
	require.Equal(t, models.CategoryFeatureRequest, category)
	category, err = parseTicketCategoryString("3")
	require.NoError(t, err)
	require.Equal(t, models.CategoryQuestion, category)
	_, err = parseTicketCategoryString("bad")
	require.Error(t, err)
	_, err = parseTicketCategoryString("99")
	require.Error(t, err)

	category, err = parseTicketCategoryJSON([]byte(`"complaint"`))
	require.NoError(t, err)
	require.Equal(t, models.CotegoryComplaint, category)
	category, err = parseTicketCategoryJSON([]byte(`4`))
	require.NoError(t, err)
	require.Equal(t, models.CategoryOther, category)
	_, err = parseTicketCategoryJSON([]byte(`false`))
	require.Error(t, err)

	require.Equal(t, "open", ticketStatusToString(models.TicketStatusOpen))
	require.Equal(t, "in_progress", ticketStatusToString(models.TicketStatusInProgress))
	require.Equal(t, "waiting_user", ticketStatusToString(models.TicketStatusWaitingUser))
	require.Equal(t, "closed", ticketStatusToString(models.TicketStatusClosed))
	require.Equal(t, "open", ticketStatusToString(models.TicketStatus(99)))
	require.Equal(t, "bug", ticketCategoryToString(models.CategoryBug))
	require.Equal(t, "feature_request", ticketCategoryToString(models.CategoryFeatureRequest))
	require.Equal(t, "complaint", ticketCategoryToString(models.CotegoryComplaint))
	require.Equal(t, "question", ticketCategoryToString(models.CategoryQuestion))
	require.Equal(t, "other", ticketCategoryToString(models.CategoryOther))
	require.Equal(t, "other", ticketCategoryToString(models.TicketCategory(99)))
	require.True(t, isValidTicketStatus(models.TicketStatusClosed))
	require.False(t, isValidTicketStatus(models.TicketStatus(99)))
	require.True(t, isValidTicketCategory(models.CategoryOther))
	require.False(t, isValidTicketCategory(models.TicketCategory(99)))
}

func TestTicketRequestHelpers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?status=open&category=bug&line=1&assignedAgentId=10", nil)
	rec := httptest.NewRecorder()
	filter, ok := parseTicketFilter(rec, req)
	require.True(t, ok)
	require.Equal(t, models.TicketStatusOpen, *filter.Status)
	require.Equal(t, models.CategoryBug, *filter.Category)
	require.Equal(t, 1, *filter.Line)
	require.Equal(t, int64(10), *filter.AssignedAgentID)

	rec = httptest.NewRecorder()
	_, ok = parseTicketFilter(rec, httptest.NewRequest(http.MethodGet, "/?line=3", nil))
	require.False(t, ok)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("ticketID", "42")
	req = httptest.NewRequest(http.MethodGet, "/tickets/42", nil).WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx))
	rec = httptest.NewRecorder()
	id, ok := parseTicketID(rec, req)
	require.True(t, ok)
	require.Equal(t, int64(42), id)

	rec = httptest.NewRecorder()
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("ticketID", "bad")
	req = httptest.NewRequest(http.MethodGet, "/tickets/bad", nil).WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx))
	_, ok = parseTicketID(rec, req)
	require.False(t, ok)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	agent := int64(7)
	require.Equal(t, "7", *formatOptionalInt64(&agent))
	require.Nil(t, formatOptionalInt64(nil))
	value := "  hi  "
	require.Equal(t, "hi", *normalizeOptionalString(&value))
	require.Nil(t, normalizeOptionalString(nil))
	require.Equal(t, "support:42", supportRoomID(42))
	require.True(t, roleCanWorkWithTicket(models.SupportRoleAdmin, &models.SupportTicket{Line: 2}))
	require.True(t, roleCanWorkWithTicket(models.SupportRoleSupportL1, &models.SupportTicket{Line: 1}))
	require.False(t, roleCanWorkWithTicket(models.SupportRoleSupportL1, &models.SupportTicket{Line: 2}))
}

func TestTicketAccessErrorMapping(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code int
	}{
		{xerrors.SupportTicketNotFound, http.StatusNotFound},
		{tickets.ErrForbidden, http.StatusForbidden},
		{errors.New("boom"), http.StatusInternalServerError},
	} {
		rec := httptest.NewRecorder()
		handleTicketAccessError(rec, nil, tc.err, 1, 2)
		require.Equal(t, tc.code, rec.Code)
	}
}
