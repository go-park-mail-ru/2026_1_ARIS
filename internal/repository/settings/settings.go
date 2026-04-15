package settings

//go:generate mockgen -destination=./../mocks/settings_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/settings UserSettingsRepository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type userSettingsRepository struct {
	db settingsDB
}

type UserSettingsRepository interface {
	GetByUserID(ctx context.Context, userID int64) (*models.UserSettings, error)
	Update(ctx context.Context, userID int64, upd dto.UserSettingsUpdate) (*models.UserSettings, error)
}

type settingsDB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func NewUserSettingsStorage(db settingsDB) UserSettingsRepository {
	return &userSettingsRepository{db: db}
}

func (r *userSettingsRepository) GetByUserID(ctx context.Context, userID int64) (*models.UserSettings, error) {
	logger := logger.FromContext(ctx)
	query := `
		SELECT user_account_id, lang, theme
		FROM user_settings
		WHERE user_account_id = $1`

	start := time.Now()
	row, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("userSettingsRepository.GetByUserID query: %w", err)
	}
	logger.Debug("db query",
		zap.String("query", "userSettingsRepository.GetByUserID"),
		zap.Duration("duration_ms", time.Since(start)))

	settings, err := pgx.CollectOneRow(row, pgx.RowToAddrOfStructByName[models.UserSettings])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, xerrors.ErrUserSettingsNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("userSettingsRepository.GetByUserID scan: %w", err)
	}

	return settings, nil
}

func (r *userSettingsRepository) Update(ctx context.Context, userID int64, upd dto.UserSettingsUpdate) (*models.UserSettings, error) {
	logger := logger.FromContext(ctx)
	if upd.IsEmpty() {
		return r.GetByUserID(ctx, userID)
	}

	args := []any{userID}
	setClauses := []string{}

	// lang
	var langArg any
	if upd.Language != nil {
		langArg = strings.ToUpper(string(*upd.Language))
		args = append(args, langArg)
		setClauses = append(setClauses, fmt.Sprintf("lang = $%d", len(args)))
	}

	// theme
	var themeArg any
	if upd.Theme != nil {
		themeArg = strings.ToLower(string(*upd.Theme))
		args = append(args, themeArg)
		setClauses = append(setClauses, fmt.Sprintf("theme = $%d", len(args)))
	}

	// Собираем SET-часть
	setSQL := ""
	for i, clause := range setClauses {
		if i > 0 {
			setSQL += ", "
		}
		setSQL += clause
	}

	query := fmt.Sprintf(`
		UPDATE user_settings
		SET %s
		WHERE user_account_id = $1
		RETURNING user_account_id, lang, theme`,
		setSQL,
	)

	start := time.Now()
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("userSettingsRepository.Update query: %w", err)
	}
	logger.Debug("db query",
		zap.String("query", "userSettingsRepository.Update"),
		zap.Duration("duration_ms", time.Since(start)))

	settings, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[models.UserSettings])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, xerrors.ErrUserSettingsNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("userSettingsRepository.Update scan: %w", err)
	}

	return settings, nil
}
