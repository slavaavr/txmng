package repos

import (
	"context"

	"github.com/slavaavr/txmng"
)

type SomeRepo interface {
	Do1(ctx context.Context) error
	Do2(ctx context.Context) error
}

type someRepo struct {
	dbm txmng.DBManager
}

func NewSomeRepo(dbm txmng.DBManager) SomeRepo {
	return &someRepo{
		dbm: dbm,
	}
}

func (r *someRepo) Do1(ctx context.Context) error {
	db, err := r.dbm.GetDB(ctx)
	if err != nil {
		return err
	}

	// do some work with db
	_ = db

	return nil
}

func (r *someRepo) Do2(ctx context.Context) error {
	db, err := r.dbm.GetDB(ctx)
	if err != nil {
		return err
	}

	// do some work with db
	_ = db

	return nil
}
