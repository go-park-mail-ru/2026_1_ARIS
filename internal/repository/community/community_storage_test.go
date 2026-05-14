package community

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func communityRows() []string {
	return []string{"id", "uid", "title", "bio", "community_type", "profile_id", "username", "cover_media_id", "is_active", "created_at", "updated_at"}
}

func memberRows() []string {
	return []string{"id", "uid", "profile_id", "community_id", "community_role", "is_active", "joined_at", "leave_at", "created_at", "updated_at"}
}

func addCommunityRow(rows *pgxmock.Rows, id int64) *pgxmock.Rows {
	now := time.Now()
	bio := "bio"
	coverID := int64(88)
	return rows.AddRow(id, uuid.New(), "Team", &bio, models.PublicGroup, int64(77), "team", &coverID, true, now, now)
}

func addMemberRow(rows *pgxmock.Rows, profileID int64, role models.CommunityMemberRole) *pgxmock.Rows {
	now := time.Now()
	return rows.AddRow(int64(1), uuid.New(), profileID, int64(10), role, true, now, nil, now, now)
}

func TestCommunityStorageCreate(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewCommunityStorage(mockPool)
	avatarID := int64(9)
	coverID := int64(8)
	bio := "bio"
	input := models.Community{Title: "Team", Bio: &bio, Type: models.PublicGroup, Username: "team", CoverMediaID: &coverID}

	mockPool.ExpectBegin()
	mockPool.ExpectQuery("INSERT INTO profile").WithArgs(pgxmock.AnyArg(), &avatarID).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(77)))
	mockPool.ExpectQuery("INSERT INTO community").
		WithArgs(pgxmock.AnyArg(), input.Title, input.Bio, input.Type, int64(77), input.Username, input.CoverMediaID).
		WillReturnRows(addCommunityRow(pgxmock.NewRows(communityRows()), 10))
	mockPool.ExpectExec("INSERT INTO community_member").
		WithArgs(pgxmock.AnyArg(), int64(5), int64(10), models.Owner).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mockPool.ExpectCommit()

	got, err := repo.Create(context.Background(), input, 5, &avatarID)

	require.NoError(t, err)
	require.Equal(t, int64(10), got.ID)
	require.Equal(t, "team", got.Username)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestCommunityStorageReadMethods(t *testing.T) {
	t.Parallel()

	t.Run("Get", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()
		mockPool.ExpectQuery("WHERE c.id=\\$1").WithArgs(int64(10)).
			WillReturnRows(addCommunityRow(pgxmock.NewRows(communityRows()), 10))

		got, err := NewCommunityStorage(mockPool).Get(context.Background(), 10)
		require.NoError(t, err)
		require.Equal(t, int64(10), got.ID)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("GetByProfileID not found", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()
		mockPool.ExpectQuery("WHERE c.profile_id=\\$1").WithArgs(int64(77)).
			WillReturnError(pgx.ErrNoRows)

		_, err = NewCommunityStorage(mockPool).GetByProfileID(context.Background(), 77)
		require.ErrorIs(t, err, ErrCommunityNotFound)
	})

	t.Run("List", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()
		mockPool.ExpectQuery("ORDER BY c.created_at DESC").WithArgs(20, 0).
			WillReturnRows(addCommunityRow(pgxmock.NewRows(communityRows()), 10))

		got, err := NewCommunityStorage(mockPool).List(context.Background(), 20, 0)
		require.NoError(t, err)
		require.Len(t, got, 1)
	})

	t.Run("Update", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()
		bio := "updated"
		community := models.Community{ID: 10, Title: "Updated", Bio: &bio, Type: models.PrivateGroup, Username: "updated"}
		mockPool.ExpectQuery("UPDATE community").
			WithArgs(community.Title, community.Bio, community.Type, community.Username, community.CoverMediaID, community.ID).
			WillReturnRows(addCommunityRow(pgxmock.NewRows(communityRows()), 10))

		got, err := NewCommunityStorage(mockPool).Update(context.Background(), community)
		require.NoError(t, err)
		require.Equal(t, int64(10), got.ID)
	})
}

