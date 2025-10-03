package txmng

import (
	"context"
	"errors"
)

var (
	ErrCommitNotSupported   = errors.New("commit is not supported")
	ErrRollbackNotSupported = errors.New("rollback is not supported")
)

//go:generate mockgen -source=./db_provider.go -destination=./db_provider_mock.go -package txmng

type Tx[T any] interface {
	GetDB() T
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type DBProvider[T any] interface {
	BeginTx(opts Opts) (Tx[T], error)
}

type txImpl[T any] struct {
	getDB    func() T
	commit   func(ctx context.Context) error
	rollback func(ctx context.Context) error
}

func newTx[T any](
	getDB func() T,
	commit func(ctx context.Context) error,
	rollback func(ctx context.Context) error,
) Tx[T] {
	return &txImpl[T]{
		getDB:    getDB,
		commit:   commit,
		rollback: rollback,
	}
}

func (s *txImpl[T]) GetDB() T                           { return s.getDB() }
func (s *txImpl[T]) Commit(ctx context.Context) error   { return s.commit(ctx) }
func (s *txImpl[T]) Rollback(ctx context.Context) error { return s.rollback(ctx) }
