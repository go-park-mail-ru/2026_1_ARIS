package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/model"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestChatRepositoriesReturnDBErrors(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	db := repomocks.NewMockDB(ctrl)
	row := repomocks.NewMockRow(ctrl)
	dbErr := errors.New("db down")

	row.EXPECT().Scan(gomock.Any()).Return(dbErr).AnyTimes()
	db.EXPECT().QueryRow(gomock.Any(), gomock.Any(), gomock.Any()).Return(row).AnyTimes()
	db.EXPECT().Exec(gomock.Any(), gomock.Any(), gomock.Any()).Return(pgconn.CommandTag{}, dbErr).AnyTimes()
	db.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbErr).AnyTimes()

	ctx := context.Background()
	store := NewStore(db)
	text := "hi"
	authorID := int64(2)
	packID := int64(1)
	mediaID := int64(2)

	require.Error(t, store.Chats.Save(ctx, &model.Chat{Title: "chat"}))
	_, err := store.Chats.GetByID(ctx, 1)
	require.Error(t, err)
	require.Error(t, store.Chats.Delete(ctx, 1))

	require.Error(t, store.ChatMembers.Save(ctx, model.ChatMember{ChatID: 1, MemberID: 2}))
	_, err = store.ChatMembers.GetByChatID(ctx, 1)
	require.Error(t, err)
	_, err = store.ChatMembers.GetByUserID(ctx, 2)
	require.Error(t, err)
	require.Error(t, store.ChatMembers.Delete(ctx, 1))

	require.Error(t, store.Messages.Save(ctx, &model.Message{ChatID: 1, AuthorID: 2, Text: &text}))
	_, err = store.Messages.GetByID(ctx, 1)
	require.Error(t, err)
	_, err = store.Messages.GetByChatID(ctx, 1, 10, 0)
	require.Error(t, err)
	_, err = store.Messages.GetByChatIDAfter(ctx, 1, 2, 10)
	require.Error(t, err)
	require.Error(t, store.Messages.Update(ctx, &model.Message{ID: 1, Text: &text}))
	require.Error(t, store.Messages.Delete(ctx, 1))

	require.Error(t, store.MessageMedia.Save(ctx, model.MessageMedia{MessageID: 1, MediaID: 2}))
	_, err = store.MessageMedia.GetByMessageIDs(ctx, []int64{1})
	require.Error(t, err)
	_, err = store.MessageMedia.GetMediaAuthorID(ctx, 2)
	require.Error(t, err)
	require.Error(t, store.MessageMedia.DeleteByMessageID(ctx, 1))
	_, err = store.MessageMedia.GetMediaInfo(ctx, 2)
	require.Error(t, err)
	media, err := store.MessageMedia.GetByMessageIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, media)

	_, err = store.Stickers.Get(ctx, 1)
	require.Error(t, err)
	_, err = store.Stickers.GetPack(ctx, 1)
	require.Error(t, err)
	_, err = store.Stickers.ListPacks(ctx, 10, 0)
	require.Error(t, err)
	_, err = store.Stickers.ListPacksByAuthorID(ctx, 2, 10, 0)
	require.Error(t, err)
	_, err = store.Stickers.SearchPacks(ctx, "cat", 10, 0)
	require.Error(t, err)
	_, err = store.Stickers.ListByPackID(ctx, 1, 10, 0)
	require.Error(t, err)
	_, err = store.Stickers.CreatePack(ctx, model.StickerPack{AuthorID: &authorID, Title: "pack"})
	require.Error(t, err)
	_, err = store.Stickers.CreateSticker(ctx, model.Sticker{PackID: &packID, MediaID: &mediaID})
	require.Error(t, err)
	_, err = store.Stickers.NextStickerOrder(ctx, 1)
	require.Error(t, err)
	_, err = store.Stickers.GetMediaInfo(ctx, 2)
	require.Error(t, err)

	require.Error(t, store.Reactions.Upsert(ctx, 1, 2, "like"))
	require.Error(t, store.Reactions.Delete(ctx, 1, 2))
	_, err = store.Reactions.GetSummaryByMessageIDs(ctx, []int64{1})
	require.Error(t, err)
	_, err = store.Reactions.GetUserReactionsByMessageIDs(ctx, []int64{1}, 2)
	require.Error(t, err)
	summary, err := store.Reactions.GetSummaryByMessageIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, summary)
	reactions, err := store.Reactions.GetUserReactionsByMessageIDs(ctx, nil, 2)
	require.NoError(t, err)
	require.Empty(t, reactions)
}
