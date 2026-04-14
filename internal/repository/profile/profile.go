package profile

//go:generate mockgen -destination=./../mocks/profile_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile ProfileRepo

import (
	"context"
	"errors"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type ProfileRepo interface {
	Get(ctx context.Context, profileID int64) (*models.Profile, error)
	Save(ctx context.Context, profile models.Profile) (int64, error)
	GetAll(ctx context.Context) ([]models.Profile, error)
	GetByUserAccountID(ctx context.Context, userAccountID int64) (*models.Profile, error)
	UpdateAvatar(ctx context.Context, profileID int64, avatarID *int64) error
}

type profileStorage struct {
	db profileDB
	// logger
}

type profileDB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func NewProfileStorage(db profileDB) ProfileRepo {
	return &profileStorage{
		db: db,
	}
}

func (storage *profileStorage) GetByUserAccountID(ctx context.Context, userAccountID int64) (*models.Profile, error) {
	query := `select p.id, p.uid, p.avatar_id, p.is_active, p.created_at, p.updated_at from user_account ua join user_profile up on up.user_account_id=ua.id join profile p on up.profile_id=p.id where ua.id=$1;`

	var profile models.Profile

	err := pgxscan.Get(ctx, storage.db, &profile, query, userAccountID)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, xerrors.ProfileNotFound
		}
		return nil, err
	}

	return &profile, nil
}

func (storage *profileStorage) Get(ctx context.Context, profileID int64) (*models.Profile, error) {
	query := `SELECT * FROM profile WHERE id=$1;`

	var profile models.Profile

	err := pgxscan.Get(ctx, storage.db, &profile, query, profileID)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, xerrors.ProfileNotFound
		}
		return nil, err
	}

	return &profile, nil
}

func (storage *profileStorage) UpdateAvatar(
	ctx context.Context,
	profileID int64,
	avatarID *int64,
) error {
	commandTag, err := storage.db.Exec(
		ctx,
		`UPDATE profile SET avatar_id = $1, updated_at = NOW() WHERE id = $2`,
		avatarID,
		profileID,
	)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() != 1 {
		return errors.New("UPDATE affected not on 1 row")
	}

	return nil
}

func (storage *profileStorage) Save(ctx context.Context, profile models.Profile) (int64, error) {
	query := `INSERT INTO profile (uid, avatar_id) VALUES ($1, $2) RETURNING id;`

	row := storage.db.QueryRow(ctx, query, uuid.New(), profile.AvatarID)

	var profileID int64

	if err := row.Scan(&profileID); err != nil {
		return 0, err
	}

	return profileID, nil
}

func (storage *profileStorage) GetAll(ctx context.Context) ([]models.Profile, error) {
	query := `SELECT * FROM profile`

	rows, err := storage.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	profiles, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Profile])
	if err != nil {
		return nil, err
	}

	return profiles, nil
}
