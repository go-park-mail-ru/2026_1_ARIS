package service

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	mediamock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/media/mock"
	supportrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/support/repository"
	supportmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/support/repository/mock"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	mediaservicemock "github.com/go-park-mail-ru/2026_1_ARIS/proto/media/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type supportMocks struct {
	tickets *supportmock.MockTicketRepository
	media   *mediamock.MockMediaRepo
	client  *mediaservicemock.MockMediaServiceClient
	service TicketService
}

func newSupportMocks(t *testing.T) (*gomock.Controller, supportMocks) {
	t.Helper()

	ctrl := gomock.NewController(t)
	m := supportMocks{
		tickets: supportmock.NewMockTicketRepository(ctrl),
		media:   mediamock.NewMockMediaRepo(ctrl),
		client:  mediaservicemock.NewMockMediaServiceClient(ctrl),
	}
	m.service = NewTicketService(m.tickets, m.media, m.client)
	return ctrl, m
}

func TestTicketServiceSaveAndReadMethods(t *testing.T) {
	ctrl, m := newSupportMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	ticket := &models.SupportTicket{ProfileID: 10, Login: "neo", Email: "neo@example.test", Title: "Help", Description: "Please"}
	m.tickets.EXPECT().Save(ctx, ticket).DoAndReturn(func(_ context.Context, got *models.SupportTicket) (int64, error) {
		require.False(t, got.CreatedAt.IsZero())
		require.False(t, got.UpdatedAt.IsZero())
		require.Equal(t, 1, got.Line)
		return int64(99), nil
	})
	id, err := m.service.Save(ctx, ticket)
	require.NoError(t, err)
	require.Equal(t, int64(99), id)

	m.tickets.EXPECT().GetByIDAndProfileID(ctx, int64(99), int64(10)).Return(ticket, nil)
	got, err := m.service.GetByID(ctx, 99, 10)
	require.NoError(t, err)
	require.Equal(t, ticket, got)

	m.tickets.EXPECT().GetByID(ctx, int64(99)).Return(ticket, nil)
	got, err = m.service.GetByIDForAgent(ctx, 99)
	require.NoError(t, err)
	require.Equal(t, ticket, got)

	m.tickets.EXPECT().GetByProfileID(ctx, int64(10)).Return([]models.SupportTicket{*ticket}, nil)
	list, err := m.service.GetByProfileID(ctx, 10)
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestTicketServiceMutations(t *testing.T) {
	ctrl, m := newSupportMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	ticket := &models.SupportTicket{ID: 99, ProfileID: 10, Status: models.TicketStatusOpen, Title: "Old", Description: "Old"}
	title := "New"
	description := "Fresh"
	category := models.CategoryQuestion

	m.tickets.EXPECT().GetByIDAndProfileID(ctx, int64(99), int64(10)).Return(ticket, nil)
	m.tickets.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, got *models.SupportTicket) (*models.SupportTicket, error) {
		require.Equal(t, title, got.Title)
		require.Equal(t, description, got.Description)
		require.Equal(t, category, got.Category)
		return got, nil
	})
	updated, err := m.service.Update(ctx, 99, 10, TicketUpdate{Title: &title, Description: &description, Category: &category})
	require.NoError(t, err)
	require.Equal(t, title, updated.Title)

	m.tickets.EXPECT().UpdateStatus(ctx, int64(99), int64(10), models.TicketStatusClosed, gomock.Not(nil), gomock.Any()).Return(ticket, nil)
	_, err = m.service.UpdateStatus(ctx, 99, 10, models.TicketStatusClosed)
	require.NoError(t, err)

	m.tickets.EXPECT().UpdateStatusByID(ctx, int64(99), models.TicketStatusInProgress, nil, gomock.Any()).Return(ticket, nil)
	_, err = m.service.UpdateStatusByAgent(ctx, 99, models.TicketStatusInProgress)
	require.NoError(t, err)

	m.tickets.EXPECT().Assign(ctx, int64(99), int64(7), gomock.Any()).Return(ticket, nil)
	_, err = m.service.Assign(ctx, 99, 7)
	require.NoError(t, err)

	m.tickets.EXPECT().Escalate(ctx, int64(99), gomock.Any()).Return(ticket, nil)
	_, err = m.service.Escalate(ctx, 99)
	require.NoError(t, err)

	m.tickets.EXPECT().Rate(ctx, int64(99), int64(10), 5, gomock.Any()).Return(ticket, nil)
	_, err = m.service.Rate(ctx, 99, 10, 5)
	require.NoError(t, err)
}

