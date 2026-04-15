package settings

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/settings"
)

//go:generate mockgen -destination=../mocks/user_settings_service_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/service/settings UserSettingsService

type UserSettingsService interface {
	GetByUserID(ctx context.Context, userID int64) (*models.UserSettings, error)
	Update(ctx context.Context, userID int64, upd dto.UserSettingsUpdate) (*models.UserSettings, error)
}

func defaultSettings(userID int64) *models.UserSettings {
	return &models.UserSettings{
		UserAccountID: userID,
		Language:      models.LanguageRU,
		Theme:         models.ThemeLight,
	}
}

type userSettingsService struct {
	settingsRepo settings.UserSettingsRepository
}

func NewUserSettingsService(settingsRepo settings.UserSettingsRepository) UserSettingsService {
	return &userSettingsService{settingsRepo: settingsRepo}
}

func (s *userSettingsService) GetByUserID(ctx context.Context, userID int64) (*models.UserSettings, error) {
	settings, err := s.settingsRepo.GetByUserID(ctx, userID)
	if errors.Is(err, xerrors.ErrUserSettingsNotFound) {
		return defaultSettings(userID), nil
	}
	if err != nil {
		return nil, fmt.Errorf("userSettingsService.GetByUserID: %w", err)
	}

	return settings, nil
}

func (s *userSettingsService) Update(ctx context.Context, userID int64, upd dto.UserSettingsUpdate) (*models.UserSettings, error) {
	if upd.IsEmpty() {
		return nil, xerrors.ErrNothingToUpdate
	}

	settings, err := s.settingsRepo.Update(ctx, userID, upd)
	if errors.Is(err, xerrors.ErrUserSettingsNotFound) {
		return applyUpdate(defaultSettings(userID), upd), nil
	}
	if err != nil {
		return nil, fmt.Errorf("userSettingsService.Update: %w", err)
	}

	return settings, nil
}

func applyUpdate(s *models.UserSettings, upd dto.UserSettingsUpdate) *models.UserSettings {
	if upd.Language != nil {
		s.Language = *upd.Language
	}
	if upd.Theme != nil {
		s.Theme = *upd.Theme
	}
	return s
}
