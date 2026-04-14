package useraccount

import (
	"context"
	"testing"
	"time"

	hdto "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func TestUserAccountStorageSave(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewUserAccountStorage(mockPool)
	email := "mail@test.ru"
	phone := "+79990000000"
	rows := pgxmock.NewRows([]string{"id"}).AddRow(int64(101))
	mockPool.ExpectQuery("INSERT INTO user_account").
		WithArgs(pgxmock.AnyArg(), &email, &phone, "hash", "alex").
		WillReturnRows(rows)

	id, err := repo.Save(context.Background(), models.UserAccount{
		Email:        &email,
		Phone:        &phone,
		PasswordHash: "hash",
		Username:     "alex",
	})
	require.NoError(t, err)
	require.Equal(t, int64(101), id)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestUserAccountStorageUpdate(t *testing.T) {
	t.Parallel()

	t.Run("nothing to update", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewUserAccountStorage(mockPool)
		err = repo.Update(context.Background(), hdto.UpdateUserAccountDTO{ID: 1})
		require.NoError(t, err)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("updates single row", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewUserAccountStorage(mockPool)
		email := "new@test.ru"
		username := "newname"
		mockPool.ExpectExec("UPDATE user_account SET email=\\$1, username=\\$2 WHERE id=\\$3").
			WithArgs(email, username, int64(5)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err = repo.Update(context.Background(), hdto.UpdateUserAccountDTO{
			ID:       5,
			Email:    &email,
			Username: &username,
		})
		require.NoError(t, err)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})
}

func TestUserAccountStorageList(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewUserAccountStorage(mockPool)
	now := time.Now()
	rows := pgxmock.NewRows([]string{
		"id", "uid", "username", "email", "phone", "password_hash", "is_active", "created_at", "updated_at",
	}).AddRow(1, uuid.MustParse("00000000-0000-0000-0000-000000000001"), "u1", nil, nil, "h", true, now, now)
	mockPool.ExpectQuery("SELECT \\* FROM user_account ORDER BY id LIMIT \\$1 OFFSET \\$2").
		WithArgs(10, 0).
		WillReturnRows(rows)

	got, err := repo.List(context.Background(), 0, 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "u1", got[0].Username)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestUserAccountStorageReadAndDeleteMethods(t *testing.T) {
	t.Parallel()

	now := time.Now()
	colsShort := []string{"id", "uid", "username", "email", "phone", "is_active", "created_at", "updated_at"}
	colsAll := []string{"id", "uid", "username", "email", "phone", "password_hash", "is_active", "created_at", "updated_at"}
	id := int64(1)
	uid := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	email := "mail@test.ru"
	phone := "+79990000000"

	t.Run("Get", func(t *testing.T) {
		mockPool, _ := pgxmock.NewPool()
		defer mockPool.Close()
		repo := NewUserAccountStorage(mockPool)
		rows := pgxmock.NewRows(colsShort).AddRow(id, uid, "alex", &email, &phone, true, now, now)
		mockPool.ExpectQuery("SELECT id, uid, username, email, phone, is_active, created_at, updated_at FROM user_account WHERE id=\\$1;").WithArgs(id).WillReturnRows(rows)
		got, err := repo.Get(context.Background(), id)
		require.NoError(t, err)
		require.Equal(t, "alex", got.Username)
	})

	t.Run("GetByEmail phone username uid", func(t *testing.T) {
		mockPool, _ := pgxmock.NewPool()
		defer mockPool.Close()
		repo := NewUserAccountStorage(mockPool)
		r1 := pgxmock.NewRows(colsAll).AddRow(id, uid, "alex", &email, &phone, "h", true, now, now)
		r2 := pgxmock.NewRows(colsAll).AddRow(id, uid, "alex", &email, &phone, "h", true, now, now)
		r3 := pgxmock.NewRows(colsAll).AddRow(id, uid, "alex", &email, &phone, "h", true, now, now)
		r4 := pgxmock.NewRows(colsAll).AddRow(id, uid, "alex", &email, &phone, "h", true, now, now)
		mockPool.ExpectQuery("SELECT \\* FROM user_account WHERE email=\\$1;").WithArgs(email).WillReturnRows(r1)
		mockPool.ExpectQuery("SELECT \\* FROM user_account WHERE phone=\\$1;").WithArgs(phone).WillReturnRows(r2)
		mockPool.ExpectQuery("SELECT \\* FROM user_account WHERE username=\\$1;").WithArgs("alex").WillReturnRows(r3)
		mockPool.ExpectQuery("SELECT \\* FROM user_account WHERE uid=\\$1;").WithArgs(uid.String()).WillReturnRows(r4)
		_, err := repo.GetByEmail(context.Background(), email)
		require.NoError(t, err)
		_, err = repo.GetByPhone(context.Background(), phone)
		require.NoError(t, err)
		_, err = repo.GetByUsername(context.Background(), "alex")
		require.NoError(t, err)
		_, err = repo.GetByUid(context.Background(), uid)
		require.NoError(t, err)
	})

	t.Run("Delete", func(t *testing.T) {
		mockPool, _ := pgxmock.NewPool()
		defer mockPool.Close()
		repo := NewUserAccountStorage(mockPool)
		mockPool.ExpectExec("DELETE FROM user_account WHERE id=\\$1").WithArgs(id).WillReturnResult(pgxmock.NewResult("DELETE", 1))
		err := repo.Delete(context.Background(), id)
		require.NoError(t, err)
	})
}
