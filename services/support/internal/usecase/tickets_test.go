package usecase

import (
	"context"
	"errors"
	"testing"

	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	mediamock "github.com/go-park-mail-ru/2026_1_ARIS/proto/media/mock"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	models "github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/model"
	supportrepo "github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/repository"
	repomock "github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/repository/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/xerrors"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newTicketServiceWithMocks(t *testing.T) (*ticketService, *repomock.MockTicketRepository, *usermock.MockUserServiceClient, *mediamock.MockMediaServiceClient) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := repomock.NewMockTicketRepository(ctrl)
	users := usermock.NewMockUserServiceClient(ctrl)
	media := mediamock.NewMockMediaServiceClient(ctrl)
	service := NewTicketService(repo, Clients{User: users, Media: media}).(*ticketService)
	return service, repo, users, media
}

func supportTicket() *models.SupportTicket {
	return &models.SupportTicket{
		ID:          10,
		ProfileID:   20,
		Login:       "neo",
		Email:       "neo@example.com",
		Category:    models.CategoryBug,
		Title:       "bug",
		Description: "broken",
		Status:      models.TicketStatusOpen,
		Priority:    models.TicketPriorityLow,
		Line:        1,
	}
}

func TestTicketServiceSaveDefaultsAndDelegates(t *testing.T) {
	service, repo, _, _ := newTicketServiceWithMocks(t)
	ticket := &models.SupportTicket{ProfileID: 20, Login: "neo", Email: "neo@example.com"}
	repo.EXPECT().
		Save(gomock.Any(), ticket).
		DoAndReturn(func(_ context.Context, got *models.SupportTicket) (int64, error) {
			require.Equal(t, 1, got.Line)
			require.False(t, got.CreatedAt.IsZero())
			require.False(t, got.UpdatedAt.IsZero())
			return 99, nil
		})

	id, err := service.Save(context.Background(), ticket)

	require.NoError(t, err)
	require.Equal(t, int64(99), id)
}

func TestTicketServiceListFiltersByRole(t *testing.T) {
	service, repo, _, _ := newTicketServiceWithMocks(t)
	repo.EXPECT().
		GetAll(gomock.Any(), gomock.AssignableToTypeOf(supportrepo.TicketFilter{})).
		DoAndReturn(func(_ context.Context, filter supportrepo.TicketFilter) ([]models.SupportTicket, error) {
			require.NotNil(t, filter.Line)
			require.Equal(t, 1, *filter.Line)
			return []models.SupportTicket{*supportTicket()}, nil
		})

	tickets, err := service.GetAll(context.Background(), models.SupportRoleSupportL1, TicketFilter{})

	require.NoError(t, err)
	require.Len(t, tickets, 1)
}

func TestTicketServiceRejectsInvalidRoleForList(t *testing.T) {
	service, _, _, _ := newTicketServiceWithMocks(t)

	tickets, err := service.GetAll(context.Background(), models.SupportRoleUser, TicketFilter{})

	require.Nil(t, tickets)
	require.ErrorIs(t, err, ErrForbidden)
}

func TestTicketServiceUpdateAppliesFields(t *testing.T) {
	service, repo, _, _ := newTicketServiceWithMocks(t)
	title := "new title"
	description := "new description"
	category := models.CategoryQuestion
	ticket := supportTicket()
	repo.EXPECT().GetByIDAndProfileID(gomock.Any(), int64(10), int64(20)).Return(ticket, nil)
	repo.EXPECT().
		Update(gomock.Any(), ticket).
		DoAndReturn(func(_ context.Context, got *models.SupportTicket) (*models.SupportTicket, error) {
			require.Equal(t, title, got.Title)
			require.Equal(t, description, got.Description)
			require.Equal(t, category, got.Category)
			return got, nil
		})

	updated, err := service.Update(context.Background(), 10, 20, TicketUpdate{Title: &title, Description: &description, Category: &category})

	require.NoError(t, err)
	require.Equal(t, title, updated.Title)
}

func TestTicketServiceStatusRatingAndRoleValidation(t *testing.T) {
	service, repo, _, _ := newTicketServiceWithMocks(t)

	t.Run("invalid status", func(t *testing.T) {
		updated, err := service.UpdateStatus(context.Background(), 10, 20, models.TicketStatus(999))
		require.Nil(t, updated)
		require.ErrorIs(t, err, ErrInvalidTicketStatus)
	})

	t.Run("closed status sets closed at", func(t *testing.T) {
		repo.EXPECT().
			UpdateStatus(gomock.Any(), int64(10), int64(20), models.TicketStatusClosed, gomock.Not(nil), gomock.Any()).
			Return(supportTicket(), nil)

		updated, err := service.UpdateStatus(context.Background(), 10, 20, models.TicketStatusClosed)

		require.NoError(t, err)
		require.NotNil(t, updated)
	})

	t.Run("invalid rating", func(t *testing.T) {
		updated, err := service.Rate(context.Background(), 10, 20, 6)
		require.Nil(t, updated)
		require.ErrorIs(t, err, ErrInvalidRating)
	})

	t.Run("valid rating", func(t *testing.T) {
		rating := 5
		ticket := supportTicket()
		ticket.Rating = &rating
		repo.EXPECT().
			Rate(gomock.Any(), int64(10), int64(20), 5, gomock.Any()).
			Return(ticket, nil)

		updated, err := service.Rate(context.Background(), 10, 20, 5)

		require.NoError(t, err)
		require.Equal(t, 5, *updated.Rating)
	})

	t.Run("invalid support role", func(t *testing.T) {
		err := service.SetProfileRole(context.Background(), 20, models.SupportRoleUser)
		require.ErrorIs(t, err, ErrInvalidSupportRole)
	})
}

