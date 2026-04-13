package settings

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userSettingsRepository struct {
	db *pgxpool.Pool
}

type UserSettingsRepository interface {
	GetByUserID(ctx context.Context, userID int64) (*models.UserSettings, error)
	Update(ctx context.Context, userID int64, upd dto.UserSettingsUpdate) (*models.UserSettings, error)
}

func NewUserSettingsStorage(db *pgxpool.Pool) UserSettingsRepository {
	return &userSettingsRepository{db: db}
}

func (r *userSettingsRepository) GetByUserID(ctx context.Context, userID int64) (*models.UserSettings, error) {
	query := `
		SELECT user_account_id, lang, theme
		FROM user_settings
		WHERE user_account_id = $1`

	row, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("userSettingsRepository.GetByUserID query: %w", err)
	}

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
	if upd.IsEmpty() {
		return r.GetByUserID(ctx, userID)
	}

	args := []any{userID}
	setClauses := []string{}

	// lang
	var langArg any
	if upd.Language != nil {
		langArg = string(*upd.Language)
		args = append(args, langArg)
		setClauses = append(setClauses, fmt.Sprintf("lang = $%d", len(args)))
	}

	// theme
	var themeArg any
	if upd.Theme != nil {
		themeArg = string(*upd.Theme)
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

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("userSettingsRepository.Update query: %w", err)
	}

	settings, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[models.UserSettings])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, xerrors.ErrUserSettingsNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("userSettingsRepository.Update scan: %w", err)
	}

	return settings, nil
}
