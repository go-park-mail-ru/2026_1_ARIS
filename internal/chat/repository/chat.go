package repository

import (
	legacychat "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/chat"
	legacychatmember "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/chat_member"
	legacymessage "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/message"
)

type Store struct {
	Chats       legacychat.ChatRepo
	ChatMembers legacychatmember.ChatMemberRepo
	Messages    legacymessage.MessageRepo
}

func NewStore(chats legacychat.ChatRepo, chatMembers legacychatmember.ChatMemberRepo, messages legacymessage.MessageRepo) Store {
	return Store{Chats: chats, ChatMembers: chatMembers, Messages: messages}
}
