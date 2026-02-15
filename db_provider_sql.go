package txmng

import (
	"context"
	"database/sql"
)

type SQLDB interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type sqlDB struct {
	db *sql.DB
}

func NewSQLDB(db *sql.DB) DBProvider[SQLDB] {
	return &sqlDB{
		db: db,
	}
}

func (s *sqlDB) BeginTx(opts TxOpts) (Tx[SQLDB], error) {
	tx, err := s.db.BeginTx(opts.Ctx, &sql.TxOptions{
		Isolation: opts.Isolation.toSQL(),
		ReadOnly:  opts.ReadOnly,
	})
	if err != nil {
		return nil, err
	}

	return newTx(
		func() SQLDB { return tx },
		func(ctx context.Context) error { return tx.Commit() },
		func(ctx context.Context) error { return tx.Rollback() },
	), nil
}

func (s *sqlDB) GetDB(_ NoTxOpts) SQLDB { return s.db }
