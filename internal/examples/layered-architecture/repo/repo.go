package repo

import (
	"context"

	"github.com/slavaavr/txmng"
)

type SomeRepo interface {
	Do1(ctx txmng.Context) error
	Do2(ctx txmng.Context) error
	Do3(ctx txmng.Context) error
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
	_ = r.dbm.GetDB(ctx)
	return nil
}

func (r *someRepo) Do2(ctx txmng.Context) error {
	// do some work with db
	_ = r.dbm.GetDB(ctx)
	return nil
}

func (r *someRepo) Do3(ctx txmng.Context) error {
	// do some work with db
	return someJob(ctx, r.dbm.GetDB(ctx))
}

func someJob(ctx context.Context, db txmng.SQLDB) error { return nil }