func TestTicketServiceValidation(t *testing.T) {
	_, m := newSupportMocks(t)
	ctx := context.Background()

	id, err := m.service.Save(ctx, nil)
	require.Zero(t, id)
	require.ErrorIs(t, err, ErrNilTicket)

	updated, err := m.service.Update(ctx, 1, 2, TicketUpdate{})
	require.Nil(t, updated)
	require.Error(t, err)

	updated, err = m.service.UpdateStatus(ctx, 1, 2, models.TicketStatus(100))
	require.Nil(t, updated)
	require.ErrorIs(t, err, ErrInvalidTicketStatus)

	updated, err = m.service.Rate(ctx, 1, 2, 6)
	require.Nil(t, updated)
	require.ErrorIs(t, err, ErrInvalidRating)

	require.ErrorIs(t, m.service.SetProfileRole(ctx, 1, models.SupportRoleUser), ErrInvalidSupportRole)
}

func TestTicketServiceRolesAndFilters(t *testing.T) {
	ctrl, m := newSupportMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	line := 1
	m.tickets.EXPECT().GetAll(ctx, supportrepo.TicketFilter{Line: &line}).Return([]models.SupportTicket{{ID: 1}}, nil)
	tickets, err := m.service.GetAll(ctx, models.SupportRoleSupportL1, TicketFilter{})
	require.NoError(t, err)
	require.Len(t, tickets, 1)

	_, err = m.service.GetAll(ctx, models.SupportRoleUser, TicketFilter{})
	require.ErrorIs(t, err, ErrForbidden)

	m.tickets.EXPECT().SetProfileRole(ctx, int64(10), models.SupportRoleAdmin).Return(nil)
	require.NoError(t, m.service.SetProfileRole(ctx, 10, models.SupportRoleAdmin))

	m.tickets.EXPECT().GetProfileRole(ctx, int64(10)).Return(&models.SupportProfileRole{ProfileID: 10, Role: models.SupportRoleSupportL2}, nil)
	role, err := m.service.GetProfileRole(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, models.SupportRoleSupportL2, role)

	m.tickets.EXPECT().GetProfileRole(ctx, int64(11)).Return(nil, xerrors.SupportForbidden)
	role, err = m.service.GetProfileRole(ctx, 11)
	require.NoError(t, err)
	require.Equal(t, models.SupportRoleUser, role)
}

