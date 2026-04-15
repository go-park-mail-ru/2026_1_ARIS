package useraccount

//go:generate mockgen -destination=./../mocks/user_account_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account UserAccountRepo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	pgerrors "github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/pg_errors"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type UserAccountRepo interface {
	Save(ctx context.Context, userAccount models.UserAccount) (int64, error)
	Delete(ctx context.Context, id int64) error
	Update(ctx context.Context, dto dto.UpdateUserAccountDTO) error

	Get(ctx context.Context, id int64) (*models.UserAccount, error)
	GetByEmail(ctx context.Context, email string) (*models.UserAccount, error)
	GetByPhone(ctx context.Context, phone string) (*models.UserAccount, error)
	GetByUsername(ctx context.Context, username string) (*models.UserAccount, error)
	GetByUid(ctx context.Context, uid uuid.UUID) (*models.UserAccount, error)

	List(ctx context.Context, offset, limit int) ([]models.UserAccount, error)
}

type UserAccountStorage struct {
	db userAccountDB
	// logger
}

type userAccountDB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func NewUserAccountStorage(db userAccountDB) UserAccountRepo {
	return &UserAccountStorage{
		db: db,
	}
}

func (storage *UserAccountStorage) Update(ctx context.Context, dto dto.UpdateUserAccountDTO) error {
	logger := logger.FromContext(ctx)
	setClauses := []string{}
	args := []any{}
	argIdx := 1

	// собираем запрос на обновление, чтобы обновлять только то, что изменилось
	if dto.Email != nil {
		setClauses = append(setClauses, fmt.Sprintf("email=$%d", argIdx))
		args = append(args, *dto.Email)
		argIdx++
	}
	if dto.Phone != nil {
		setClauses = append(setClauses, fmt.Sprintf("phone=$%d", argIdx))
		args = append(args, *dto.Phone)
		argIdx++
	}
	if dto.Username != nil {
		setClauses = append(setClauses, fmt.Sprintf("username=$%d", argIdx))
		args = append(args, *dto.Username)
		argIdx++
	}

	// нечего обновлять
	if len(setClauses) == 0 {
		return nil
	}

	args = append(args, dto.ID)

	query := fmt.Sprintf("UPDATE user_account SET %s WHERE id=$%d", strings.Join(setClauses, ", "), argIdx)

	start := time.Now()
	res, err := storage.db.Exec(ctx, query, args...)
	if err != nil {
		return pgerrors.MapPgError(err)
	}
	logger.Debug("db query",
		zap.String("query", "userAccountStorage.Update"),
		zap.Duration("duration_ms", time.Since(start)))

	if res.RowsAffected() != 1 {
		return errors.New("UPDATE affected not on 1 row")
	}

	return nil
}

func (storage *UserAccountStorage) Save(ctx context.Context, userAccount models.UserAccount) (int64, error) {
	logger := logger.FromContext(ctx)
	query := `INSERT INTO user_account (uid, email, phone, password_hash, username) VALUES ($1, $2, $3, $4, $5) RETURNING id;`

	start := time.Now()
	row := storage.db.QueryRow(ctx, query, uuid.New(), userAccount.Email, userAccount.Phone, userAccount.PasswordHash, userAccount.Username)
	logger.Debug("db query",
		zap.String("query", "userAccountStorage.Save"),
		zap.Duration("duration_ms", time.Since(start)))

	var userAccountID int64

	err := row.Scan(&userAccountID)
	if err != nil {
		return 0, err
	}

	return userAccountID, nil
}

func (storage *UserAccountStorage) Delete(ctx context.Context, id int64) error {
	logger := logger.FromContext(ctx)
	query := `DELETE FROM user_account WHERE id=$1`

	start := time.Now()
	res, err := storage.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	logger.Debug("db query",
		zap.String("query", "userAccountStorage.Delete"),
		zap.Duration("duration_ms", time.Since(start)))

	if res.RowsAffected() == 0 {
		// ни одна запись не удалилась
	}
	if res.RowsAffected() != 1 {
		// удалилась больше, чем одна запись
	}

	return nil
}

