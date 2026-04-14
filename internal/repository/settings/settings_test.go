package settings

import (
	"context"
	"testing"

	hdto "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func TestUserSettingsRepositoryGetByUserID(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewUserSettingsStorage(mockPool)
		rows := pgxmock.NewRows([]string{"user_account_id", "lang", "theme"}).
			AddRow(int64(12), "RU", "light")
		mockPool.ExpectQuery("SELECT user_account_id, lang, theme").
			WithArgs(int64(12)).
			WillReturnRows(rows)

		got, err := repo.GetByUserID(context.Background(), 12)
		require.NoError(t, err)
		require.Equal(t, int64(12), got.UserAccountID)
		require.Equal(t, models.LanguageSetting("RU"), got.Language)
		require.Equal(t, models.ThemeSetting("light"), got.Theme)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewUserSettingsStorage(mockPool)
		rows := pgxmock.NewRows([]string{"user_account_id", "lang", "theme"})
		mockPool.ExpectQuery("SELECT user_account_id, lang, theme").
			WithArgs(int64(77)).
			WillReturnRows(rows)

		_, err = repo.GetByUserID(context.Background(), 77)
		require.ErrorIs(t, err, xerrors.ErrUserSettingsNotFound)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})
}

func TestUserSettingsRepositoryUpdate(t *testing.T) {
	t.Parallel()

	lang := models.LanguageSetting("en")
	theme := models.ThemeSetting("DARK")

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewUserSettingsStorage(mockPool)
	rows := pgxmock.NewRows([]string{"user_account_id", "lang", "theme"}).
		AddRow(int64(5), "EN", "dark")
	mockPool.ExpectQuery("UPDATE user_settings").
		WithArgs(int64(5), "EN", "dark").
		WillReturnRows(rows)

	got, err := repo.Update(context.Background(), 5, hdto.UserSettingsUpdate{
		Language: &lang,
		Theme:    &theme,
	})
	require.NoError(t, err)
	require.Equal(t, int64(5), got.UserAccountID)
	require.Equal(t, models.LanguageSetting("EN"), got.Language)
	require.Equal(t, models.ThemeSetting("dark"), got.Theme)
	require.NoError(t, mockPool.ExpectationsWereMet())
}
