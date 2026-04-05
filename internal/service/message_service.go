package service

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"github.com/google/uuid"
)

type MessageService interface {
	SendMessage(ctx context.Context, chatID, authorID int64, text string) (*models.Message, error)
	GetMessages(ctx context.Context, chatID int64, limit, offset int) ([]models.Message, error)
}

type messageService struct {
	msgRepo repository.MessageRepo
}

func NewMessageService(msgRepo repository.MessageRepo) MessageService {
	return &messageService{msgRepo: msgRepo}
}

func (s *messageService) SendMessage(ctx context.Context, chatID, authorID int64, text string) (*models.Message, error) {
	msg := models.Message{
		Uid:       uuid.New(),
		Text:      &text,
		ChatID:    chatID,
		AuthorID:  authorID,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.msgRepo.Save(ctx, msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (s *messageService) GetMessages(ctx context.Context, chatID int64, limit, offset int) ([]models.Message, error) {
	return s.msgRepo.GetByChatID(ctx, chatID, limit, offset)
}
