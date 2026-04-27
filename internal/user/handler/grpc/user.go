package grpc

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/user/service"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
)

type Server struct {
	userpb.UnimplementedUserServiceServer
	user *service.Service
}

func New(user *service.Service) *Server {
	return &Server{user: user}
}

func (s *Server) GetProfileByUserAccount(ctx context.Context, req *userpb.GetProfileByUserAccountRequest) (*userpb.GetProfileByUserAccountResponse, error) {
	profile, err := s.user.GetProfileByUserAccount(ctx, req.GetUserAccountId())
	if err != nil {
		return nil, service.ToStatus(err)
	}

	return &userpb.GetProfileByUserAccountResponse{ProfileId: profile.ID}, nil
}

func (s *Server) GetUserProfileByUserAccount(ctx context.Context, req *userpb.GetUserProfileByUserAccountRequest) (*userpb.GetUserProfileByUserAccountResponse, error) {
	userProfile, err := s.user.GetUserProfileByUserAccount(ctx, req.GetUserAccountId())
	if err != nil {
		return nil, service.ToStatus(err)
	}

	return &userpb.GetUserProfileByUserAccountResponse{UserProfileId: userProfile.ID}, nil
}

func (s *Server) GetProfileSummary(ctx context.Context, req *userpb.GetProfileSummaryRequest) (*userpb.GetProfileSummaryResponse, error) {
	profile, err := s.user.GetProfileSummary(ctx, req.GetProfileId())
	if err != nil {
		return nil, service.ToStatus(err)
	}

	resp := &userpb.GetProfileSummaryResponse{
		ProfileId:     profile.ProfileID,
		UserAccountId: profile.UserAccountID,
		FirstName:     profile.FirstName,
		LastName:      profile.LastName,
		Username:      profile.Username,
	}
	if profile.AvatarID != nil {
		resp.AvatarId = profile.AvatarID
	}

	return resp, nil
}
