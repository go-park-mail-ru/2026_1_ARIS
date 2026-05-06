package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	tickets "github.com/go-park-mail-ru/2026_1_ARIS/internal/support/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type supportFakeSessionService struct {
	session *models.Session
	err     error
}

func (s *supportFakeSessionService) Create(context.Context, int64) (*models.Session, error) {
	return s.session, s.err
}

func (s *supportFakeSessionService) Get(context.Context, models.SessionID) (*models.Session, error) {
	return s.session, s.err
}

func (s *supportFakeSessionService) Delete(context.Context, models.SessionID) error {
	return s.err
}

type supportFakeUserService struct {
	profileByAccount map[int64]*models.Profile
	accountByProfile map[int64]*models.UserAccount
	userByProfile    map[int64]*models.UserProfile
	err              error
}

func (s *supportFakeUserService) GetProfileByUserAccountID(_ context.Context, userAccountID int64) (*models.Profile, error) {
	if s.err != nil {
		return nil, s.err
	}
	if profile := s.profileByAccount[userAccountID]; profile != nil {
		return profile, nil
	}
	return nil, xerrors.ProfileNotFound
}

func (s *supportFakeUserService) GetUserAccountByProfileID(_ context.Context, profileID int64) (*models.UserAccount, error) {
	if s.err != nil {
		return nil, s.err
	}
	if account := s.accountByProfile[profileID]; account != nil {
		return account, nil
	}
	return nil, xerrors.UserAccountNotFound
}

func (s *supportFakeUserService) GetUserProfileByProfileID(_ context.Context, profileID int64) (*models.UserProfile, error) {
	if profile := s.userByProfile[profileID]; profile != nil {
		return profile, nil
	}
	return nil, xerrors.ProfileNotFound
}

type supportFakeTicketService struct {
	ticket     *models.SupportTicket
	tickets    []models.SupportTicket
	medias     []models.Media
	messages   []models.SupportTicketMessage
	stats      *models.SupportTicketStats
	roles      map[int64]models.SupportRole
	attachErrs tickets.MediaErrors
	err        error
	lastFilter tickets.TicketFilter
}

func (s *supportFakeTicketService) Save(_ context.Context, ticket *models.SupportTicket) (int64, error) {
	if s.err != nil {
		return 0, s.err
	}
	ticket.ID = s.ticket.ID
	s.ticket = ticket
	return ticket.ID, nil
}

func (s *supportFakeTicketService) GetByID(context.Context, int64, int64) (*models.SupportTicket, error) {
	return s.ticketOrErr()
}

func (s *supportFakeTicketService) GetByIDForAgent(context.Context, int64) (*models.SupportTicket, error) {
	return s.ticketOrErr()
}

func (s *supportFakeTicketService) GetByProfileID(context.Context, int64) ([]models.SupportTicket, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.tickets, nil
}

func (s *supportFakeTicketService) GetAll(_ context.Context, _ models.SupportRole, filter tickets.TicketFilter) ([]models.SupportTicket, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.lastFilter = filter
	return s.tickets, nil
}

func (s *supportFakeTicketService) Update(_ context.Context, _ int64, _ int64, upd tickets.TicketUpdate) (*models.SupportTicket, error) {
	if s.err != nil {
		return nil, s.err
	}
	updated := *s.ticket
	if upd.Title != nil {
		updated.Title = *upd.Title
	}
	if upd.Description != nil {
		updated.Description = *upd.Description
	}
	if upd.Category != nil {
		updated.Category = *upd.Category
	}
	s.ticket = &updated
	return s.ticket, nil
}

func (s *supportFakeTicketService) UpdateStatus(context.Context, int64, int64, models.TicketStatus) (*models.SupportTicket, error) {
	return s.ticketOrErr()
}

func (s *supportFakeTicketService) UpdateStatusByAgent(_ context.Context, _ int64, status models.TicketStatus) (*models.SupportTicket, error) {
	if s.err != nil {
		return nil, s.err
	}
	updated := *s.ticket
	updated.Status = status
	s.ticket = &updated
	return s.ticket, nil
}

func (s *supportFakeTicketService) Assign(_ context.Context, _ int64, agentID int64) (*models.SupportTicket, error) {
	if s.err != nil {
		return nil, s.err
	}
	updated := *s.ticket
	updated.AssignedAgentID = &agentID
	s.ticket = &updated
	return s.ticket, nil
}

