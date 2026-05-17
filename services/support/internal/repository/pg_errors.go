package repository

import (
	"errors"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/xerrors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func MapPgError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return xerrors.NoRowsAffected
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return xerrors.AllreadyExists
		case "23503":
			return xerrors.NoRowsAffected
		}
	}
	return err
}
