package grpc

import (
	"context"
	"errors"
	"time"

	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	userpb.UnimplementedUserServiceServer
	user *usecase.Service
}

func New(user *usecase.Service) *Server {
	return &Server{user: user}
}

func (s *Server) CheckUsernameAvailable(ctx context.Context, req *userpb.CheckUsernameAvailableRequest) (*userpb.CheckUsernameAvailableResponse, error) {
	available, err := s.user.CheckUsernameAvailable(ctx, req.GetUsername())
	if err != nil {
		return nil, toStatus(err)
	}
	return &userpb.CheckUsernameAvailableResponse{Available: available}, nil
}

func (s *Server) CreateAuthUser(ctx context.Context, req *userpb.CreateAuthUserRequest) (*userpb.AuthUserResponse, error) {
	user, err := s.user.CreateAuthUser(ctx, usecase.CreateAuthUserInput{
		Username:     req.GetUsername(),
		PasswordHash: req.GetPasswordHash(),
		FirstName:    req.GetFirstName(),
		LastName:     req.GetLastName(),
		Birthday:     req.GetBirthday(),
		Gender:       fromProtoGender(req.GetGender()),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return toAuthUserResponse(user), nil
}

func (s *Server) GetCredentialsByLogin(ctx context.Context, req *userpb.GetCredentialsByLoginRequest) (*userpb.GetCredentialsByLoginResponse, error) {
	credentials, err := s.user.GetCredentialsByLogin(ctx, req.GetLogin())
	if err != nil {
		return nil, toStatus(err)
	}
	return &userpb.GetCredentialsByLoginResponse{
		UserAccountId: credentials.UserAccountID,
		PasswordHash:  credentials.PasswordHash,
	}, nil
}

func (s *Server) GetAuthUserByAccount(ctx context.Context, req *userpb.GetAuthUserByAccountRequest) (*userpb.AuthUserResponse, error) {
	user, err := s.user.GetAuthUserByAccount(ctx, req.GetUserAccountId())
	if err != nil {
		return nil, toStatus(err)
	}
	return toAuthUserResponse(user), nil
}

func (s *Server) GetProfileByUserAccount(ctx context.Context, req *userpb.GetProfileByUserAccountRequest) (*userpb.GetProfileByUserAccountResponse, error) {
	profile, err := s.user.GetProfileByUserAccount(ctx, req.GetUserAccountId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &userpb.GetProfileByUserAccountResponse{ProfileId: profile.ID}, nil
}

func (s *Server) GetUserProfileByUserAccount(ctx context.Context, req *userpb.GetUserProfileByUserAccountRequest) (*userpb.GetUserProfileByUserAccountResponse, error) {
	userProfile, err := s.user.GetUserProfileByUserAccount(ctx, req.GetUserAccountId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &userpb.GetUserProfileByUserAccountResponse{UserProfileId: userProfile.ID}, nil
}

func (s *Server) GetProfileSummary(ctx context.Context, req *userpb.GetProfileSummaryRequest) (*userpb.GetProfileSummaryResponse, error) {
	user, err := s.user.GetProfileSummary(ctx, req.GetProfileId())
	if err != nil {
		return nil, toStatus(err)
	}
	resp := &userpb.GetProfileSummaryResponse{
		ProfileId:     user.ProfileID,
		UserAccountId: user.UserAccountID,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Username:      user.Login,
	}
	if user.AvatarID != nil {
		resp.AvatarId = user.AvatarID
	}
	return resp, nil
}

func (s *Server) SearchProfiles(ctx context.Context, req *userpb.SearchProfilesRequest) (*userpb.SearchProfilesResponse, error) {
	profiles, err := s.user.SearchProfiles(ctx, req.GetQuery(), int(req.GetLimit()))
	if err != nil {
		return nil, toStatus(err)
	}
	resp := &userpb.SearchProfilesResponse{Profiles: make([]*userpb.SearchProfileItem, 0, len(profiles))}
	for _, profile := range profiles {
		item := &userpb.SearchProfileItem{
			ProfileId:     profile.ProfileID,
			UserAccountId: profile.UserAccountID,
			Username:      profile.Username,
			FirstName:     profile.FirstName,
			LastName:      profile.LastName,
		}
		if profile.AvatarID != nil {
			item.AvatarId = profile.AvatarID
		}
		resp.Profiles = append(resp.Profiles, item)
	}
	return resp, nil
}

func toAuthUserResponse(user *usecase.AuthUser) *userpb.AuthUserResponse {
	resp := &userpb.AuthUserResponse{
		UserAccountId: user.UserAccountID,
		UserProfileId: user.UserProfileID,
		ProfileId:     user.ProfileID,
		Login:         user.Login,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		CreatedAt:     user.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if user.Email != nil {
		resp.Email = user.Email
	}
	if user.AvatarID != nil {
		resp.AvatarId = user.AvatarID
	}
	return resp
}

func fromProtoGender(value userpb.Gender) model.Gender {
	if value == userpb.Gender_GENDER_MALE {
		return model.Male
	}
	return model.Female
}

func toStatus(err error) error {
	switch {
	case errors.Is(err, usecase.ErrUsernameTaken):
		return status.Error(codes.AlreadyExists, "username already exists")
	case errors.Is(err, usecase.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, usecase.ErrUserAccountNotFound), errors.Is(err, usecase.ErrProfileNotFound), errors.Is(err, usecase.ErrUserProfileNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
