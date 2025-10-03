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
	rawDB Tx[StdSQL]
}

type rawSqlAdapter struct {
	*sql.DB
}

func (s *rawSqlAdapter) Commit() error   { return ErrCommitNotSupported }
func (s *rawSqlAdapter) Rollback() error { return ErrRollbackNotSupported }

func NewSQLProvider(db *sql.DB) DBProvider[StdSQL] {
	return &sqlProvider{
		db: db,
		rawDB: newTx[StdSQL](
			func() StdSQL { return &rawSqlAdapter{db} },
			func(ctx context.Context) error { return ErrCommitNotSupported },
			func(ctx context.Context) error { return ErrRollbackNotSupported },
		),
	}
}

func (s *sqlProvider) BeginTx(opts Opts) (Tx[StdSQL], error) {
	if opts.UseRawDB() {
		return s.rawDB, nil
	}

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
