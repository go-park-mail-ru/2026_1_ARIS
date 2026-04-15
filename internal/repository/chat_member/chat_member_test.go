package chatmember

import (
	"context"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func TestChatMemberStorageSaveAndDelete(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewChatMemberStorage(mockPool)
	member := models.ChatMember{
		Uid:      uuid.New(),
		ChatID:   1,
		MemberID: 2,
		JoinedAt: time.Now(),
		Role:     "member",
	}

	mockPool.ExpectExec("INSERT INTO chat_member").
		WithArgs(member.Uid, member.ChatID, member.MemberID, member.JoinedAt, member.Role).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mockPool.ExpectExec("UPDATE chat_member SET leave_at=NOW\\(\\), updated_at=NOW\\(\\) WHERE id=\\$1").
		WithArgs(int64(10)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.Save(context.Background(), member)
	require.NoError(t, err)
	err = repo.Delete(context.Background(), 10)
	require.NoError(t, err)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestChatMemberStorageGetByChatIDAndUserID(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewChatMemberStorage(mockPool)
	now := time.Now()
	rows := pgxmock.NewRows([]string{
		"id", "uid", "chat_id", "profile_id", "joined_at", "is_active", "leave_at", "created_at", "updated_at", "chat_role",
	}).AddRow(int64(1), uuid.New(), int64(7), int64(11), now, true, nil, now, now, "member")
	rows2 := pgxmock.NewRows([]string{
		"id", "uid", "chat_id", "profile_id", "joined_at", "is_active", "leave_at", "created_at", "updated_at", "chat_role",
	}).AddRow(int64(2), uuid.New(), int64(7), int64(11), now, true, nil, now, now, "member")

	mockPool.ExpectQuery("SELECT \\* FROM chat_member WHERE chat_id=\\$1 AND leave_at IS NULL").
		WithArgs(int64(7)).
		WillReturnRows(rows)
	mockPool.ExpectQuery("SELECT \\* FROM chat_member WHERE profile_id=\\$1 AND leave_at IS NULL").
		WithArgs(int64(11)).
		WillReturnRows(rows2)

	byChat, err := repo.GetByChatID(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, byChat, 1)

	byUser, err := repo.GetByUserID(context.Background(), 11)
	require.NoError(t, err)
	require.Len(t, byUser, 1)
	require.NoError(t, mockPool.ExpectationsWereMet())
}