func TestTicketServiceRoleAndAccess(t *testing.T) {
	service, repo, _, _ := newTicketServiceWithMocks(t)

	repo.EXPECT().
		GetProfileRole(gomock.Any(), int64(20)).
		Return(&models.SupportProfileRole{ProfileID: 20, Role: models.SupportRoleAdmin}, nil)
	isAdmin, err := service.IsAdmin(context.Background(), 20)
	require.NoError(t, err)
	require.True(t, isAdmin)

	repo.EXPECT().
		GetProfileRole(gomock.Any(), int64(21)).
		Return(nil, xerrors.SupportForbidden)
	role, err := service.GetProfileRole(context.Background(), 21)
	require.NoError(t, err)
	require.Equal(t, models.SupportRoleUser, role)

	repo.EXPECT().GetByID(gomock.Any(), int64(10)).Return(&models.SupportTicket{ID: 10, ProfileID: 99}, nil)
	ticket, err := service.CanAccessTicket(context.Background(), 10, 20, models.SupportRoleSupportL2)
	require.NoError(t, err)
	require.Equal(t, int64(10), ticket.ID)

	repo.EXPECT().GetByID(gomock.Any(), int64(11)).Return(&models.SupportTicket{ID: 11, ProfileID: 99}, nil)
	ticket, err = service.CanAccessTicket(context.Background(), 11, 20, models.SupportRoleUser)
	require.Nil(t, ticket)
	require.ErrorIs(t, err, ErrForbidden)
}

func TestTicketServiceUserAndMediaClients(t *testing.T) {
	service, repo, users, media := newTicketServiceWithMocks(t)
	email := "neo@example.com"

	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: 7}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 20}, nil)
	profile, err := service.GetProfileByUserAccountID(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, int64(20), profile.ID)

	users.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: 20}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: 20, UserAccountId: 7, Username: "neo", FirstName: "Neo", LastName: "Anderson"}, nil)
	users.EXPECT().
		GetAuthUserByAccount(gomock.Any(), &userpb.GetAuthUserByAccountRequest{UserAccountId: 7}).
		Return(&userpb.AuthUserResponse{UserAccountId: 7, Login: "neo", Email: &email}, nil)
	account, err := service.GetUserAccountByProfileID(context.Background(), 20)
	require.NoError(t, err)
	require.Equal(t, email, *account.Email)

	users.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: 21}).
		Return(nil, status.Error(codes.NotFound, "missing"))
	userProfile, err := service.GetUserProfileByProfileID(context.Background(), 21)
	require.Nil(t, userProfile)
	require.ErrorIs(t, err, xerrors.UserProfileNotFound)

	media.EXPECT().
		GetMedia(gomock.Any(), &mediapb.GetMediaRequest{MediaId: 1}).
		Return(&mediapb.GetMediaResponse{MediaId: 1, MimeType: "image/png", Url: "/media/1.png"}, nil)
	repo.EXPECT().
		SaveMedia(gomock.Any(), models.TicketWithMedia{TicketID: 10, MediaID: 1, Order: 0}).
		Return(nil)
	errs := service.AttachMedia(context.Background(), 10, []MediaRef{{MediaID: 1}})
	require.Empty(t, errs.Errs)
}

func TestTicketServiceMessages(t *testing.T) {
	service, repo, _, _ := newTicketServiceWithMocks(t)

	repo.EXPECT().
		GetMessages(gomock.Any(), int64(10)).
		Return([]models.SupportTicketMessage{{ID: 1, Text: "hello"}}, nil)
	messages, err := service.GetMessages(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, messages, 1)

	repo.EXPECT().
		SaveMessage(gomock.Any(), gomock.AssignableToTypeOf(&models.SupportTicketMessage{})).
		DoAndReturn(func(_ context.Context, msg *models.SupportTicketMessage) (int64, error) {
			require.Equal(t, "answer", msg.Text)
			require.Equal(t, int64(20), msg.AuthorID)
			return 55, nil
		})
	message, err := service.SaveMessage(context.Background(), 10, 20, models.SupportRoleSupportL1, " answer ")
	require.NoError(t, err)
	require.Equal(t, "answer", message.Text)

	message, err = service.SaveMessage(context.Background(), 10, 20, models.SupportRoleSupportL1, " ")
	require.Nil(t, message)
	require.Error(t, err)
}

func TestTicketServiceSimpleRepositoryDelegates(t *testing.T) {
	service, repo, _, _ := newTicketServiceWithMocks(t)
	expectedErr := errors.New("repo")

	repo.EXPECT().GetByIDAndProfileID(gomock.Any(), int64(10), int64(20)).Return(nil, expectedErr)
	_, err := service.GetByID(context.Background(), 10, 20)
	require.ErrorIs(t, err, expectedErr)

	repo.EXPECT().GetByID(gomock.Any(), int64(10)).Return(supportTicket(), nil)
	ticket, err := service.GetByIDForAgent(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, int64(10), ticket.ID)

	repo.EXPECT().Assign(gomock.Any(), int64(10), int64(30), gomock.Any()).Return(supportTicket(), nil)
	ticket, err = service.Assign(context.Background(), 10, 30)
	require.NoError(t, err)
	require.Equal(t, int64(10), ticket.ID)

	repo.EXPECT().Escalate(gomock.Any(), int64(10), gomock.Any()).Return(supportTicket(), nil)
	ticket, err = service.Escalate(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, int64(10), ticket.ID)

	repo.EXPECT().GetStats(gomock.Any()).Return(&models.SupportTicketStats{TotalCount: 3}, nil)
	stats, err := service.GetStats(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(3), stats.TotalCount)
}
