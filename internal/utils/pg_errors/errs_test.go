package pgerrors

import (
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestMapPgError(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		require.NoError(t, MapPgError(nil))
	})

	t.Run("non pg error", func(t *testing.T) {
		err := errors.New("boom")
		require.Same(t, err, MapPgError(err))
	})

	t.Run("unique email", func(t *testing.T) {
		err := &pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: models.ConstraintUserEmail}
		require.ErrorIs(t, MapPgError(err), models.ErrEmailAlreadyTaken)
	})

	t.Run("unique username", func(t *testing.T) {
		err := &pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: models.ConstraintUserUsername}
		require.ErrorIs(t, MapPgError(err), models.ErrUsernameAlreadyTaken)
	})

	t.Run("unique phone", func(t *testing.T) {
		err := &pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: models.ConstraintUserPhone}
		require.ErrorIs(t, MapPgError(err), models.ErrPhoneAlreadyTaken)
	})

	t.Run("unknown unique", func(t *testing.T) {
		err := &pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: "other"}
		require.ErrorIs(t, MapPgError(err), models.ErrDuplicateEntry)
	})

	t.Run("other pg error", func(t *testing.T) {
		err := &pgconn.PgError{Code: pgerrcode.ForeignKeyViolation}
		require.Same(t, err, MapPgError(err))
	})
}

