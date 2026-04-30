package grpc

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/media/service"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	mediapb.UnimplementedMediaServiceServer
	media *service.Service
}

func New(media *service.Service) *Server {
	return &Server{media: media}
}

func (s *Server) GetMediaURL(ctx context.Context, req *mediapb.GetMediaURLRequest) (*mediapb.GetMediaURLResponse, error) {
	url, err := s.media.GetFileURL(ctx, req.GetMediaId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &mediapb.GetMediaURLResponse{Url: url}, nil
}

func (s *Server) GetMedia(ctx context.Context, req *mediapb.GetMediaRequest) (*mediapb.GetMediaResponse, error) {
	media, err := s.media.GetFile(ctx, req.GetMediaId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &mediapb.GetMediaResponse{
		MediaId:  media.ID,
		Uid:      media.Uid.String(),
		MimeType: media.MimeType,
		Url:      media.Link,
	}, nil
}

func toStatus(err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, service.ErrMediaNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
