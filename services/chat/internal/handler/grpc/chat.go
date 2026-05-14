package grpc

import (
	"context"

	chatpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/chat"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/usecase"
)

type Server struct {
	chatpb.UnimplementedChatServiceServer
	chat *usecase.Service
}

func New(chat *usecase.Service) *Server {
	return &Server{chat: chat}
}

func (s *Server) CheckUserInChat(ctx context.Context, req *chatpb.CheckUserInChatRequest) (*chatpb.CheckUserInChatResponse, error) {
	ok, err := s.chat.CheckUserInChat(ctx, req.GetChatId(), req.GetUserAccountId())
	if err != nil {
		return nil, usecase.ToStatus(err)
	}
	return &chatpb.CheckUserInChatResponse{Ok: ok}, nil
}
