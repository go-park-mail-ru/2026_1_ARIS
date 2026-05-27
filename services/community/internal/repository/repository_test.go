package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/repository"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

// errRow is a minimal pgx.Row that always returns an error on Scan.
type errRow struct{ err error }

func (r *errRow) Scan(...any) error { return r.err }

func TestCommunityRepositoryReturnsDBErrors(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	db := repomocks.NewMockDB(ctrl)
	dbErr := errors.New("db down")

	db.EXPECT().Begin(gomock.Any()).Return(nil, dbErr).AnyTimes()
	db.EXPECT().QueryRow(gomock.Any(), gomock.Any(), gomock.Any()).Return(&errRow{err: dbErr}).AnyTimes()
	db.EXPECT().Exec(gomock.Any(), gomock.Any(), gomock.Any()).Return(pgconn.CommandTag{}, dbErr).AnyTimes()
	db.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbErr).AnyTimes()

	ctx := context.Background()
	store := repository.NewCommunityStorage(db)

	community := model.Community{
		ID:       1,
		Title:    "Test Community",
		Username: "testcommunity",
		Type:     model.PublicGroup,
	}

	// Create uses Begin (transaction) — will fail on Begin
	_, err := store.Create(ctx, community, 10, nil)
	require.Error(t, err)

	// Get uses QueryRow via pgxscan — will fail on Scan
	_, err = store.Get(ctx, 1)
	require.Error(t, err)

	// GetByProfileID uses QueryRow via pgxscan — will fail on Scan
	_, err = store.GetByProfileID(ctx, 10)
	require.Error(t, err)

	// List uses Query — will fail on Query
	_, err = store.List(ctx, 10, 0)
	require.Error(t, err)

	// Update uses Begin (transaction) — will fail on Begin
	_, err = store.Update(ctx, community)
	require.Error(t, err)

	// UpdateAvatar uses Begin (transaction) — will fail on Begin
	err = store.UpdateAvatar(ctx, 1, nil)
	require.Error(t, err)

	// GetAvatarID uses QueryRow — will fail on Scan
	_, err = store.GetAvatarID(ctx, 1)
	require.Error(t, err)

	// ExistsByTitleOrUsername uses QueryRow — will fail on Scan
	_, _, err = store.ExistsByTitleOrUsername(ctx, "title", "username")
	require.Error(t, err)

	// Delete uses Begin (transaction) — will fail on Begin
	err = store.Delete(ctx, 1)
	require.Error(t, err)

	// GetMember uses QueryRow via pgxscan — will fail on Scan
	_, err = store.GetMember(ctx, 1, 10)
	require.Error(t, err)

	// ListMembers uses Query — will fail on Query
	_, err = store.ListMembers(ctx, 1, false)
	require.Error(t, err)

	// UpsertMemberRole uses QueryRow via pgxscan — will fail on Scan
	_, err = store.UpsertMemberRole(ctx, 1, 10, model.Member)
	require.Error(t, err)

	// DeactivateMember uses Exec — will fail on Exec
	err = store.DeactivateMember(ctx, 1, 10)
	require.Error(t, err)

	// Search uses Query — will fail on Query
	_, err = store.Search(ctx, "query", 10)
	require.Error(t, err)
}
