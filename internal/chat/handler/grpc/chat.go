package grpc

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/chat/service"
	chatpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/chat"
)

type Server struct {
	chatpb.UnimplementedChatServiceServer
	chat *service.Service
}

func New(chat *service.Service) *Server {
	return &Server{chat: chat}
}

func (s *Server) CheckUserInChat(ctx context.Context, req *chatpb.CheckUserInChatRequest) (*chatpb.CheckUserInChatResponse, error) {
	ok, err := s.chat.CheckUserInChat(ctx, req.GetChatId(), req.GetUserAccountId())
	if err != nil {
		return nil, service.ToStatus(err)
	}
	return &chatpb.CheckUserInChatResponse{Ok: ok}, nil
}
