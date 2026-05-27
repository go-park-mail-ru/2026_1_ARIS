package usecase

//go:generate mockgen -destination=mocks/support_service_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/usecase TicketService

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	authpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/auth"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	models "github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/model"
	supportrepo "github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/xerrors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrNilTicket = errors.New("support ticket is nil")
var ErrInvalidInput = errors.New("invalid input")
var ErrInvalidTicketStatus = errors.New("invalid support ticket status")
var ErrInvalidSupportRole = errors.New("invalid support role")
var ErrInvalidTicketLine = errors.New("invalid support ticket line")
var ErrInvalidRating = errors.New("invalid support ticket rating")
var ErrForbidden = errors.New("support access forbidden")

type TicketService interface {
	GetProfileBySessionID(ctx context.Context, sessionID models.SessionID) (*models.Profile, error)
	GetProfileByUserAccountID(ctx context.Context, userAccountID int64) (*models.Profile, error)
	GetUserAccountByProfileID(ctx context.Context, profileID int64) (*models.UserAccount, error)
	GetUserProfileByProfileID(ctx context.Context, profileID int64) (*models.UserProfile, error)
	Save(ctx context.Context, ticket *models.SupportTicket) (int64, error)
	GetByID(ctx context.Context, ticketID, profileID int64) (*models.SupportTicket, error)
	GetByIDForAgent(ctx context.Context, ticketID int64) (*models.SupportTicket, error)
	GetByProfileID(ctx context.Context, profileID int64) ([]models.SupportTicket, error)
	GetAll(ctx context.Context, role models.SupportRole, filter TicketFilter) ([]models.SupportTicket, error)
	Update(ctx context.Context, ticketID, profileID int64, upd TicketUpdate) (*models.SupportTicket, error)
	UpdateStatus(ctx context.Context, ticketID, profileID int64, status models.TicketStatus) (*models.SupportTicket, error)
	UpdateStatusByAgent(ctx context.Context, ticketID int64, status models.TicketStatus) (*models.SupportTicket, error)
	Assign(ctx context.Context, ticketID, agentID int64) (*models.SupportTicket, error)
	Escalate(ctx context.Context, ticketID int64) (*models.SupportTicket, error)
	Rate(ctx context.Context, ticketID, profileID int64, rating int) (*models.SupportTicket, error)
	GetStats(ctx context.Context) (*models.SupportTicketStats, error)
	SetProfileRole(ctx context.Context, profileID int64, role models.SupportRole) error
	GetProfileRole(ctx context.Context, profileID int64) (models.SupportRole, error)
	IsSupportAgent(ctx context.Context, profileID int64) (bool, error)
	IsAdmin(ctx context.Context, profileID int64) (bool, error)
	CanAccessTicket(ctx context.Context, ticketID, profileID int64, role models.SupportRole) (*models.SupportTicket, error)
	AttachMedia(ctx context.Context, ticketID int64, medias []MediaRef) MediaErrors
	GetMediasByTicketID(ctx context.Context, ticketID int64) []models.Media
	GetMessages(ctx context.Context, ticketID int64) ([]models.SupportTicketMessage, error)
	SaveMessage(ctx context.Context, ticketID, authorID int64, authorRole models.SupportRole, text string) (*models.SupportTicketMessage, error)
}

type ticketService struct {
	ticketRepo  supportrepo.TicketRepository
	mediaClient mediapb.MediaServiceClient
	authClient  authpb.AuthServiceClient
	userClient  userpb.UserServiceClient
}

type Clients struct {
	Media mediapb.MediaServiceClient
	Auth  authpb.AuthServiceClient
	User  userpb.UserServiceClient
}

func NewTicketService(ticketRepo supportrepo.TicketRepository, clients Clients) TicketService {
	return &ticketService{
		ticketRepo:  ticketRepo,
		mediaClient: clients.Media,
		authClient:  clients.Auth,
		userClient:  clients.User,
	}
}

func SetProfileRole(ctx context.Context, ticketService TicketService, profileID int64, role models.SupportRole) error {
	return ticketService.SetProfileRole(ctx, profileID, role)
}

func MakeProfileAdmin(ctx context.Context, ticketService TicketService, profileID int64) error {
	return ticketService.SetProfileRole(ctx, profileID, models.SupportRoleAdmin)
}

func MakeProfileSupportL1(ctx context.Context, ticketService TicketService, profileID int64) error {
	return ticketService.SetProfileRole(ctx, profileID, models.SupportRoleSupportL1)
}

func MakeProfileSupportL2(ctx context.Context, ticketService TicketService, profileID int64) error {
	return ticketService.SetProfileRole(ctx, profileID, models.SupportRoleSupportL2)
}

