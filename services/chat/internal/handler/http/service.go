package http

//go:generate mockgen -source=service.go -destination=mocks/service_mock.go -package=mocks

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/usecase"
)

type ChatService interface {
	SetPresenceOnline(context.Context, int64) error
	SetPresenceOffline(context.Context, int64) error
	ForcePresenceOffline(context.Context, int64) error
	HeartbeatPresence(context.Context, int64) error
	GetUserChats(context.Context, int64) ([]usecase.Chat, error)
	CreatePrivateChat(context.Context, int64, int64) (usecase.Chat, error)
	CheckUserInChat(context.Context, int64, int64) (bool, error)
	GetMessages(context.Context, int64, int64, int, int) ([]usecase.Message, error)
	GetMessagesAfter(context.Context, int64, int64, int64, int) ([]usecase.Message, error)
	SendMessage(context.Context, int64, int64, usecase.MessageInput) (usecase.Message, error)
	UpdateMessage(context.Context, int64, int64, int64, string) (usecase.Message, error)
	GetStickerPacks(context.Context, int64, string, bool, int, int) ([]usecase.StickerPack, error)
	CreateStickerPack(context.Context, int64, usecase.StickerPackInput) (usecase.StickerPack, error)
	GetStickersByPack(context.Context, int64, int64, int, int) ([]usecase.Sticker, error)
	CreateSticker(context.Context, int64, int64, usecase.StickerInput) (usecase.Sticker, error)
	SetMessageReaction(context.Context, int64, int64, int64, string) (usecase.Message, error)
	DeleteMessageReaction(context.Context, int64, int64, int64) (usecase.Message, error)
}
