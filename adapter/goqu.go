package adapter

import (
	"context"
	"database/sql"

	"github.com/doug-martin/goqu/v9"
	//_ "github.com/doug-martin/goqu/v9/dialect/postgres"

	"github.com/slavaavr/txmng"
)

type goquAdapter struct {
	db txmng.DB
}

func NewGoqu(db txmng.DB, dialect string) *goqu.Database {
	return goqu.New(dialect, &goquAdapter{db})
}

func NewPostgresGoqu(db txmng.DB) *goqu.Database {
	return goqu.New("postgres", &goquAdapter{db})
}

func (a *goquAdapter) Begin() (*sql.Tx, error) {
	panic("Begin method is not supported")
}

func (a *goquAdapter) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	panic("BeginTx method is not supported")
}

func (a *goquAdapter) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return a.db.ExecContext(ctx, query, args...)
}

func (a *goquAdapter) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return a.db.PrepareContext(ctx, query)
}

func (a *goquAdapter) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return a.db.QueryContext(ctx, query, args...)
}

func (a *goquAdapter) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return a.db.QueryRowContext(ctx, query, args...)
}
