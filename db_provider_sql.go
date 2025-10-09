package txmng

import (
	"context"
	"database/sql"
)

//go:generate mockgen -source=./db_provider_sql.go -destination=./db_provider_sql_mock.go -package txmng

type StdSQL interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	Commit() error
	Rollback() error
}

type sqlProvider struct {
	db    *sql.DB
	rawDB StdSQL
}

type sqlAdapter struct {
	*sql.DB
}

func (s *sqlAdapter) Commit() error   { return ErrCommitNotSupported }
func (s *sqlAdapter) Rollback() error { return ErrRollbackNotSupported }

func NewSQLProvider(db *sql.DB) DBProvider[StdSQL] {
	return &sqlProvider{
		db:    db,
		rawDB: &sqlAdapter{db},
	}
}

func (s *sqlProvider) BeginTx(opts TxOpts) (Tx[StdSQL], error) {
	tx, err := s.db.BeginTx(opts.Ctx, &sql.TxOptions{
		Isolation: opts.Isolation.toSQL(),
		ReadOnly:  opts.ReadOnly,
	})
	if err != nil {
		return nil, err
	}

	return newTx(
		func() StdSQL { return tx },
		func(ctx context.Context) error { return tx.Commit() },
		func(ctx context.Context) error { return tx.Rollback() },
	), nil
}

func (s *sqlProvider) GetDB(_ NoTxOpts) StdSQL { return s.rawDB }
