package settings

import (
	"context"
	"errors"
	"testing"

	hdto "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestUserSettingsServiceGetByUserID(t *testing.T) {
	t.Parallel()

	t.Run("from repo", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := repomocks.NewMockUserSettingsRepository(ctrl)
		svc := NewUserSettingsService(repo)
		want := &models.UserSettings{UserAccountID: 1, Language: models.LanguageEN, Theme: models.ThemeDark}
		repo.EXPECT().GetByUserID(gomock.Any(), int64(1)).Return(want, nil)
		got, err := svc.GetByUserID(context.Background(), 1)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("not found returns defaults", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := repomocks.NewMockUserSettingsRepository(ctrl)
		svc := NewUserSettingsService(repo)
		repo.EXPECT().GetByUserID(gomock.Any(), int64(9)).Return(nil, xerrors.ErrUserSettingsNotFound)
		got, err := svc.GetByUserID(context.Background(), 9)
		require.NoError(t, err)
		require.Equal(t, models.LanguageRU, got.Language)
		require.Equal(t, models.ThemeLight, got.Theme)
		require.Equal(t, int64(9), got.UserAccountID)
	})

	t.Run("other error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := repomocks.NewMockUserSettingsRepository(ctrl)
		svc := NewUserSettingsService(repo)
		repo.EXPECT().GetByUserID(gomock.Any(), int64(1)).Return(nil, errors.New("db"))
		_, err := svc.GetByUserID(context.Background(), 1)
		require.Error(t, err)
	})
}

func TestUserSettingsServiceUpdate(t *testing.T) {
	t.Parallel()
	lang := models.LanguageEN

	t.Run("empty update", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := repomocks.NewMockUserSettingsRepository(ctrl)
		svc := NewUserSettingsService(repo)
		_, err := svc.Update(context.Background(), 1, hdto.UserSettingsUpdate{})
		require.ErrorIs(t, err, xerrors.ErrNothingToUpdate)
	})

	t.Run("repo success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := repomocks.NewMockUserSettingsRepository(ctrl)
		svc := NewUserSettingsService(repo)
		want := &models.UserSettings{UserAccountID: 1, Language: models.LanguageEN}
		repo.EXPECT().Update(gomock.Any(), int64(1), hdto.UserSettingsUpdate{Language: &lang}).Return(want, nil)
		got, err := svc.Update(context.Background(), 1, hdto.UserSettingsUpdate{Language: &lang})
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("not found applies to defaults", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := repomocks.NewMockUserSettingsRepository(ctrl)
		svc := NewUserSettingsService(repo)
		repo.EXPECT().Update(gomock.Any(), int64(5), hdto.UserSettingsUpdate{Language: &lang}).Return(nil, xerrors.ErrUserSettingsNotFound)
		got, err := svc.Update(context.Background(), 5, hdto.UserSettingsUpdate{Language: &lang})
		require.NoError(t, err)
		require.Equal(t, models.LanguageEN, got.Language)
		require.Equal(t, int64(5), got.UserAccountID)
	})
}