func TestCommunityStorageExecMethods(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		expect    func(pgxmock.PgxPoolIface)
		call      func(CommunityRepo) error
		wantError error
	}{
		{
			name: "UpdateAvatar",
			expect: func(mockPool pgxmock.PgxPoolIface) {
				mockPool.ExpectExec("UPDATE profile SET avatar_id").WithArgs((*int64)(nil), int64(77)).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			call: func(repo CommunityRepo) error { return repo.UpdateAvatar(context.Background(), 77, nil) },
		},
		{
			name: "Delete no rows",
			expect: func(mockPool pgxmock.PgxPoolIface) {
				mockPool.ExpectExec("UPDATE community SET is_active=FALSE").WithArgs(int64(10)).
					WillReturnResult(pgxmock.NewResult("UPDATE", 0))
			},
			call:      func(repo CommunityRepo) error { return repo.Delete(context.Background(), 10) },
			wantError: ErrCommunityNotFound,
		},
		{
			name: "DeactivateMember",
			expect: func(mockPool pgxmock.PgxPoolIface) {
				mockPool.ExpectExec("UPDATE community_member").WithArgs(int64(10), int64(77)).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			call: func(repo CommunityRepo) error { return repo.DeactivateMember(context.Background(), 10, 77) },
		},
		{
			name: "DeactivateMember no rows",
			expect: func(mockPool pgxmock.PgxPoolIface) {
				mockPool.ExpectExec("UPDATE community_member").WithArgs(int64(10), int64(77)).
					WillReturnResult(pgxmock.NewResult("UPDATE", 0))
			},
			call:      func(repo CommunityRepo) error { return repo.DeactivateMember(context.Background(), 10, 77) },
			wantError: ErrCommunityMemberNotFound,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()
			tc.expect(mockPool)

			err = tc.call(NewCommunityStorage(mockPool))

			if tc.wantError != nil {
				require.ErrorIs(t, err, tc.wantError)
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestCommunityStorageMembers(t *testing.T) {
	t.Parallel()

	t.Run("GetMember", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()
		mockPool.ExpectQuery("SELECT \\* FROM community_member").WithArgs(int64(10), int64(77)).
			WillReturnRows(addMemberRow(pgxmock.NewRows(memberRows()), 77, models.Admin))

		got, err := NewCommunityStorage(mockPool).GetMember(context.Background(), 10, 77)
		require.NoError(t, err)
		require.Equal(t, models.Admin, got.Role)
	})

	t.Run("ListMembers excludes blocked", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()
		mockPool.ExpectQuery("community_role <> \\$2").WithArgs(int64(10), models.Blocked).
			WillReturnRows(addMemberRow(pgxmock.NewRows(memberRows()), 77, models.Member))

		got, err := NewCommunityStorage(mockPool).ListMembers(context.Background(), 10, false)
		require.NoError(t, err)
		require.Len(t, got, 1)
	})

	t.Run("UpsertMemberRole", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()
		mockPool.ExpectQuery("INSERT INTO community_member").
			WithArgs(pgxmock.AnyArg(), int64(77), int64(10), models.Moderator).
			WillReturnRows(addMemberRow(pgxmock.NewRows(memberRows()), 77, models.Moderator))

		got, err := NewCommunityStorage(mockPool).UpsertMemberRole(context.Background(), 10, 77, models.Moderator)
		require.NoError(t, err)
		require.Equal(t, models.Moderator, got.Role)
	})
}

func TestCommunityStorageHelpersAndErrors(t *testing.T) {
	t.Parallel()

	require.Contains(t, communitySelect(), "FROM community c")
	require.NoError(t, normalizeCommunityError(nil))
	require.ErrorIs(t, normalizeCommunityError(pgx.ErrNoRows), ErrCommunityNotFound)
	require.EqualError(t, normalizeCommunityError(errors.New("boom")), "boom")
}
