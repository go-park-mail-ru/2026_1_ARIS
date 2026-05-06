package grpc

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/support/service"
	supportpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	supportpb.UnimplementedSupportServiceServer
	support service.TicketService
}

func New(support service.TicketService) *Server {
	return &Server{support: support}
}

func (s *Server) GetProfileRole(ctx context.Context, req *supportpb.GetProfileRoleRequest) (*supportpb.GetProfileRoleResponse, error) {
	if req.GetProfileId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid profile id")
	}

	role, err := s.support.GetProfileRole(ctx, req.GetProfileId())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get profile role")
	}

	return &supportpb.GetProfileRoleResponse{Role: string(role)}, nil
}
