package repo

import (
	"context"

	"github.com/slavaavr/txmng"
)

type SomeRepo interface {
	Do1(ctx txmng.Context) error
	Do2(ctx txmng.Context) error
}

type someRepo struct {
	dbm txmng.DBManager[txmng.SQLDB]
}

func NewSomeRepo(dbm txmng.DBManager[txmng.SQLDB]) SomeRepo {
	return &someRepo{
		dbm: dbm,
	}
}

func (r *someRepo) Do1(ctx txmng.Context) error {
	// do some work with db
	db, rawCtx := r.dbm.GetDB(ctx)
	foo(rawCtx, db)

	return nil
}

func (r *someRepo) Do2(ctx txmng.Context) error {
	// do some work with db
	_, _ = r.dbm.GetDB(ctx)
	return nil
}

func foo(ctx context.Context, db txmng.SQLDB) {}
