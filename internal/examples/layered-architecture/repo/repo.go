package repo

import (
	"github.com/slavaavr/txmng"
)

type SomeRepo interface {
	Do1(ctx txmng.Context) error
	Do2(ctx txmng.Context) error
}

type someRepo struct {
	dbm txmng.DBManager[txmng.StdSQL]
}

func NewSomeRepo(dbm txmng.DBManager[txmng.StdSQL]) SomeRepo {
	return &someRepo{
		dbm: dbm,
	}
}

func (r *someRepo) Do1(ctx txmng.Context) error {
	db := r.dbm.GetDB(ctx)

	// do some work with db
	_ = db

	return nil
}

func (r *someRepo) Do2(ctx txmng.Context) error {
	db := r.dbm.GetDB(ctx)

	// do some work with db
	_ = db

	return nil
}
