package service

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"github.com/google/uuid"
)

type ChatService interface {
	GetUserChats(ctx context.Context, userID uuid.UUID) ([]models.Chat, error)
	CreatePrivateChat(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Chat, error)
	GetChatByID(ctx context.Context, chatID uuid.UUID) (*models.Chat, error)
	CheckUserInChat(ctx context.Context, chatID, userID uuid.UUID) (bool, error)
}

type chatService struct {
	chatRepo       repository.ChatRepo
	chatMemberRepo repository.ChatMemberRepo
}

func NewChatService(chatRepo repository.ChatRepo, chatMemberRepo repository.ChatMemberRepo) ChatService {
	return &chatService{
		chatRepo:       chatRepo,
		chatMemberRepo: chatMemberRepo,
	}
}

func (s *chatService) GetUserChats(ctx context.Context, userID uuid.UUID) ([]models.Chat, error) {
	members, err := s.chatMemberRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	chats := make([]models.Chat, 0, len(members))
	for _, m := range members {
		chat, err := s.chatRepo.GetByID(ctx, m.ChatID)
		if err != nil {
			continue
		}
		chats = append(chats, *chat)
	}
	return chats, nil
}

func (s *chatService) CreatePrivateChat(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Chat, error) {

	members1, err := s.chatMemberRepo.GetByUserID(ctx, user1ID)
	if err != nil {
		return nil, err
	}
	for _, m := range members1 {
		chat, err := s.chatRepo.GetByID(ctx, m.ChatID)
		if err != nil {
			continue
		}
		if chat.TypeID != models.PrivateChat {
			continue
		}
		members2, err := s.chatMemberRepo.GetByChatID(ctx, chat.ID)
		if err != nil {
			continue
		}
		for _, m2 := range members2 {
			if m2.MemberID == user2ID {

				return chat, nil
			}
		}
	}
	// Создаём новый чат
	chatID := uuid.New()
	chat := models.Chat{
		ID:        chatID,
		TypeID:    models.PrivateChat,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		IsDeleted: false,
	}
	if err := s.chatRepo.Save(ctx, chat); err != nil {
		return nil, err
	}
	member1 := models.ChatMember{
		ID:        uuid.New(),
		ChatID:    chatID,
		MemberID:  user1ID,
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Role:      "member",
	}
	member2 := models.ChatMember{
		ID:        uuid.New(),
		ChatID:    chatID,
		MemberID:  user2ID,
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Role:      "member",
	}
	if err := s.chatMemberRepo.Save(ctx, member1); err != nil {
		return nil, err
	}
	if err := s.chatMemberRepo.Save(ctx, member2); err != nil {
		return nil, err
	}
	return &chat, nil
}

func (s *chatService) GetChatByID(ctx context.Context, chatID uuid.UUID) (*models.Chat, error) {
	return s.chatRepo.GetByID(ctx, chatID)
}

func (s *chatService) CheckUserInChat(ctx context.Context, chatID, userID uuid.UUID) (bool, error) {
	members, err := s.chatMemberRepo.GetByChatID(ctx, chatID)
	if err != nil {
		return false, err
	}
	for _, m := range members {
		if m.MemberID == userID {
			return true, nil
		}
	}
	return false, nil
}
