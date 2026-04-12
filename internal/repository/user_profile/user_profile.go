package userprofile

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserProfileRepo interface {
	Save(ctx context.Context, userProfile models.UserProfile) (int64, error)
	Get(ctx context.Context, userProfileID int64) (*models.UserProfile, error)
	GetByProfileID(ctx context.Context, profileID int64) (*models.UserProfile, error)
	GetByUserAccountID(ctx context.Context, userAccountID int64) (*models.UserProfile, error)
	Update(ctx context.Context, dto dto.UpdateUserProfileDTO) error
}

type userProfileStorage struct {
	db *pgxpool.Pool
	// logger
}

type inmemoryUserProfileRepo struct {
	mu           sync.RWMutex
	userProfiles map[int64]models.UserProfile
}

func NewUserProfileStorage(db *pgxpool.Pool) UserProfileRepo {
	return &userProfileStorage{
		db: db,
	}
}

func NewUserProfileRepo() UserProfileRepo {
	repo := inmemoryUserProfileRepo{}
	repo.userProfiles = make(map[int64]models.UserProfile)
	return &repo
}

func (storage *userProfileStorage) Update(ctx context.Context, dto dto.UpdateUserProfileDTO) error {
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

	res, err := storage.db.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	if res.RowsAffected() != 1 {
		return errors.New("UPDATE affected not on 1 row")
	}

	return nil
}

func (storage *userProfileStorage) Save(ctx context.Context, userProfile models.UserProfile) (int64, error) {
	query := `INSERT INTO user_profile (uid, user_account_id, profile_id, first_name, last_name, bio, birthday_date, gender) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id;`

	row := storage.db.QueryRow(ctx, query, uuid.New(), userProfile.UserAccountID, userProfile.ProfileID, userProfile.FirstName, userProfile.LastName, userProfile.Bio, userProfile.BirthdayDate, userProfile.Gender)

	var userAccountID int64

	err := row.Scan(&userAccountID)
	if err != nil {
		return 0, err
	}

	return userAccountID, nil
}

func (storage *userProfileStorage) Get(ctx context.Context, userProfileID int64) (*models.UserProfile, error) {
	query := `SELECT id, uid, user_account_id, profile_id, first_name, last_name, bio, birthday_date, gender, native_town, town, institution, study_group, company, job_title, interests, fav_music, is_active, created_at, updated_at FROM user_profile WHERE id=$1 AND is_active=true;`

	var userProfile models.UserProfile

	err := pgxscan.Get(ctx, storage.db, &userProfile, query, userProfileID)
	if err != nil {
		return nil, err
	}

	return &userProfile, nil
}

func (storage *userProfileStorage) GetByProfileID(ctx context.Context, profileID int64) (*models.UserProfile, error) {
	query := `SELECT id, uid, user_account_id, profile_id, first_name, last_name, bio, birthday_date, gender, native_town, town, institution, study_group, company, job_title, interests, fav_music, is_active, created_at, updated_at FROM user_profile WHERE profile_id=$1 AND is_active=true;`

	var userProfile models.UserProfile

	err := pgxscan.Get(ctx, storage.db, &userProfile, query, profileID)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, xerrors.UserProfileNotFound
		}
		return nil, err
	}

	return &userProfile, nil
}

func (storage *userProfileStorage) GetByUserAccountID(ctx context.Context, userAccountID int64) (*models.UserProfile, error) {
	query := `SELECT id, uid, user_account_id, profile_id, first_name, last_name, bio, birthday_date, gender, native_town, town, institution, study_group, company, job_title, interests, fav_music, is_active, created_at, updated_at FROM user_profile WHERE user_account_id=$1 AND is_active=true;`

	var userProfile models.UserProfile

	err := pgxscan.Get(ctx, storage.db, &userProfile, query, userAccountID)
	if err != nil {
		return nil, err
	}

	return &userProfile, nil
}

func (r *inmemoryUserProfileRepo) Save(ctx context.Context, userProfile models.UserProfile) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.userProfiles[userProfile.ID] = userProfile
	return userProfile.ID, nil
}

func (r *inmemoryUserProfileRepo) Get(ctx context.Context, userProfileID int64) (*models.UserProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userProfile, ok := r.userProfiles[userProfileID]
	if !ok {
		return nil, errors.New("UserProfile not found")
	}

	return &userProfile, nil
}

func (r *inmemoryUserProfileRepo) GetByProfileID(ctx context.Context, profileID int64) (*models.UserProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.userProfiles {
		if p.ProfileID == profileID {
			fmt.Println("returned GetByProfileID. profileID =", profileID)
			return &p, nil
		}
	}
	fmt.Println("error in GetByProfileID. profileID =", profileID)
	return nil, errors.New("UserProfile not found")
}

func (r *inmemoryUserProfileRepo) GetByUserAccountID(ctx context.Context, userAccountID int64) (*models.UserProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.userProfiles {
		if p.UserAccountID == userAccountID {
			return &p, nil
		}
	}

	return nil, errors.New("UserProfile not found")
}

// заглушка
func (r *inmemoryUserProfileRepo) Update(ctx context.Context, dto dto.UpdateUserProfileDTO) error {
	return nil
}