func (storage *UserAccountStorage) Get(ctx context.Context, id int64) (*models.UserAccount, error) {
	logger := logger.FromContext(ctx)
	query := `SELECT id, uid, username, email, phone, is_active, created_at, updated_at FROM user_account WHERE id=$1;`

	var userAccount models.UserAccount

	start := time.Now()
	err := pgxscan.Get(ctx, storage.db, &userAccount, query, id)
	if err != nil {
		return nil, err
	}
	logger.Debug("db query",
		zap.String("query", "userAccountStorage.Get"),
		zap.Duration("duration_ms", time.Since(start)))

	return &userAccount, nil
}

func (storage *UserAccountStorage) GetByEmail(ctx context.Context, email string) (*models.UserAccount, error) {
	logger := logger.FromContext(ctx)
	query := `SELECT * FROM user_account WHERE email=$1;`

	var userAccount models.UserAccount

	start := time.Now()
	err := pgxscan.Get(ctx, storage.db, &userAccount, query, email)
	if err != nil {
		return nil, err
	}
	logger.Debug("db query",
		zap.String("query", "userAccountStorage.GetByEmail"),
		zap.Duration("duration_ms", time.Since(start)))

	return &userAccount, nil
}

func (storage *UserAccountStorage) GetByPhone(ctx context.Context, phone string) (*models.UserAccount, error) {
	logger := logger.FromContext(ctx)
	query := `SELECT * FROM user_account WHERE phone=$1;`

	var userAccount models.UserAccount

	start := time.Now()
	err := pgxscan.Get(ctx, storage.db, &userAccount, query, phone)
	if err != nil {
		return nil, err
	}
	logger.Debug("db query",
		zap.String("query", "userAccountStorage.GetByPhone"),
		zap.Duration("duration_ms", time.Since(start)))

	return &userAccount, nil
}

func (storage *UserAccountStorage) GetByUsername(ctx context.Context, username string) (*models.UserAccount, error) {
	logger := logger.FromContext(ctx)
	query := `SELECT * FROM user_account WHERE username=$1;`

	var userAccount models.UserAccount

	start := time.Now()
	err := pgxscan.Get(ctx, storage.db, &userAccount, query, username)
	if err != nil {
		return nil, err
	}
	logger.Debug("db query",
		zap.String("query", "userAccountStorage.GetByUsername"),
		zap.Duration("duration_ms", time.Since(start)))

	return &userAccount, nil
}

func (storage *UserAccountStorage) GetByUid(ctx context.Context, uid uuid.UUID) (*models.UserAccount, error) {
	logger := logger.FromContext(ctx)
	query := `SELECT * FROM user_account WHERE uid=$1;`

	var userAccount models.UserAccount

	start := time.Now()
	err := pgxscan.Get(ctx, storage.db, &userAccount, query, uid.String())
	if err != nil {
		return nil, err
	}
	logger.Debug("db query",
		zap.String("query", "userAccountStorage.GetByUid"),
		zap.Duration("duration_ms", time.Since(start)))

	return &userAccount, nil
}

func (storage *UserAccountStorage) List(ctx context.Context, offset, limit int) ([]models.UserAccount, error) {
	logger := logger.FromContext(ctx)
	query := `SELECT * FROM user_account ORDER BY id LIMIT $1 OFFSET $2`

	start := time.Now()
	rows, err := storage.db.Query(ctx, query, limit, offset)
	if err != nil {
		return []models.UserAccount{}, err
	}

	logger.Debug("db query",
		zap.String("query", "userAccountStorage.List"),
		zap.Duration("duration_ms", time.Since(start)))

	userAccounts, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.UserAccount])
	if err != nil {
		return []models.UserAccount{}, err
	}

	return userAccounts, nil
}