func (s *ticketService) GetProfileBySessionID(ctx context.Context, sessionID models.SessionID) (*models.Profile, error) {
	if s.authClient == nil || strings.TrimSpace(string(sessionID)) == "" {
		return nil, ErrInvalidInput
	}

	callCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	resp, err := s.authClient.ValidateSession(callCtx, &authpb.ValidateSessionRequest{SessionId: string(sessionID)})
	if err != nil {
		return nil, err
	}
	return s.GetProfileByUserAccountID(ctx, resp.GetUserAccountId())
}

func (s *ticketService) GetProfileByUserAccountID(ctx context.Context, userAccountID int64) (*models.Profile, error) {
	if s.userClient == nil || userAccountID <= 0 {
		return nil, ErrInvalidInput
	}

	callCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	resp, err := s.userClient.GetProfileByUserAccount(callCtx, &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID})
	if err != nil {
		return nil, mapUserGRPCError(err, xerrors.ProfileNotFound)
	}

	return &models.Profile{
		ID:       resp.GetProfileId(),
		IsActive: true,
	}, nil
}

func (s *ticketService) GetUserAccountByProfileID(ctx context.Context, profileID int64) (*models.UserAccount, error) {
	if s.userClient == nil || profileID <= 0 {
		return nil, ErrInvalidInput
	}

	callCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	summary, err := s.userClient.GetProfileSummary(callCtx, &userpb.GetProfileSummaryRequest{ProfileId: profileID})
	if err != nil {
		return nil, mapUserGRPCError(err, xerrors.UserAccountNotFound)
	}

	authUser, err := s.userClient.GetAuthUserByAccount(callCtx, &userpb.GetAuthUserByAccountRequest{UserAccountId: summary.GetUserAccountId()})
	if err != nil {
		return nil, mapUserGRPCError(err, xerrors.UserAccountNotFound)
	}

	var email *string
	if authUser.Email != nil {
		email = authUser.Email
	}
	username := summary.GetUsername()
	if username == "" {
		username = authUser.GetLogin()
	}

	return &models.UserAccount{
		ID:       summary.GetUserAccountId(),
		Username: username,
		Email:    email,
		IsActive: true,
	}, nil
}

func (s *ticketService) GetUserProfileByProfileID(ctx context.Context, profileID int64) (*models.UserProfile, error) {
	if s.userClient == nil || profileID <= 0 {
		return nil, ErrInvalidInput
	}

	callCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	summary, err := s.userClient.GetProfileSummary(callCtx, &userpb.GetProfileSummaryRequest{ProfileId: profileID})
	if err != nil {
		return nil, mapUserGRPCError(err, xerrors.UserProfileNotFound)
	}

	return &models.UserProfile{
		ProfileID:     summary.GetProfileId(),
		UserAccountID: summary.GetUserAccountId(),
		FirstName:     summary.GetFirstName(),
		LastName:      summary.GetLastName(),
		IsActive:      true,
	}, nil
}

func (s *ticketService) Save(ctx context.Context, ticket *models.SupportTicket) (int64, error) {
	if ticket == nil {
		return 0, ErrNilTicket
	}

	now := time.Now()
	if ticket.CreatedAt.IsZero() {
		ticket.CreatedAt = now
	}
	ticket.UpdatedAt = now
	if ticket.Line == 0 {
		ticket.Line = 1
	}

	ticketID, err := s.ticketRepo.Save(ctx, ticket)
	if err != nil {
		return 0, fmt.Errorf("ticketService.Save: %w", err)
	}
	return ticketID, nil
}

func (s *ticketService) GetByID(ctx context.Context, ticketID, profileID int64) (*models.SupportTicket, error) {
	ticket, err := s.ticketRepo.GetByIDAndProfileID(ctx, ticketID, profileID)
	if err != nil {
		return nil, fmt.Errorf("ticketService.GetByID: %w", err)
	}
	return ticket, nil
}

func (s *ticketService) GetByIDForAgent(ctx context.Context, ticketID int64) (*models.SupportTicket, error) {
	ticket, err := s.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("ticketService.GetByIDForAgent: %w", err)
	}
	return ticket, nil
}

func (s *ticketService) GetByProfileID(ctx context.Context, profileID int64) ([]models.SupportTicket, error) {
	tickets, err := s.ticketRepo.GetByProfileID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("ticketService.GetByProfileID: %w", err)
	}
	return tickets, nil
}

