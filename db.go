package txmng

import (
	"context"
	"database/sql"
	"errors"
)

//go:generate mockgen -destination=./db_mock.go -package txmng github.com/slavaavr/txmng DB

// DB is the common database interface
type DB interface {
	dbRaw
	dbTx
}

type dbRaw interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type dbTx interface {
	Commit() error
	Rollback() error
}

type dbRawAdapter struct{ db dbRaw }

func newDBRawAdapter(db dbRaw) DB { return &dbRawAdapter{db: db} }

func (a *dbRawAdapter) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return a.db.ExecContext(ctx, query, args...)
}

func (a *dbRawAdapter) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return a.db.PrepareContext(ctx, query)
}

func (a *dbRawAdapter) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return a.db.QueryContext(ctx, query, args...)
}

func (a *dbRawAdapter) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return a.db.QueryRowContext(ctx, query, args...)
}

func (a *dbRawAdapter) Commit() error {
	return errors.New("unable to commit on raw db")
}

func (a *dbRawAdapter) Rollback() error {
	return errors.New("unable to rollback on raw db")
}
