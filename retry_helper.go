package txmng

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func isPGXSerializationError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "40001" {
		return true
	}

	return false
}