func (s *supportFakeTicketService) Escalate(context.Context, int64) (*models.SupportTicket, error) {
	if s.err != nil {
		return nil, s.err
	}
	updated := *s.ticket
	updated.Line = 2
	s.ticket = &updated
	return s.ticket, nil
}

func (s *supportFakeTicketService) Rate(_ context.Context, _ int64, _ int64, rating int) (*models.SupportTicket, error) {
	if s.err != nil {
		return nil, s.err
	}
	updated := *s.ticket
	updated.Rating = &rating
	s.ticket = &updated
	return s.ticket, nil
}

func (s *supportFakeTicketService) GetStats(context.Context) (*models.SupportTicketStats, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.stats, nil
}

func (s *supportFakeTicketService) SetProfileRole(context.Context, int64, models.SupportRole) error {
	return s.err
}

func (s *supportFakeTicketService) GetProfileRole(_ context.Context, profileID int64) (models.SupportRole, error) {
	if s.err != nil {
		return models.SupportRoleUser, s.err
	}
	if role, ok := s.roles[profileID]; ok {
		return role, nil
	}
	return models.SupportRoleUser, nil
}

func (s *supportFakeTicketService) IsSupportAgent(context.Context, int64) (bool, error) {
	return true, s.err
}

func (s *supportFakeTicketService) IsAdmin(context.Context, int64) (bool, error) {
	return true, s.err
}

func (s *supportFakeTicketService) CanAccessTicket(_ context.Context, _ int64, profileID int64, role models.SupportRole) (*models.SupportTicket, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.ticket == nil {
		return nil, xerrors.SupportTicketNotFound
	}
	if profileID == s.ticket.ProfileID || role != models.SupportRoleUser {
		return s.ticket, nil
	}
	return nil, tickets.ErrForbidden
}

func (s *supportFakeTicketService) AttachMedia(context.Context, int64, []tickets.MediaRef) tickets.MediaErrors {
	return s.attachErrs
}

func (s *supportFakeTicketService) GetMediasByTicketID(context.Context, int64) []models.Media {
	return s.medias
}

func (s *supportFakeTicketService) GetMessages(context.Context, int64) ([]models.SupportTicketMessage, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.messages, nil
}

func (s *supportFakeTicketService) SaveMessage(_ context.Context, ticketID, authorID int64, authorRole models.SupportRole, text string) (*models.SupportTicketMessage, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &models.SupportTicketMessage{ID: 90, TicketID: ticketID, AuthorID: authorID, AuthorRole: authorRole, Text: text, CreatedAt: time.Now()}, nil
}

func (s *supportFakeTicketService) ticketOrErr() (*models.SupportTicket, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.ticket == nil {
		return nil, xerrors.SupportTicketNotFound
	}
	return s.ticket, nil
}

func newSupportHandlerFixture() (*SupportHandler, *supportFakeTicketService) {
	email := "user@example.com"
	ticket := &models.SupportTicket{
		ID:          42,
		ProfileID:   100,
		Login:       "support-user",
		Email:       email,
		Category:    models.CategoryBug,
		Title:       "Broken flow",
		Description: "Something failed",
		Status:      models.TicketStatusOpen,
		Priority:    models.TicketPriorityLow,
		Line:        1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	ticketService := &supportFakeTicketService{
		ticket:  ticket,
		tickets: []models.SupportTicket{*ticket},
		medias:  []models.Media{{ID: 7, Link: "/media/7"}},
		messages: []models.SupportTicketMessage{{
			ID:         81,
			TicketID:   ticket.ID,
			Text:       "hello",
			AuthorID:   100,
			AuthorRole: models.SupportRoleUser,
			CreatedAt:  time.Now(),
		}},
		stats: &models.SupportTicketStats{TotalCount: 1, OpenCount: 1, ByCategory: map[string]int64{"bug": 1}},
		roles: map[int64]models.SupportRole{
			100: models.SupportRoleAdmin,
			200: models.SupportRoleSupportL1,
		},
	}
	userService := &supportFakeUserService{
		profileByAccount: map[int64]*models.Profile{10: {ID: 100}, 20: {ID: 200}},
		accountByProfile: map[int64]*models.UserAccount{100: {ID: 10, Email: &email}},
		userByProfile: map[int64]*models.UserProfile{
			100: {ProfileID: 100, FirstName: "Ada", LastName: "Lovelace"},
			200: {ProfileID: 200, FirstName: "Grace", LastName: "Hopper"},
		},
	}
	sessionService := &supportFakeSessionService{session: &models.Session{SessionID: "sid", UserID: 10}}
	return NewSupportHandler(sessionService, userService, ticketService), ticketService
}

func supportRequest(method, target, body string, ticketID string, userAccountID int64) *stdhttp.Request {
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))
	if userAccountID > 0 {
		req = req.WithContext(context.WithValue(req.Context(), "user_id", userAccountID))
	}
	if ticketID != "" {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("ticketID", ticketID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}
	return req
}

