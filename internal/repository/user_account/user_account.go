package useraccount

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserAccountRepo interface {
	Save(ctx context.Context, userAccount models.UserAccount) (int64, error)
	Delete(ctx context.Context, id int64) error

	Get(ctx context.Context, id int64) (*models.UserAccount, error)
	GetByEmail(ctx context.Context, email string) (*models.UserAccount, error)
	GetByPhone(ctx context.Context, phone string) (*models.UserAccount, error)
	GetByUsername(ctx context.Context, username string) (*models.UserAccount, error)
	GetByUid(ctx context.Context, uid uuid.UUID) (*models.UserAccount, error)

	List(ctx context.Context, offset, limit int) ([]models.UserAccount, error)
}

type inmemoryUserRepo struct {
	mu           sync.RWMutex
	userAccounts map[int64]models.UserAccount
}

type UserAccountStorage struct {
	db *pgxpool.Pool
	// logger
}

func NewUserAccountStorage(db *pgxpool.Pool) UserAccountRepo {
	return &UserAccountStorage{
		db: db,
	}
}

func (storage *UserAccountStorage) Save(ctx context.Context, userAccount models.UserAccount) (int64, error) {
	query := `INSERT INTO user_account (uid, email, phone, password_hash, username) VALUES ($1, $2, $3, $4, $5) RETURNING id;`

	row := storage.db.QueryRow(ctx, query, uuid.New(), userAccount.Email, userAccount.Phone, userAccount.PasswordHash, userAccount.Username)

	var userAccountID int64

	err := row.Scan(&userAccountID)
	if err != nil {
		return 0, err
	}

	return userAccountID, nil
}

func (storage *UserAccountStorage) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM user_account WHERE id=$1`

	res, err := storage.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		// ни одна запись не удалилась
	}
	if res.RowsAffected() != 1 {
		// удалилась больше, чем одна запись
	}

	return nil
}

func (storage *UserAccountStorage) Get(ctx context.Context, id int64) (*models.UserAccount, error) {
	query := `SELECT id, uid, username, email, phone, is_active, created_at, updated_at FROM user_account WHERE id=$1;`

	var userAccount models.UserAccount

	err := pgxscan.Get(ctx, storage.db, &userAccount, query, id)
	if err != nil {
		return nil, err
	}

	return &userAccount, nil
}

func (storage *UserAccountStorage) GetByEmail(ctx context.Context, email string) (*models.UserAccount, error) {
	query := `SELECT * FROM user_account WHERE email=$1;`

	var userAccount models.UserAccount

	err := pgxscan.Get(ctx, storage.db, &userAccount, query, email)
	if err != nil {
		return nil, err
	}

	return &userAccount, nil
}

func (storage *UserAccountStorage) GetByPhone(ctx context.Context, phone string) (*models.UserAccount, error) {
	query := `SELECT * FROM user_account WHERE phone=$1;`

	var userAccount models.UserAccount

	err := pgxscan.Get(ctx, storage.db, &userAccount, query, phone)
	if err != nil {
		return nil, err
	}

	return &userAccount, nil
}

func (storage *UserAccountStorage) GetByUsername(ctx context.Context, username string) (*models.UserAccount, error) {
	query := `SELECT * FROM user_account WHERE username=$1;`

	var userAccount models.UserAccount

	err := pgxscan.Get(ctx, storage.db, &userAccount, query, username)
	if err != nil {
		return nil, err
	}

	return &userAccount, nil
}

func (storage *UserAccountStorage) GetByUid(ctx context.Context, uid uuid.UUID) (*models.UserAccount, error) {
	query := `SELECT * FROM user_account WHERE uid=$1;`

	var userAccount models.UserAccount

	err := pgxscan.Get(ctx, storage.db, &userAccount, query, uid.String())
	if err != nil {
		return nil, err
	}

	return &userAccount, nil
}

func (storage *UserAccountStorage) List(ctx context.Context, offset, limit int) ([]models.UserAccount, error) {
	query := `SELECT * FROM user_account ORDER BY id LIMIT $1 OFFSET $2`

	rows, err := storage.db.Query(ctx, query, limit, offset)
	if err != nil {
		return []models.UserAccount{}, err
	}

	userAccounts, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.UserAccount])
	if err != nil {
		return []models.UserAccount{}, err
	}

	return userAccounts, nil
}

func NewUserRepo() UserAccountRepo {
	repo := inmemoryUserRepo{}
	repo.userAccounts = make(map[int64]models.UserAccount)
	return &repo
}

func (r *inmemoryUserRepo) Save(ctx context.Context, userAccount models.UserAccount) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.userAccounts[userAccount.ID] = userAccount
	return userAccount.ID, nil
}

func (r *inmemoryUserRepo) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.userAccounts[id]

	if ok {
		delete(r.userAccounts, id)
		return nil
	}

	return errors.New("user not found")
}

func (r *inmemoryUserRepo) Get(ctx context.Context, id int64) (*models.UserAccount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.userAccounts[id]

	if !ok {
		return nil, errors.New("user not found")
	}
	return &user, nil
}

func (r *inmemoryUserRepo) GetByEmail(ctx context.Context, email string) (*models.UserAccount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.userAccounts {
		if *u.Email == email {
			return &u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (r *inmemoryUserRepo) GetByPhone(ctx context.Context, phone string) (*models.UserAccount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.userAccounts {
		if *u.Phone == phone {
			return &u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (r *inmemoryUserRepo) List(ctx context.Context, offset, limit int) ([]models.UserAccount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if offset >= len(r.userAccounts) {
		return []models.UserAccount{}, nil
	}
	if offset+limit > len(r.userAccounts) {
		return slices.Collect(maps.Values(r.userAccounts))[offset:], nil
	}

	return slices.Collect(maps.Values(r.userAccounts))[offset : offset+limit], nil
}

func (r *inmemoryUserRepo) GetByUsername(ctx context.Context, username string) (*models.UserAccount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.userAccounts {
		if u.Username == username {
			return &u, nil
		}
	}
	return nil, errors.New("User not found")
}

func (r *inmemoryUserRepo) GetByUid(ctx context.Context, uid uuid.UUID) (*models.UserAccount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.userAccounts {
		if u.Uid == uid {
			return &u, nil
		}
	}
	return nil, errors.New("User not found")
}
