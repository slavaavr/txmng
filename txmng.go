package txmng

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

type TxManager interface {
	RunTx(opts TxOpts, fn func(ctx Context) (Scanner, error)) (Scanner, error)
	RunNoTx(opts NoTxOpts, fn func(ctx Context) (Scanner, error)) (Scanner, error)
}

type DBManager[T any] interface {
	GetDB(ctx Context) (T, error)
	MustGetDB(ctx Context) T
}

type manager[T any] struct {
	dbProvider DBProvider[T]

	dbs      sync.Map
	sequence int64
}

func New[T any](p DBProvider[T], opts ...Option) (txm TxManager, dbm DBManager[T]) {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}

	m := &manager[T]{
		dbProvider: p,
	}

	txm, dbm = m, m
	if cfg.retrier != nil {
		txm = newTxManagerWithRetrier(txm, cfg.retrier)
	}

	return txm, dbm
}

func (s *manager[T]) RunTx(opts TxOpts, fn func(ctx Context) (Scanner, error)) (_ Scanner, err error) {
	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}

	tx, err := s.dbProvider.BeginTx(opts)
	if err != nil {
		return nil, fmt.Errorf("beginning tx: %w", err)
	}

	id := s.nextID()
	s.dbs.Store(id, tx.GetDB())

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}

		if err != nil {
			if err2 := tx.Rollback(opts.Ctx); err2 != nil {
				err = fmt.Errorf("rolling back the error='%s': %w", err, err2)
			}
		}

		s.dbs.Delete(id)
	}()

	scanner, err := fn(newContext(opts.Ctx, id))
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(opts.Ctx); err != nil {
		return nil, fmt.Errorf("committing tx: %w", err)
	}

	return scanner, nil
}

func (s *manager[T]) RunNoTx(opts NoTxOpts, fn func(ctx Context) (Scanner, error)) (_ Scanner, err error) {
	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}

	db := s.dbProvider.GetDB(opts)
	id := s.nextID()
	s.dbs.Store(id, db)

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}

		s.dbs.Delete(id)
	}()

	return fn(newContext(opts.Ctx, id))
}

func (s *manager[T]) GetDB(ctx Context) (T, error) {
	id := ctx.getID()

	v, ok := s.dbs.Load(id)
	if !ok {
		var empty T
		return empty, errDBNotFound
	}

	return v.(T), nil
}

func (s *manager[T]) MustGetDB(ctx Context) T {
	db, err := s.GetDB(ctx)
	if err != nil {
		panic(err)
	}

	return db
}

func (s *manager[T]) nextID() int64 {
	return atomic.AddInt64(&s.sequence, 1)
}
