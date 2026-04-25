package authgrpc

import (
	"context"

	authservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/auth/service"
	authpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/auth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GRPCAuthHandler struct {
	authpb.AuthServiceServer
	authService    authservice.AuthService
	sessionService authservice.SessionService
}

func NewGRPCAuthHandler(authService authservice.AuthService, sessionService authservice.SessionService) *GRPCAuthHandler {
	return &GRPCAuthHandler{
		authService:    authService,
		sessionService: sessionService,
	}
}

func (h *GRPCAuthHandler) Register(ctx context.Context, in *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {

	registerServiceDTO := authservice.RegisterServiceDTO{
		FirstName: in.FirstName,
		LastName:  in.LastName,
		Login:     in.Login,
		Password:  in.Password1,
		Birthday:  in.BirthdayDate,
		Gender:    int(in.Gender),
	}

	profile, err := h.authService.Register(ctx, registerServiceDTO)
	if err != nil {
		return nil, err
	}

	return &authpb.RegisterResponse{
		ProfileId: profile.ID,
	}, nil
}

func (h *GRPCAuthHandler) Login(ctx context.Context, in *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	loginServiceDTO := authservice.LoginServiceRequestDTO{
		Login:    in.Login,
		Password: in.Password,
	}

	res, err := h.authService.Login(ctx, loginServiceDTO)
	if err != nil {
		return nil, err
	}

	return &authpb.LoginResponse{
		ProfileId: res.ProfileID,
		SessionId: res.SessionID,
		ExpiresAt: timestamppb.New(res.ExpiresAt),
		FirstName: res.FirstName,
		LastName:  res.LastName,
		AvatarUrl: res.AvatarLink,
	}, nil
}
