package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	models "github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/model"
	tickets "github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/usecase"
	ticketmocks "github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/usecase/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/xerrors"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestSupportHandlerRoutesSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		body   any
		setup  func(*ticketmocks.MockTicketService)
		status int
	}{
		{
			name:   "send ticket",
			method: http.MethodPost,
			path:   "/tickets",
			body: map[string]any{
				"category":    "bug",
				"title":       " Bug report ",
				"login":       " tester ",
				"description": "broken flow",
			},
			setup: func(svc *ticketmocks.MockTicketService) {
				email := "tester@example.com"
				expectCurrentProfile(svc)
				svc.EXPECT().GetUserAccountByProfileID(gomock.Any(), int64(10)).Return(&models.UserAccount{ID: 5, Email: &email}, nil)
				svc.EXPECT().Save(gomock.Any(), gomock.Any()).Return(int64(101), nil)
				expectMediaLookup(svc)
			},
			status: http.StatusCreated,
		},
		{
			name:   "my tickets",
			method: http.MethodGet,
			path:   "/tickets/my",
			setup: func(svc *ticketmocks.MockTicketService) {
				expectCurrentProfile(svc)
				svc.EXPECT().GetByProfileID(gomock.Any(), int64(10)).Return([]models.SupportTicket{*supportTicket(101, 10, 1)}, nil)
				expectMediaLookup(svc)
			},
			status: http.StatusOK,
		},
		{
			name:   "all tickets",
			method: http.MethodGet,
			path:   "/tickets?status=open&category=bug&line=1&assignedAgentId=42",
			setup: func(svc *ticketmocks.MockTicketService) {
				expectProfile(svc, models.SupportRoleAdmin)
				svc.EXPECT().GetAll(gomock.Any(), models.SupportRoleAdmin, gomock.Any()).Return([]models.SupportTicket{*supportTicket(101, 10, 1)}, nil)
				expectMediaLookup(svc)
			},
			status: http.StatusOK,
		},
		{
			name:   "get ticket",
			method: http.MethodGet,
			path:   "/tickets/101",
			setup: func(svc *ticketmocks.MockTicketService) {
				expectProfile(svc, models.SupportRoleUser)
				svc.EXPECT().CanAccessTicket(gomock.Any(), int64(101), int64(10), models.SupportRoleUser).Return(supportTicket(101, 10, 1), nil)
				expectMediaLookup(svc)
			},
			status: http.StatusOK,
		},
		{
			name:   "update ticket",
			method: http.MethodPatch,
			path:   "/tickets/101",
			body:   map[string]any{"title": "Updated", "description": "Updated description", "category": "question"},
			setup: func(svc *ticketmocks.MockTicketService) {
				expectCurrentProfile(svc)
				svc.EXPECT().Update(gomock.Any(), int64(101), int64(10), gomock.Any()).Return(supportTicket(101, 10, 1), nil)
				expectMediaLookup(svc)
			},
			status: http.StatusOK,
		},
		{
			name:   "update status",
			method: http.MethodPatch,
			path:   "/tickets/101/status",
			body:   map[string]any{"status": "in_progress"},
			setup: func(svc *ticketmocks.MockTicketService) {
				expectProfile(svc, models.SupportRoleSupportL1)
				svc.EXPECT().GetByIDForAgent(gomock.Any(), int64(101)).Return(supportTicket(101, 10, 1), nil)
				svc.EXPECT().UpdateStatusByAgent(gomock.Any(), int64(101), models.TicketStatusInProgress).Return(supportTicket(101, 10, 1), nil)
				expectMediaLookup(svc)
			},
			status: http.StatusOK,
		},
		{
			name:   "assign ticket",
			method: http.MethodPatch,
			path:   "/tickets/101/assign",
			body:   map[string]any{"agentId": "42"},
			setup: func(svc *ticketmocks.MockTicketService) {
				expectProfile(svc, models.SupportRoleAdmin)
				svc.EXPECT().GetProfileRole(gomock.Any(), int64(42)).Return(models.SupportRoleSupportL1, nil)
				svc.EXPECT().GetByIDForAgent(gomock.Any(), int64(101)).Return(supportTicket(101, 10, 1), nil)
				svc.EXPECT().Assign(gomock.Any(), int64(101), int64(42)).Return(supportTicket(101, 10, 1), nil)
				expectMediaLookup(svc)
			},
			status: http.StatusOK,
		},
		{
			name:   "escalate ticket",
			method: http.MethodPatch,
			path:   "/tickets/101/escalate",
			setup: func(svc *ticketmocks.MockTicketService) {
				expectProfile(svc, models.SupportRoleSupportL1)
				svc.EXPECT().GetByIDForAgent(gomock.Any(), int64(101)).Return(supportTicket(101, 10, 1), nil)
				svc.EXPECT().Escalate(gomock.Any(), int64(101)).Return(supportTicket(101, 10, 2), nil)
				expectMediaLookup(svc)
			},
			status: http.StatusOK,
		},
		{
			name:   "rate ticket",
			method: http.MethodPost,
			path:   "/tickets/101/rating",
			body:   map[string]any{"rating": 5},
			setup: func(svc *ticketmocks.MockTicketService) {
				expectCurrentProfile(svc)
				ticket := supportTicket(101, 10, 1)
				rating := 5
				ticket.Rating = &rating
				svc.EXPECT().Rate(gomock.Any(), int64(101), int64(10), 5).Return(ticket, nil)
			},
			status: http.StatusOK,
		},
		{
			name:   "stats",
			method: http.MethodGet,
			path:   "/stats",
			setup: func(svc *ticketmocks.MockTicketService) {
				expectProfile(svc, models.SupportRoleAdmin)
				svc.EXPECT().GetStats(gomock.Any()).Return(&models.SupportTicketStats{TotalCount: 3}, nil)
			},
			status: http.StatusOK,
		},
		{
			name:   "ticket messages",
			method: http.MethodGet,
			path:   "/tickets/101/messages",
			setup: func(svc *ticketmocks.MockTicketService) {
				expectProfile(svc, models.SupportRoleUser)
				svc.EXPECT().CanAccessTicket(gomock.Any(), int64(101), int64(10), models.SupportRoleUser).Return(supportTicket(101, 10, 1), nil)
				svc.EXPECT().GetMessages(gomock.Any(), int64(101)).Return([]models.SupportTicketMessage{*supportMessage(1, 101, 10)}, nil)
				svc.EXPECT().GetUserProfileByProfileID(gomock.Any(), int64(10)).Return(&models.UserProfile{FirstName: "Test", LastName: "User"}, nil)
			},
			status: http.StatusOK,
		},
		{
			name:   "send ticket message",
			method: http.MethodPost,
			path:   "/tickets/101/messages",
			body:   map[string]any{"text": "new answer"},
			setup: func(svc *ticketmocks.MockTicketService) {
				expectProfile(svc, models.SupportRoleUser)
				svc.EXPECT().CanAccessTicket(gomock.Any(), int64(101), int64(10), models.SupportRoleUser).Return(supportTicket(101, 10, 1), nil)
				svc.EXPECT().SaveMessage(gomock.Any(), int64(101), int64(10), models.SupportRoleUser, "new answer").Return(supportMessage(2, 101, 10), nil)
				svc.EXPECT().GetUserProfileByProfileID(gomock.Any(), int64(10)).Return(&models.UserProfile{FirstName: "Test", LastName: "User"}, nil)
			},
			status: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			svc := ticketmocks.NewMockTicketService(ctrl)
			tt.setup(svc)

			router := chi.NewRouter()
			NewSupportHandler(svc).RegisterRoutes(router)

			req := supportRequest(t, tt.method, tt.path, tt.body)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			require.Equal(t, tt.status, rr.Code)
			require.NotEmpty(t, rr.Body.String())
		})
	}
}

