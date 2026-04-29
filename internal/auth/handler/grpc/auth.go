package grpc

import (
	"context"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/auth/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	authpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	authpb.UnimplementedAuthServiceServer
	auth *service.Service
}

func New(auth *service.Service) *Server {
	return &Server{auth: auth}
}

func (s *Server) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.AuthResponse, error) {
	result, err := s.auth.Register(ctx, service.RegisterInput{
		FirstName: req.GetFirstName(),
		LastName:  req.GetLastName(),
		Login:     req.GetLogin(),
		Password1: req.GetPassword1(),
		Password2: req.GetPassword2(),
		Birthday:  req.GetBirthday(),
		Gender:    fromProtoGender(req.GetGender()),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return toProtoAuthResponse(result), nil
}

func (s *Server) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.AuthResponse, error) {
	result, err := s.auth.Login(ctx, service.LoginInput{Login: req.GetLogin(), Password: req.GetPassword()})
	if err != nil {
		return nil, toStatus(err)
	}
	return toProtoAuthResponse(result), nil
}

func (s *Server) ValidateSession(ctx context.Context, req *authpb.ValidateSessionRequest) (*authpb.ValidateSessionResponse, error) {
	session, err := s.auth.ValidateSession(ctx, req.GetSessionId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &authpb.ValidateSessionResponse{
		UserAccountId: session.UserID,
		ExpiresAt:     session.ExpiredAt.UTC().Format(time.RFC3339Nano),
	}, nil
}

func (s *Server) Logout(ctx context.Context, req *authpb.LogoutRequest) (*authpb.LogoutResponse, error) {
	if err := s.auth.Logout(ctx, req.GetSessionId()); err != nil {
		return nil, status.Error(codes.Internal, "logout failed")
	}
	return &authpb.LogoutResponse{Ok: true}, nil
}

func fromProtoGender(value authpb.Gender) models.Gender {
	if value == authpb.Gender_GENDER_MALE {
		return models.Male
	}
	return models.Female
}

func toProtoAuthResponse(result *service.AuthResult) *authpb.AuthResponse {
	return &authpb.AuthResponse{
		User: &authpb.AuthUser{
			UserAccountId: result.User.UserAccountID,
			ProfileId:     result.User.ProfileID,
			FirstName:     result.User.FirstName,
			LastName:      result.User.LastName,
			AvatarUrl:     result.User.AvatarURL,
			CreatedAt:     result.User.CreatedAt.UTC().Format(time.RFC3339Nano),
		},
		Session: &authpb.Session{
			Id:        string(result.Session.SessionID),
			ExpiresAt: result.Session.ExpiredAt.UTC().Format(time.RFC3339Nano),
		},
	}
}

func toStatus(err error) error {
	switch {
	case errors.Is(err, service.ErrLoginAlreadyExists):
		return status.Error(codes.AlreadyExists, "login already exists")
	case errors.Is(err, service.ErrInvalidCredentials), errors.Is(err, service.ErrSessionNotFound):
		return status.Error(codes.Unauthenticated, "unauthenticated")
	case errors.Is(err, service.ErrInvalidInput), errors.Is(err, service.ErrInvalidBirthday), errors.Is(err, service.ErrTooYoung):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