func TestSupportHandlerBuildsResponsesAndProfile(t *testing.T) {
	handler, ticketService := newSupportHandlerFixture()
	req := supportRequest(stdhttp.MethodGet, "/support/tickets/42", "", "42", 10)

	ticketResponse := handler.buildSupportTicketResponse(req, ticketService.ticket)
	require.Equal(t, "42", ticketResponse.ID)
	require.Equal(t, "SUP-42", ticketResponse.UID)
	require.Equal(t, "bug", ticketResponse.Category)
	require.Len(t, ticketResponse.Media, 1)
	require.Equal(t, int64(7), ticketResponse.Media[0].MediaID)
	require.Len(t, handler.buildSupportTicketResponses(req, ticketService.tickets), 1)

	messageResponse := handler.buildMessageResponse(req, ticketService.messages[0])
	require.Equal(t, "81", messageResponse.ID)
	require.Equal(t, "Ada Lovelace", messageResponse.AuthorName)
	require.Len(t, handler.buildMessageResponses(req, ticketService.messages), 1)

	rec := httptest.NewRecorder()
	profile, ok := handler.getCurrentProfile(rec, supportRequest(stdhttp.MethodGet, "/", "", "", 0))
	require.False(t, ok)
	require.Nil(t, profile)
	require.Equal(t, stdhttp.StatusUnauthorized, rec.Code)

	req = supportRequest(stdhttp.MethodGet, "/", "", "", 0)
	req.AddCookie(&stdhttp.Cookie{Name: "session_id", Value: "sid"})
	rec = httptest.NewRecorder()
	profile, ok = handler.getCurrentProfile(rec, req)
	require.True(t, ok)
	require.Equal(t, int64(100), profile.ID)
}