func TestSupportHandlerErrorsAndParsers(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	svc := ticketmocks.NewMockTicketService(ctrl)
	router := chi.NewRouter()
	NewSupportHandler(svc).RegisterRoutes(router)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/tickets/my", nil))
	require.Equal(t, http.StatusUnauthorized, rr.Code)

	req := supportRequest(t, http.MethodGet, "/tickets/0", nil)
	expectProfile(svc, models.SupportRoleUser)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)

	require.True(t, roleCanWorkWithTicket(models.SupportRoleAdmin, supportTicket(1, 1, 2)))
	require.True(t, roleCanWorkWithTicket(models.SupportRoleSupportL1, supportTicket(1, 1, 1)))
	require.False(t, roleCanWorkWithTicket(models.SupportRoleSupportL1, supportTicket(1, 1, 2)))
	require.Equal(t, "support:42", supportRoomID(42))
	require.Equal(t, "closed", ticketStatusToString(models.TicketStatusClosed))
	require.Equal(t, "open", ticketStatusToString(models.TicketStatus(99)))
	require.Equal(t, "other", ticketCategoryToString(models.TicketCategory(99)))

	status, err := parseTicketStatusString("waiting_user")
	require.NoError(t, err)
	require.Equal(t, models.TicketStatusWaitingUser, status)
	_, err = parseTicketStatusString("bad")
	require.Error(t, err)

	category, err := parseTicketCategoryString("feature_request")
	require.NoError(t, err)
	require.Equal(t, models.CategoryFeatureRequest, category)
	_, err = parseTicketCategoryString("bad")
	require.Error(t, err)

	require.Equal(t, xerrors.SupportTicketNotFound.Error(), xerrors.SupportTicketNotFound.Error())
	w := httptest.NewRecorder()
	handleTicketAccessError(w, nil, xerrors.SupportTicketNotFound, 1, 2)
	require.Equal(t, http.StatusNotFound, w.Code)
	w = httptest.NewRecorder()
	handleTicketAccessError(w, nil, tickets.ErrForbidden, 1, 2)
	require.Equal(t, http.StatusForbidden, w.Code)
	w = httptest.NewRecorder()
	handleTicketAccessError(w, nil, errors.New("boom"), 1, 2)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func supportRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()

	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_id", int64(5))
	return req.WithContext(ctx)
}

