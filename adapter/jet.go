package adapter

import (
	"context"
	"database/sql"

	"github.com/go-jet/jet/v2/qrm"

	"github.com/slavaavr/txmng"
)

type jetAdapter struct {
	db txmng.DB
}

func NewJet(db txmng.DB) qrm.DB {
	return &jetAdapter{db}
}

func (a *jetAdapter) Exec(query string, args ...interface{}) (sql.Result, error) {
	return a.db.ExecContext(context.Background(), query, args...)
}

func (a *jetAdapter) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return a.db.ExecContext(ctx, query, args...)
}

func (a *jetAdapter) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return a.db.QueryContext(context.Background(), query, args...)
}

func (a *jetAdapter) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return a.db.QueryContext(ctx, query, args...)
}
