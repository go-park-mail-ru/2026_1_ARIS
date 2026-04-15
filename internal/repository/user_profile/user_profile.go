package userprofile

//go:generate mockgen -destination=./../mocks/user_profile_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile UserProfileRepo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type UserProfileRepo interface {
	Save(ctx context.Context, userProfile models.UserProfile) (int64, error)
	Get(ctx context.Context, userProfileID int64) (*models.UserProfile, error)
	GetByProfileID(ctx context.Context, profileID int64) (*models.UserProfile, error)
	GetByUserAccountID(ctx context.Context, userAccountID int64) (*models.UserProfile, error)
	Update(ctx context.Context, dto dto.UpdateUserProfileDTO) error
}

type userProfileStorage struct {
	db userProfileDB
	// logger
}

type userProfileDB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func NewUserProfileStorage(db userProfileDB) UserProfileRepo {
	return &userProfileStorage{
		db: db,
	}
}

func (storage *userProfileStorage) Update(ctx context.Context, dto dto.UpdateUserProfileDTO) error {
	logger := logger.FromContext(ctx)
	setClauses := []string{}
	args := []any{}
	argIdx := 1

	// собираем запрос на обновление, чтобы обновлять только то, что изменилось
	if dto.FirstName != nil {
		setClauses = append(setClauses, fmt.Sprintf("first_name=$%d", argIdx))
		args = append(args, *dto.FirstName)
		argIdx++
	}
	if dto.LastName != nil {
		setClauses = append(setClauses, fmt.Sprintf("last_name=$%d", argIdx))
		args = append(args, *dto.LastName)
		argIdx++
	}
	if dto.Bio != nil {
		setClauses = append(setClauses, fmt.Sprintf("bio=$%d", argIdx))
		args = append(args, *dto.Bio)
		argIdx++
	}
	if dto.BirthdayDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("birthday_date=$%d", argIdx))
		args = append(args, *dto.BirthdayDate)
		argIdx++
	}
	if dto.Gender != nil {
		setClauses = append(setClauses, fmt.Sprintf("gender=$%d", argIdx))
		args = append(args, *dto.Gender)
		argIdx++
	}
	if dto.NativeTown != nil {
		setClauses = append(setClauses, fmt.Sprintf("native_town=$%d", argIdx))
		args = append(args, *dto.NativeTown)
		argIdx++
	}
	if dto.Town != nil {
		setClauses = append(setClauses, fmt.Sprintf("town=$%d", argIdx))
		args = append(args, *dto.Town)
		argIdx++
	}
	if dto.Institution != nil {
		setClauses = append(setClauses, fmt.Sprintf("institution=$%d", argIdx))
		args = append(args, *dto.Institution)
		argIdx++
	}
	if dto.Group != nil {
		setClauses = append(setClauses, fmt.Sprintf("study_group=$%d", argIdx))
		args = append(args, *dto.Group)
		argIdx++
	}
	if dto.Company != nil {
		setClauses = append(setClauses, fmt.Sprintf("company=$%d", argIdx))
		args = append(args, *dto.Company)
		argIdx++
	}
	if dto.JobTitle != nil {
		setClauses = append(setClauses, fmt.Sprintf("job_title=$%d", argIdx))
		args = append(args, *dto.JobTitle)
		argIdx++
	}
	if dto.Interests != nil {
		setClauses = append(setClauses, fmt.Sprintf("interests=$%d", argIdx))
		args = append(args, *dto.Interests)
		argIdx++
	}
	if dto.FavMusic != nil {
		setClauses = append(setClauses, fmt.Sprintf("fav_music=$%d", argIdx))
		args = append(args, *dto.FavMusic)
		argIdx++
	}

	// нечего обновлять
	if len(setClauses) == 0 {
		return nil
	}

	args = append(args, dto.ID)

	query := fmt.Sprintf("UPDATE user_profile SET %s WHERE id=$%d", strings.Join(setClauses, ", "), argIdx)

	start := time.Now()
	res, err := storage.db.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	logger.Debug("db query",
		zap.String("query", "userProfileStorage.Update"),
		zap.Duration("duration_ms", time.Since(start)))

	if res.RowsAffected() != 1 {
		return errors.New("UPDATE affected not on 1 row")
	}

	return nil
}

func (storage *userProfileStorage) Save(ctx context.Context, userProfile models.UserProfile) (int64, error) {
	logger := logger.FromContext(ctx)
	query := `INSERT INTO user_profile (uid, user_account_id, profile_id, first_name, last_name, bio, birthday_date, gender) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id;`

	start := time.Now()
	row := storage.db.QueryRow(ctx, query, uuid.New(), userProfile.UserAccountID, userProfile.ProfileID, userProfile.FirstName, userProfile.LastName, userProfile.Bio, userProfile.BirthdayDate, userProfile.Gender)

	logger.Debug("db query",
		zap.String("query", "userProfileStorage.Save"),
		zap.Duration("duration_ms", time.Since(start)))

	var userAccountID int64

	err := row.Scan(&userAccountID)
	if err != nil {
		return 0, err
	}

	return userAccountID, nil
}

func (storage *userProfileStorage) Get(ctx context.Context, userProfileID int64) (*models.UserProfile, error) {
	logger := logger.FromContext(ctx)
	query := `SELECT * FROM user_profile WHERE id=$1;`

	var userProfile models.UserProfile
	start := time.Now()
	err := pgxscan.Get(ctx, storage.db, &userProfile, query, userProfileID)
	if err != nil {
		return nil, err
	}
	logger.Debug("db query",
		zap.String("query", "userProfileStorage.Get"),
		zap.Duration("duration_ms", time.Since(start)))

	return &userProfile, nil
}

func (storage *userProfileStorage) GetByProfileID(ctx context.Context, profileID int64) (*models.UserProfile, error) {
	logger := logger.FromContext(ctx)
	query := `SELECT * FROM user_profile WHERE profile_id=$1;`

	var userProfile models.UserProfile

	start := time.Now()
	err := pgxscan.Get(ctx, storage.db, &userProfile, query, profileID)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, xerrors.UserProfileNotFound
		}
		return nil, err
	}
	logger.Debug("db query",
		zap.String("query", "userProfileStorage.GetByProfileID"),
		zap.Duration("duration_ms", time.Since(start)))

	return &userProfile, nil
}

func (storage *userProfileStorage) GetByUserAccountID(ctx context.Context, userAccountID int64) (*models.UserProfile, error) {
	logger := logger.FromContext(ctx)
	query := `SELECT * FROM user_profile WHERE user_account_id=$1;`

	var userProfile models.UserProfile

	start := time.Now()
	err := pgxscan.Get(ctx, storage.db, &userProfile, query, userAccountID)
	if err != nil {
		return nil, err
	}
	logger.Debug("db query",
		zap.String("query", "userProfileStorage.GetByUserAccountID"),
		zap.Duration("duration_ms", time.Since(start)))

	return &userProfile, nil
}
