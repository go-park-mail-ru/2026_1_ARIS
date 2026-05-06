package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func TestSearchStorageSearchUsers(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewSearchStorage(mockPool)
	avatarID := int64(7)
	rows := pgxmock.NewRows([]string{"profile_id", "user_account_id", "username", "first_name", "last_name", "avatar_id"}).
		AddRow(int64(10), int64(20), "neo", "Neo", "Anderson", &avatarID)
	mockPool.ExpectQuery("FROM user_profile").
		WithArgs(`%n\%eo\_%`, `n\%eo\_%`, 5).
		WillReturnRows(rows)

	got, err := repo.SearchUsers(context.Background(), " n%eo_ ", 5)

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, int64(10), got[0].ProfileID)
	require.Equal(t, "neo", got[0].Username)
	require.Equal(t, &avatarID, got[0].AvatarID)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestSearchStorageSearchCommunities(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewSearchStorage(mockPool)
	bio := "bio"
	avatarID := int64(7)
	coverID := int64(8)
	rows := pgxmock.NewRows([]string{"id", "profile_id", "username", "title", "bio", "community_type", "avatar_id", "cover_media_id"}).
		AddRow(int64(1), int64(2), "team", "Team", &bio, models.PublicGroup, &avatarID, &coverID)
	mockPool.ExpectQuery("FROM community c").
		WithArgs(models.PublicGroup, "%team%", "team%", 3).
		WillReturnRows(rows)

	got, err := repo.SearchCommunities(context.Background(), "team", 3)

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, int64(1), got[0].ID)
	require.Equal(t, "Team", got[0].Title)
	require.Equal(t, &coverID, got[0].CoverMediaID)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestSearchStorageQueryErrors(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		call func(SearchRepo) error
	}{
		{
			name: "users",
			call: func(repo SearchRepo) error {
				_, err := repo.SearchUsers(context.Background(), "neo", 1)
				return err
			},
		},
		{
			name: "communities",
			call: func(repo SearchRepo) error {
				_, err := repo.SearchCommunities(context.Background(), "neo", 1)
				return err
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			if tc.name == "users" {
				mockPool.ExpectQuery("SELECT").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnError(errors.New("db down"))
			} else {
				mockPool.ExpectQuery("SELECT").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnError(errors.New("db down"))
			}
			err = tc.call(NewSearchStorage(mockPool))

			require.EqualError(t, err, "db down")
			require.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestSearchStoreAndPatterns(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewSearchStorage(mockPool)
	store := NewStore(repo)
	require.Same(t, repo, store.Search)
	require.Equal(t, `%a\\b\%\_%`, likePattern(` a\b%_ `))
	require.Equal(t, `a\\b\%\_%`, likePrefixPattern(` a\b%_ `))
}
