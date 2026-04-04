package service

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/chat"
	userservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
	"github.com/google/uuid"
)

type ChatService interface {
	GetUserChats(ctx context.Context, userID int64) ([]models.Chat, error)
	CreatePrivateChat(ctx context.Context, user1ID, user2ID int64) (*models.Chat, error)
	GetChatByID(ctx context.Context, chatID int64) (*models.Chat, error)
	GetChatMembers(ctx context.Context, chatID int64) ([]models.ChatMember, error)
	CheckUserInChat(ctx context.Context, chatID, userID int64) (bool, error)
}

type chatService struct {
	chatRepo       *chat.SQLChatRepo
	chatMemberRepo repository.ChatMemberRepo
	userService    userservice.UserService
}

func NewChatService(
	chatRepo *chat.SQLChatRepo,
	chatMemberRepo repository.ChatMemberRepo,
	userService userservice.UserService,
) ChatService {
	return &chatService{
		chatRepo:       chatRepo,
		chatMemberRepo: chatMemberRepo,
		userService:    userService,
	}
}

func (s *chatService) GetUserChats(ctx context.Context, userID int64) ([]models.Chat, error) {
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

func (s *chatService) CreatePrivateChat(ctx context.Context, user1ID, user2ID int64) (*models.Chat, error) {
	members1, err := s.chatMemberRepo.GetByUserID(ctx, user1ID)
	if err != nil {
		return nil, err
	}

	for _, m := range members1 {
		chat, err := s.chatRepo.GetByID(ctx, m.ChatID)
		if err != nil {
			continue
		}
		if chat.Type != models.PrivateChat {
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

	title := fmt.Sprintf("Личный чат %d", user2ID)
	if s.userService != nil {
		targetProfile, err := s.userService.GetUserProfileByUserAccountID(ctx, user2ID)
		if err == nil && targetProfile != nil {
			title = fmt.Sprintf("%s %s", targetProfile.FirstName, targetProfile.LastName)
		}
	}

	chat := models.NewChat(models.PrivateChat, title, nil)
	if err := s.chatRepo.Save(ctx, chat); err != nil {
		return nil, err
	}

	now := time.Now()
	member1 := models.ChatMember{
		ID:        rand.Int64(),
		Uid:       uuid.New(),
		ChatID:    chat.ID,
		MemberID:  user1ID,
		JoinedAt:  now,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
		Role:      "member",
	}
	member2 := models.ChatMember{
		ID:        rand.Int64(),
		Uid:       uuid.New(),
		ChatID:    chat.ID,
		MemberID:  user2ID,
		JoinedAt:  now,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
		Role:      "member",
	}

	if err := s.chatMemberRepo.Save(ctx, member1); err != nil {
		return nil, err
	}
	if err := s.chatMemberRepo.Save(ctx, member2); err != nil {
		return nil, err
	}

	return chat, nil
}

func (s *chatService) GetChatByID(ctx context.Context, chatID int64) (*models.Chat, error) {
	return s.chatRepo.GetByID(ctx, chatID)
}

func (s *chatService) GetChatMembers(ctx context.Context, chatID int64) ([]models.ChatMember, error) {
	return s.chatMemberRepo.GetByChatID(ctx, chatID)
}

func (s *chatService) CheckUserInChat(ctx context.Context, chatID, userID int64) (bool, error) {
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