func TestSupportHandlerTicketLifecycle(t *testing.T) {
	handler, ticketService := newSupportHandlerFixture()

	rec := httptest.NewRecorder()
	handler.SendTicket(rec, supportRequest(stdhttp.MethodPost, "/support/tickets", `{"category":"bug","title":" Login fails ","login":" user ","description":" Details ","media":[{"mediaID":7,"mediaURL":"/media/7"}]}`, "", 10))
	require.Equal(t, stdhttp.StatusCreated, rec.Code)
	var created SupportTicketResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	require.Equal(t, "42", created.ID)
	require.Equal(t, "user@example.com", created.Email)

	rec = httptest.NewRecorder()
	handler.GetMyTickets(rec, supportRequest(stdhttp.MethodGet, "/support/tickets/my", "", "", 10))
	require.Equal(t, stdhttp.StatusOK, rec.Code)
	var mine []SupportTicketResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &mine))
	require.Len(t, mine, 1)

	rec = httptest.NewRecorder()
	handler.GetAllTickets(rec, supportRequest(stdhttp.MethodGet, "/support/tickets?status=open&category=bug&line=1&assigned_agent_id=200", "", "", 10))
	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Equal(t, models.TicketStatusOpen, *ticketService.lastFilter.Status)
	require.Equal(t, int64(200), *ticketService.lastFilter.AssignedAgentID)

	rec = httptest.NewRecorder()
	handler.GetTicket(rec, supportRequest(stdhttp.MethodGet, "/support/tickets/42", "", "42", 10))
	require.Equal(t, stdhttp.StatusOK, rec.Code)

	newTitle := "Updated title"
	rec = httptest.NewRecorder()
	handler.UpdateTicket(rec, supportRequest(stdhttp.MethodPatch, "/support/tickets/42", `{"title":"`+newTitle+`"}`, "42", 10))
	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Equal(t, newTitle, ticketService.ticket.Title)

	rec = httptest.NewRecorder()
	handler.UpdateTicketStatus(rec, supportRequest(stdhttp.MethodPatch, "/support/tickets/42/status", `{"status":"in_progress"}`, "42", 10))
	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Equal(t, models.TicketStatusInProgress, ticketService.ticket.Status)

	rec = httptest.NewRecorder()
	handler.AssignTicket(rec, supportRequest(stdhttp.MethodPatch, "/support/tickets/42/assign", `{"agentId":"200"}`, "42", 10))
	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.NotNil(t, ticketService.ticket.AssignedAgentID)
	require.Equal(t, int64(200), *ticketService.ticket.AssignedAgentID)

	ticketService.ticket.Line = 1
	rec = httptest.NewRecorder()
	handler.EscalateTicket(rec, supportRequest(stdhttp.MethodPatch, "/support/tickets/42/escalate", `{}`, "42", 10))
	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Equal(t, 2, ticketService.ticket.Line)

	rec = httptest.NewRecorder()
	handler.RateTicket(rec, supportRequest(stdhttp.MethodPost, "/support/tickets/42/rating", `{"rating":5}`, "42", 10))
	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"rating":5`)
}

func TestSupportHandlerMessagesStatsAndRoutes(t *testing.T) {
	handler, _ := newSupportHandlerFixture()

	router := chi.NewRouter()
	handler.RegisterRoutes(router)
	require.NotNil(t, router)

	rec := httptest.NewRecorder()
	handler.GetStats(rec, supportRequest(stdhttp.MethodGet, "/support/stats", "", "", 10))
	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"total":1`)

	rec = httptest.NewRecorder()
	handler.GetTicketMessages(rec, supportRequest(stdhttp.MethodGet, "/support/tickets/42/messages", "", "42", 10))
	require.Equal(t, stdhttp.StatusOK, rec.Code)
	var messages []SupportMessageResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &messages))
	require.Len(t, messages, 1)
	require.Equal(t, "Ada Lovelace", messages[0].AuthorName)

	rec = httptest.NewRecorder()
	handler.SendTicketMessage(rec, supportRequest(stdhttp.MethodPost, "/support/tickets/42/messages", `{"text":"new answer"}`, "42", 10))
	require.Equal(t, stdhttp.StatusCreated, rec.Code)
	require.Contains(t, rec.Body.String(), `"text":"new answer"`)

	rec = httptest.NewRecorder()
	handler.SendTicketMessage(rec, supportRequest(stdhttp.MethodPost, "/support/tickets/42/messages", `{"text":"   "}`, "42", 10))
	require.Equal(t, stdhttp.StatusBadRequest, rec.Code)
}

func TestSupportHandlerValidationAndAccessFailures(t *testing.T) {
	handler, ticketService := newSupportHandlerFixture()

	rec := httptest.NewRecorder()
	handler.SendTicket(rec, supportRequest(stdhttp.MethodPost, "/support/tickets", `{"category":"bug","title":"","login":"user","description":"details"}`, "", 10))
	require.Equal(t, stdhttp.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	handler.UpdateTicket(rec, supportRequest(stdhttp.MethodPatch, "/support/tickets/42", `{}`, "42", 10))
	require.Equal(t, stdhttp.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	handler.UpdateTicketStatus(rec, supportRequest(stdhttp.MethodPatch, "/support/tickets/42/status", `{}`, "42", 10))
	require.Equal(t, stdhttp.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	handler.AssignTicket(rec, supportRequest(stdhttp.MethodPatch, "/support/tickets/42/assign", `{"agentId":"bad"}`, "42", 10))
	require.Equal(t, stdhttp.StatusBadRequest, rec.Code)

	ticketService.roles[100] = models.SupportRoleSupportL2
	rec = httptest.NewRecorder()
	handler.EscalateTicket(rec, supportRequest(stdhttp.MethodPatch, "/support/tickets/42/escalate", `{}`, "42", 10))
	require.Equal(t, stdhttp.StatusForbidden, rec.Code)

	ticketService.roles[100] = models.SupportRoleAdmin
	ticketService.err = tickets.ErrInvalidRating
	rec = httptest.NewRecorder()
	handler.RateTicket(rec, supportRequest(stdhttp.MethodPost, "/support/tickets/42/rating", `{"rating":9}`, "42", 10))
	require.Equal(t, stdhttp.StatusBadRequest, rec.Code)

	ticketService.err = errors.New("storage is down")
	rec = httptest.NewRecorder()
	handler.GetAllTickets(rec, supportRequest(stdhttp.MethodGet, "/support/tickets", "", "", 10))
	require.Equal(t, stdhttp.StatusInternalServerError, rec.Code)
}
