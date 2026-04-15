package message

//go:generate mockgen -destination=../mocks/message_service_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/service/message MessageService

import (
	"context"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/message"
	"github.com/google/uuid"
)

type MessageService interface {
	SendMessage(ctx context.Context, chatID, authorID int64, text string) (*models.Message, error)
	GetMessages(ctx context.Context, chatID int64, limit, offset int) ([]models.Message, error)
	UpdateMessage(ctx context.Context, messageID, authorID int64, newText string) (*models.Message, error) // новый
}

func (s *messageService) UpdateMessage(ctx context.Context, messageID, authorID int64, newText string) (*models.Message, error) {
	msg, err := s.msgRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if msg.AuthorID != authorID {
		return nil, errors.New("forbidden: you can only edit your own messages")
	}
	msg.Text = &newText
	if err := s.msgRepo.Update(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

type messageService struct {
	msgRepo message.MessageRepo
}

func NewMessageService(msgRepo message.MessageRepo) MessageService {
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
	if err := s.msgRepo.Save(ctx, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (s *messageService) GetMessages(ctx context.Context, chatID int64, limit, offset int) ([]models.Message, error) {
	return s.msgRepo.GetByChatID(ctx, chatID, limit, offset)
}
