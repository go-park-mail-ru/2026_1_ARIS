package repository

import (
	"context"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

var ErrSupportRoleNotFound = errors.New("support role not found")

type DB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type ProfileRoleRepo interface {
	SetProfileRole(ctx context.Context, profileID int64, role model.SupportRole) error
	GetProfileRole(ctx context.Context, profileID int64) (*model.SupportProfileRole, error)
}

type profileRoleStorage struct {
	db DB
}

func NewProfileRoleStorage(db DB) ProfileRoleRepo {
	return &profileRoleStorage{db: db}
}

func (s *profileRoleStorage) SetProfileRole(ctx context.Context, profileID int64, role model.SupportRole) error {
	start := time.Now()
	_, err := s.db.Exec(ctx, `
		INSERT INTO support_profile_role (profile_id, role)
		VALUES ($1, $2)
		ON CONFLICT (profile_id) DO UPDATE SET role = EXCLUDED.role
	`, profileID, string(role))
	logQuery(ctx, "profileRoleStorage.SetProfileRole", start)
	return err
}

func (s *profileRoleStorage) GetProfileRole(ctx context.Context, profileID int64) (*model.SupportProfileRole, error) {
	start := time.Now()
	row := s.db.QueryRow(ctx, `SELECT profile_id, role FROM support_profile_role WHERE profile_id = $1`, profileID)
	logQuery(ctx, "profileRoleStorage.GetProfileRole", start)

	var role model.SupportProfileRole
	if err := row.Scan(&role.ProfileID, &role.Role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSupportRoleNotFound
		}
		return nil, err
	}
	return &role, nil
}

func logQuery(ctx context.Context, query string, start time.Time) {
	logg := logger.FromContext(ctx)
	if logg == nil {
		return
	}
	logg.Debug("db query", zap.String("query", query), zap.Duration("duration_ms", time.Since(start)))
}
