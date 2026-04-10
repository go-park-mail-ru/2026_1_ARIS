package pgerrors

import (
	"errors"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func MapPgError(err error) error {
	if err != nil {

		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) {
			return err
		}

		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return mapUniqueViolation(pgErr.ConstraintName)
		default:
			return err
		}
	}
	return nil
}

func mapUniqueViolation(constraint string) error {
	switch constraint {
	case models.ConstraintUserEmail:
		return models.ErrEmailAlreadyTaken
	case models.ConstraintUserUsername:
		return models.ErrUsernameAlreadyTaken
	case models.ConstraintUserPhone:
		return models.ErrPhoneAlreadyTaken
	default:
		return models.ErrDuplicateEntry
	}
}
