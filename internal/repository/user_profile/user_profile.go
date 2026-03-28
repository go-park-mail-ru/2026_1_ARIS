package userprofile

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserProfileRepo interface {
	Save(ctx context.Context, userProfile models.UserProfile) (int64, error)
	Get(ctx context.Context, userProfileID int64) (*models.UserProfile, error)
	GetByProfileID(ctx context.Context, profileID int64) (*models.UserProfile, error)
	GetByUserAccountID(ctx context.Context, userAccountID int64) (*models.UserProfile, error)
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
	query := `SELECT * FROM user_profile WHERE id=$1;`

	var userProfile models.UserProfile

	err := pgxscan.Get(ctx, storage.db, &userProfile, query, userProfileID)
	if err != nil {
		return nil, err
	}

	return &userProfile, nil
}

func (storage *userProfileStorage) GetByProfileID(ctx context.Context, profileID int64) (*models.UserProfile, error) {
	query := `SELECT * FROM user_profile WHERE profile_id=$1;`

	var userProfile models.UserProfile

	err := pgxscan.Get(ctx, storage.db, &userProfile, query, profileID)
	if err != nil {
		return nil, err
	}

	return &userProfile, nil
}

func (storage *userProfileStorage) GetByUserAccountID(ctx context.Context, userAccountID int64) (*models.UserProfile, error) {
	query := `SELECT * FROM user_profile WHERE user_account_id=$1;`

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