func (s *ticketService) GetAll(ctx context.Context, role models.SupportRole, filter TicketFilter) ([]models.SupportTicket, error) {
	if role == models.SupportRoleSupportL1 {
		line := 1
		filter.Line = &line
	}
	if role == models.SupportRoleSupportL2 {
		line := 2
		filter.Line = &line
	}
	if !isSupportRole(role) {
		return nil, ErrForbidden
	}

	tickets, err := s.ticketRepo.GetAll(ctx, supportrepo.TicketFilter{
		Status:          filter.Status,
		Category:        filter.Category,
		Line:            filter.Line,
		AssignedAgentID: filter.AssignedAgentID,
	})
	if err != nil {
		return nil, fmt.Errorf("ticketService.GetAll: %w", err)
	}
	return tickets, nil
}

func (s *ticketService) Update(ctx context.Context, ticketID, profileID int64, upd TicketUpdate) (*models.SupportTicket, error) {
	if upd.IsEmpty() {
		return nil, errors.New("no ticket fields provided for update")
	}

	ticket, err := s.ticketRepo.GetByIDAndProfileID(ctx, ticketID, profileID)
	if err != nil {
		return nil, fmt.Errorf("ticketService.Update get ticket: %w", err)
	}

	if upd.Title != nil {
		ticket.Title = *upd.Title
	}
	if upd.Description != nil {
		ticket.Description = *upd.Description
	}
	if upd.Category != nil {
		ticket.Category = *upd.Category
	}
	ticket.UpdatedAt = time.Now()

	updatedTicket, err := s.ticketRepo.Update(ctx, ticket)
	if err != nil {
		return nil, fmt.Errorf("ticketService.Update save ticket: %w", err)
	}
	return updatedTicket, nil
}

func (s *ticketService) UpdateStatus(ctx context.Context, ticketID, profileID int64, status models.TicketStatus) (*models.SupportTicket, error) {
	if !isValidTicketStatus(status) {
		return nil, ErrInvalidTicketStatus
	}

	updatedAt := time.Now()
	var closedAt *time.Time
	if status == models.TicketStatusClosed {
		closedAt = &updatedAt
	}

	ticket, err := s.ticketRepo.UpdateStatus(ctx, ticketID, profileID, status, closedAt, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("ticketService.UpdateStatus: %w", err)
	}
	return ticket, nil
}

func (s *ticketService) UpdateStatusByAgent(ctx context.Context, ticketID int64, status models.TicketStatus) (*models.SupportTicket, error) {
	if !isValidTicketStatus(status) {
		return nil, ErrInvalidTicketStatus
	}

	updatedAt := time.Now()
	var closedAt *time.Time
	if status == models.TicketStatusClosed {
		closedAt = &updatedAt
	}

	ticket, err := s.ticketRepo.UpdateStatusByID(ctx, ticketID, status, closedAt, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("ticketService.UpdateStatusByAgent: %w", err)
	}
	return ticket, nil
}

func (s *ticketService) Assign(ctx context.Context, ticketID, agentID int64) (*models.SupportTicket, error) {
	ticket, err := s.ticketRepo.Assign(ctx, ticketID, agentID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("ticketService.Assign: %w", err)
	}
	return ticket, nil
}

func (s *ticketService) Escalate(ctx context.Context, ticketID int64) (*models.SupportTicket, error) {
	ticket, err := s.ticketRepo.Escalate(ctx, ticketID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("ticketService.Escalate: %w", err)
	}
	return ticket, nil
}

func (s *ticketService) Rate(ctx context.Context, ticketID, profileID int64, rating int) (*models.SupportTicket, error) {
	if rating < 1 || rating > 5 {
		return nil, ErrInvalidRating
	}

	ticket, err := s.ticketRepo.Rate(ctx, ticketID, profileID, rating, time.Now())
	if err != nil {
		return nil, fmt.Errorf("ticketService.Rate: %w", err)
	}
	return ticket, nil
}

func (s *ticketService) GetStats(ctx context.Context) (*models.SupportTicketStats, error) {
	stats, err := s.ticketRepo.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("ticketService.GetStats: %w", err)
	}
	return stats, nil
}

func (s *ticketService) SetProfileRole(ctx context.Context, profileID int64, role models.SupportRole) error {
	if profileID <= 0 {
		return ErrInvalidInput
	}
	if role != models.SupportRoleAdmin && role != models.SupportRoleSupportL1 && role != models.SupportRoleSupportL2 {
		return ErrInvalidSupportRole
	}
	if err := s.ticketRepo.SetProfileRole(ctx, profileID, role); err != nil {
		return fmt.Errorf("ticketService.SetProfileRole: %w", err)
	}
	return nil
}

