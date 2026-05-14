package main

import (
	"context"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	authpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/auth"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errUnsupportedSessionMutation = errors.New("support service cannot mutate sessions")

type authSessionService struct {
	client authpb.AuthServiceClient
}

func (s authSessionService) Create(context.Context, int64) (*models.Session, error) {
	return nil, errUnsupportedSessionMutation
}

func (s authSessionService) Delete(context.Context, models.SessionID) error {
	return errUnsupportedSessionMutation
}

func (s authSessionService) Get(ctx context.Context, sessionID models.SessionID) (*models.Session, error) {
	resp, err := s.client.ValidateSession(ctx, &authpb.ValidateSessionRequest{SessionId: string(sessionID)})
	if err != nil {
		return nil, err
	}

	session := &models.Session{
		SessionID: sessionID,
		UserID:    resp.GetUserAccountId(),
	}
	if expiresAt := resp.GetExpiresAt(); expiresAt != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, expiresAt); err == nil {
			session.ExpiredAt = parsed
		}
	}
	return session, nil
}

type grpcUserService struct {
	client userpb.UserServiceClient
}

func (s grpcUserService) GetProfileByUserAccountID(ctx context.Context, userAccountID int64) (*models.Profile, error) {
	resp, err := s.client.GetProfileByUserAccount(ctx, &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID})
	if err != nil {
		return nil, mapUserGRPCError(err, xerrors.ProfileNotFound)
	}

	return &models.Profile{
		ID:       resp.GetProfileId(),
		IsActive: true,
	}, nil
}

func (s grpcUserService) GetUserAccountByProfileID(ctx context.Context, profileID int64) (*models.UserAccount, error) {
	summary, err := s.client.GetProfileSummary(ctx, &userpb.GetProfileSummaryRequest{ProfileId: profileID})
	if err != nil {
		return nil, mapUserGRPCError(err, xerrors.UserAccountNotFound)
	}

	authUser, err := s.client.GetAuthUserByAccount(ctx, &userpb.GetAuthUserByAccountRequest{UserAccountId: summary.GetUserAccountId()})
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

func (s grpcUserService) GetUserProfileByProfileID(ctx context.Context, profileID int64) (*models.UserProfile, error) {
	summary, err := s.client.GetProfileSummary(ctx, &userpb.GetProfileSummaryRequest{ProfileId: profileID})
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

func mapUserGRPCError(err error, notFound error) error {
	if status.Code(err) == codes.NotFound {
		return notFound
	}
	return err
}
