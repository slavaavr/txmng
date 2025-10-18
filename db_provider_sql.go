package txmng

import (
	"context"
	"database/sql"
)

type StdSQL interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	Commit() error
	Rollback() error
}

type stdSQLProvider struct {
	db    *sql.DB
	rawDB StdSQL
}

type stdSQLAdapter struct {
	*sql.DB
}

func (s *stdSQLAdapter) Commit() error   { panic(errCommitNotSupported) }
func (s *stdSQLAdapter) Rollback() error { panic(errRollbackNotSupported) }

func NewStdSQLProvider(db *sql.DB) DBProvider[StdSQL] {
	return &stdSQLProvider{
		db: db,
		rawDB: &stdSQLAdapter{
			DB: db,
		},
	}
}

func (s *stdSQLProvider) BeginTx(opts TxOpts) (Tx[StdSQL], error) {
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

func (s *stdSQLProvider) GetDB(_ NoTxOpts) StdSQL { return s.rawDB }