func expectCurrentProfile(svc *ticketmocks.MockTicketService) {
	svc.EXPECT().GetProfileByUserAccountID(gomock.Any(), int64(5)).Return(&models.Profile{ID: 10, IsActive: true}, nil)
}

func expectProfile(svc *ticketmocks.MockTicketService, role models.SupportRole) {
	expectCurrentProfile(svc)
	svc.EXPECT().GetProfileRole(gomock.Any(), int64(10)).Return(role, nil)
}

func expectMediaLookup(svc *ticketmocks.MockTicketService) {
	svc.EXPECT().GetMediasByTicketID(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
}

func supportTicket(id, profileID int64, line int) *models.SupportTicket {
	now := time.Now()
	return &models.SupportTicket{
		ID:          id,
		ProfileID:   profileID,
		Login:       "tester",
		Email:       "tester@example.com",
		Category:    models.CategoryBug,
		Title:       "Bug",
		Description: "Something is broken",
		Status:      models.TicketStatusOpen,
		Priority:    models.TicketPriorityLow,
		Line:        line,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func supportMessage(id, ticketID, authorID int64) *models.SupportTicketMessage {
	return &models.SupportTicketMessage{
		ID:         id,
		TicketID:   ticketID,
		AuthorID:   authorID,
		AuthorRole: models.SupportRoleUser,
		Text:       "message",
		CreatedAt:  time.Now(),
	}
}