func TestSupportRoleConvenienceHelpers(t *testing.T) {
	ctrl, m := newSupportMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	m.tickets.EXPECT().SetProfileRole(ctx, int64(10), models.SupportRoleAdmin).Return(nil)
	require.NoError(t, SetProfileRole(ctx, m.service, 10, models.SupportRoleAdmin))
	m.tickets.EXPECT().SetProfileRole(ctx, int64(11), models.SupportRoleAdmin).Return(nil)
	require.NoError(t, MakeProfileAdmin(ctx, m.service, 11))
	m.tickets.EXPECT().SetProfileRole(ctx, int64(12), models.SupportRoleSupportL1).Return(nil)
	require.NoError(t, MakeProfileSupportL1(ctx, m.service, 12))
	m.tickets.EXPECT().SetProfileRole(ctx, int64(13), models.SupportRoleSupportL2).Return(nil)
	require.NoError(t, MakeProfileSupportL2(ctx, m.service, 13))

	m.tickets.EXPECT().GetProfileRole(ctx, int64(11)).Return(&models.SupportProfileRole{Role: models.SupportRoleSupportL1}, nil)
	ok, err := m.service.IsSupportAgent(ctx, 11)
	require.NoError(t, err)
	require.True(t, ok)

	m.tickets.EXPECT().GetProfileRole(ctx, int64(12)).Return(&models.SupportProfileRole{Role: models.SupportRoleAdmin}, nil)
	ok, err = m.service.IsAdmin(ctx, 12)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestTicketAccessAndMedia(t *testing.T) {
	ctrl, m := newSupportMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	ticket := &models.SupportTicket{ID: 99, ProfileID: 10}
	m.tickets.EXPECT().GetByID(ctx, int64(99)).Return(ticket, nil)
	got, err := m.service.CanAccessTicket(ctx, 99, 10, models.SupportRoleUser)
	require.NoError(t, err)
	require.Equal(t, ticket, got)

	m.tickets.EXPECT().GetByID(ctx, int64(100)).Return(&models.SupportTicket{ID: 100, ProfileID: 1}, nil)
	got, err = m.service.CanAccessTicket(ctx, 100, 10, models.SupportRoleUser)
	require.Nil(t, got)
	require.ErrorIs(t, err, ErrForbidden)

	m.media.EXPECT().Get(ctx, int64(1)).Return(&models.Media{ID: 1, MimeType: "image/png"}, nil)
	m.tickets.EXPECT().SaveMedia(ctx, models.TicketWithMedia{TicketID: 99, MediaID: 1, Order: 0}).Return(nil)
	m.media.EXPECT().Get(ctx, int64(2)).Return(&models.Media{ID: 2, MimeType: "application/pdf"}, nil)
	m.media.EXPECT().Get(ctx, int64(3)).Return(nil, errors.New("missing"))

	mediaErrs := m.service.AttachMedia(ctx, 99, []MediaRef{{MediaID: 1}, {MediaID: 2}, {MediaID: 3}})
	require.Len(t, mediaErrs.Errs, 2)
	require.Equal(t, 1, mediaErrs.Errs[0].Pos)
	require.Equal(t, 2, mediaErrs.Errs[1].Pos)

	m.tickets.EXPECT().GetMediaByTicketID(ctx, int64(99)).Return([]int64{1, 2})
	m.media.EXPECT().Get(ctx, int64(1)).Return(&models.Media{ID: 1, MimeType: "image/png"}, nil)
	m.client.EXPECT().GetMediaURL(gomock.Any(), gomock.Any()).Return(&mediapb.GetMediaURLResponse{Url: " https://cdn.test/1.png "}, nil)
	m.media.EXPECT().Get(ctx, int64(2)).Return(nil, errors.New("missing"))

	medias := m.service.GetMediasByTicketID(ctx, 99)
	require.Len(t, medias, 1)
	require.Equal(t, "https://cdn.test/1.png", medias[0].Link)
}

func TestTicketMessagesAndStats(t *testing.T) {
	ctrl, m := newSupportMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	stats := &models.SupportTicketStats{TotalCount: 5}
	m.tickets.EXPECT().GetStats(ctx).Return(stats, nil)
	gotStats, err := m.service.GetStats(ctx)
	require.NoError(t, err)
	require.Equal(t, stats, gotStats)

	messages := []models.SupportTicketMessage{{ID: 1, TicketID: 99, Text: "hello"}}
	m.tickets.EXPECT().GetMessages(ctx, int64(99)).Return(messages, nil)
	gotMessages, err := m.service.GetMessages(ctx, 99)
	require.NoError(t, err)
	require.Equal(t, messages, gotMessages)

	m.tickets.EXPECT().SaveMessage(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, msg *models.SupportTicketMessage) (int64, error) {
		require.Equal(t, int64(99), msg.TicketID)
		require.Equal(t, int64(10), msg.AuthorID)
		require.Equal(t, models.SupportRoleUser, msg.AuthorRole)
		require.Equal(t, "hello", msg.Text)
		return int64(1), nil
	})
	msg, err := m.service.SaveMessage(ctx, 99, 10, models.SupportRoleUser, " hello ")
	require.NoError(t, err)
	require.Equal(t, "hello", msg.Text)

	msg, err = m.service.SaveMessage(ctx, 99, 10, models.SupportRoleUser, " ")
	require.Nil(t, msg)
	require.Error(t, err)
}

func TestRoleHelpers(t *testing.T) {
	require.True(t, isValidTicketStatus(models.TicketStatusOpen))
	require.True(t, isValidTicketStatus(models.TicketStatusClosed))
	require.False(t, isValidTicketStatus(models.TicketStatus(99)))
	require.True(t, isSupportRole(models.SupportRoleAdmin))
	require.True(t, isSupportRole(models.SupportRoleSupportL1))
	require.True(t, isSupportRole(models.SupportRoleSupportL2))
	require.False(t, isSupportRole(models.SupportRoleUser))
}
