package retrycond

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func Or(fs ...func(error) bool) func(error) bool {
	return func(err error) bool {
		for _, f := range fs {
			if f(err) {
				return true
			}
		}

		return false
	}
}

func PGXSerializationError(err error) bool {
	err = getBaseError(err)
	if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "40001" {
		return true
	}

	return false
}

func getBaseError(err error) error {
	var prevError error

	for err != nil {
		prevError = err
		err = errors.Unwrap(err)
	}

	return prevError
}
