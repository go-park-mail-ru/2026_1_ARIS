package userprofile

import (
	"context"
	"testing"
	"time"

	hdto "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func TestUserProfileStorageSave(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewUserProfileStorage(mockPool)
	bio := "bio"
	rows := pgxmock.NewRows([]string{"id"}).AddRow(int64(99))
	mockPool.ExpectQuery("INSERT INTO user_profile").
		WithArgs(
			pgxmock.AnyArg(),
			int64(1),
			int64(2),
			"John",
			"Doe",
			&bio,
			pgxmock.AnyArg(),
			models.Gender("male"),
		).
		WillReturnRows(rows)

	gotID, err := repo.Save(context.Background(), models.UserProfile{
		UserAccountID: 1,
		ProfileID:     2,
		FirstName:     "John",
		LastName:      "Doe",
		Bio:           &bio,
		BirthdayDate:  time.Now(),
		Gender:        models.Gender("male"),
	})
	require.NoError(t, err)
	require.Equal(t, int64(99), gotID)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestUserProfileStorageUpdate(t *testing.T) {
	t.Parallel()

	t.Run("nothing to update", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewUserProfileStorage(mockPool)
		err = repo.Update(context.Background(), hdto.UpdateUserProfileDTO{ID: 1})
		require.NoError(t, err)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("updated one row", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewUserProfileStorage(mockPool)
		firstName := "Jane"
		town := "Moscow"
		mockPool.ExpectExec("UPDATE user_profile SET first_name=\\$1, town=\\$2 WHERE id=\\$3").
			WithArgs(firstName, town, int64(3)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err = repo.Update(context.Background(), hdto.UpdateUserProfileDTO{
			ID:        3,
			FirstName: &firstName,
			Town:      &town,
		})
		require.NoError(t, err)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})
}

func TestUserProfileStorageGetByProfileID(t *testing.T) {
	t.Parallel()

	t.Run("not found mapped", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewUserProfileStorage(mockPool)
		rows := pgxmock.NewRows([]string{"id", "uid", "user_account_id", "profile_id", "first_name", "last_name", "bio", "birthday_date", "gender", "native_town", "town", "institution", "study_group", "company", "job_title", "interests", "fav_music", "is_active", "created_at", "updated_at"})
		mockPool.ExpectQuery("SELECT \\* FROM user_profile WHERE profile_id=\\$1;").
			WithArgs(int64(555)).
			WillReturnRows(rows)

		_, err = repo.GetByProfileID(context.Background(), 555)
		require.ErrorIs(t, err, xerrors.UserProfileNotFound)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})
}

func TestUserProfileStorageGetAndGetByUserAccountID(t *testing.T) {
	t.Parallel()

	now := time.Now()
	cols := []string{"id", "uid", "user_account_id", "profile_id", "first_name", "last_name", "bio", "birthday_date", "gender", "native_town", "town", "institution", "study_group", "company", "job_title", "interests", "fav_music", "is_active", "created_at", "updated_at"}

	t.Run("Get", func(t *testing.T) {
		mockPool, _ := pgxmock.NewPool()
		defer mockPool.Close()
		repo := NewUserProfileStorage(mockPool)
		rows := pgxmock.NewRows(cols).AddRow(int64(1), uuid.New(), int64(10), int64(20), "A", "B", nil, now, "male", nil, nil, nil, nil, nil, nil, nil, nil, true, now, now)
		mockPool.ExpectQuery("SELECT \\* FROM user_profile WHERE id=\\$1;").WithArgs(int64(1)).WillReturnRows(rows)
		got, err := repo.Get(context.Background(), 1)
		require.NoError(t, err)
		require.Equal(t, int64(1), got.ID)
	})

	t.Run("GetByUserAccountID", func(t *testing.T) {
		mockPool, _ := pgxmock.NewPool()
		defer mockPool.Close()
		repo := NewUserProfileStorage(mockPool)
		rows := pgxmock.NewRows(cols).AddRow(int64(1), uuid.New(), int64(10), int64(20), "A", "B", nil, now, "male", nil, nil, nil, nil, nil, nil, nil, nil, true, now, now)
		mockPool.ExpectQuery("SELECT \\* FROM user_profile WHERE user_account_id=\\$1;").WithArgs(int64(10)).WillReturnRows(rows)
		got, err := repo.GetByUserAccountID(context.Background(), 10)
		require.NoError(t, err)
		require.Equal(t, int64(10), got.UserAccountID)
	})
}