func (s *ticketService) GetProfileRole(ctx context.Context, profileID int64) (models.SupportRole, error) {
	if profileID <= 0 {
		return models.SupportRoleUser, ErrInvalidInput
	}
	role, err := s.ticketRepo.GetProfileRole(ctx, profileID)
	if err != nil {
		if errors.Is(err, xerrors.SupportForbidden) {
			return models.SupportRoleUser, nil
		}
		return models.SupportRoleUser, fmt.Errorf("ticketService.GetProfileRole: %w", err)
	}
	return role.Role, nil
}

func (s *ticketService) IsSupportAgent(ctx context.Context, profileID int64) (bool, error) {
	role, err := s.GetProfileRole(ctx, profileID)
	if err != nil {
		return false, err
	}
	return isSupportRole(role), nil
}

func (s *ticketService) IsAdmin(ctx context.Context, profileID int64) (bool, error) {
	role, err := s.GetProfileRole(ctx, profileID)
	if err != nil {
		return false, err
	}
	return role == models.SupportRoleAdmin, nil
}

func (s *ticketService) CanAccessTicket(ctx context.Context, ticketID, profileID int64, role models.SupportRole) (*models.SupportTicket, error) {
	ticket, err := s.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("ticketService.CanAccessTicket: %w", err)
	}
	if ticket.ProfileID == profileID || role == models.SupportRoleAdmin || role == models.SupportRoleSupportL1 || role == models.SupportRoleSupportL2 {
		return ticket, nil
	}
	return nil, ErrForbidden
}

func (s *ticketService) AttachMedia(ctx context.Context, ticketID int64, medias []MediaRef) MediaErrors {
	var out MediaErrors

	for i, mediaRef := range medias {
		media, ok := s.getMedia(ctx, mediaRef.MediaID)
		if !ok {
			out.Errs = append(out.Errs, AttachmentError{Err: xerrors.MediaNotFound, Pos: i})
			continue
		}
		if !strings.HasPrefix(media.MimeType, "image") {
			out.Errs = append(out.Errs, AttachmentError{Err: xerrors.UnsupportedContentType, Pos: i})
			continue
		}
		ticketWithMedia := models.NewTicketWithMedia(ticketID, mediaRef.MediaID, i)
		if err := s.ticketRepo.SaveMedia(ctx, *ticketWithMedia); err != nil {
			out.Errs = append(out.Errs, AttachmentError{Err: err, Pos: i})
		}
	}
	return out
}

func (s *ticketService) GetMediasByTicketID(ctx context.Context, ticketID int64) []models.Media {
	mediaIDs := s.ticketRepo.GetMediaByTicketID(ctx, ticketID)
	medias := make([]models.Media, 0, len(mediaIDs))
	for _, mediaID := range mediaIDs {
		media, ok := s.getMedia(ctx, mediaID)
		if ok && media.Link != "" {
			medias = append(medias, media)
		}
	}
	return medias
}

func (s *ticketService) getMedia(ctx context.Context, mediaID int64) (models.Media, bool) {
	if s.mediaClient == nil || mediaID <= 0 {
		return models.Media{}, false
	}

	callCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	resp, err := s.mediaClient.GetMedia(callCtx, &mediapb.GetMediaRequest{MediaId: mediaID})
	if err != nil || resp == nil {
		return models.Media{}, false
	}
	return models.Media{
		ID:       resp.GetMediaId(),
		MimeType: resp.GetMimeType(),
		Link:     strings.TrimSpace(resp.GetUrl()),
	}, true
}

func (s *ticketService) GetMessages(ctx context.Context, ticketID int64) ([]models.SupportTicketMessage, error) {
	messages, err := s.ticketRepo.GetMessages(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("ticketService.GetMessages: %w", err)
	}
	return messages, nil
}

func (s *ticketService) SaveMessage(ctx context.Context, ticketID, authorID int64, authorRole models.SupportRole, text string) (*models.SupportTicketMessage, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, errors.New(xerrors.InvalidRequest)
	}

	message := &models.SupportTicketMessage{
		TicketID:   ticketID,
		Text:       text,
		AuthorID:   authorID,
		AuthorRole: authorRole,
		CreatedAt:  time.Now(),
	}

	if _, err := s.ticketRepo.SaveMessage(ctx, message); err != nil {
		return nil, fmt.Errorf("ticketService.SaveMessage: %w", err)
	}
	return message, nil
}

func isValidTicketStatus(status models.TicketStatus) bool {
	return status >= models.TicketStatusOpen && status <= models.TicketStatusClosed
}

func isSupportRole(role models.SupportRole) bool {
	return role == models.SupportRoleAdmin || role == models.SupportRoleSupportL1 || role == models.SupportRoleSupportL2
}

func mapUserGRPCError(err error, notFound error) error {
	if status.Code(err) == codes.NotFound {
		return notFound
	}
	return err
}
