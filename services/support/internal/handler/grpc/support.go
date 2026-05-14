package grpc

import (
	"context"
	"errors"

	supportpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/support"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	supportpb.UnimplementedSupportServiceServer
	support *usecase.Service
}

func New(support *usecase.Service) *Server {
	return &Server{support: support}
}

func (s *Server) GetProfileRole(ctx context.Context, req *supportpb.GetProfileRoleRequest) (*supportpb.GetProfileRoleResponse, error) {
	role, err := s.support.GetProfileRole(ctx, req.GetProfileId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &supportpb.GetProfileRoleResponse{Role: string(role)}, nil
}

func toStatus(err error) error {
	switch {
	case errors.Is(err, usecase.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
