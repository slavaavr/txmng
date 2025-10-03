package txmng

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

//go:generate mockgen -source=./txmng.go -destination=./txmng_mock.go -package txmng

// TxManager creates a db connection under the hood and passes it through a context.
// On the other hand, users of DBManager should use this context to get the DB.
type TxManager interface {
	Tx(opts Opts, f func(ctx Context) (Scanner, error)) (Scanner, error)
}

type DBManager[T any] interface {
	GetDB(ctx Context) (T, error)
}

type manager[T any] struct {
	dbProvider DBProvider[T]
	cfg        Config

	dbs      sync.Map
	sequence int64
}

func New[T any](p DBProvider[T], opts ...Option) (txm TxManager, dbm DBManager[T]) {
	cfg := Config{}
	for _, opt := range opts {
		opt(&cfg)
	}

	m := &manager[T]{
		dbProvider: p,
		cfg:        cfg,
	}

	txm, dbm = m, m
	if m.cfg.retrier != nil {
		txm = newManagerWithRetries(txm, m.cfg.retrier)
	}

	return txm, dbm
}

func (s *manager[T]) Tx(opts Opts, f func(ctx Context) (Scanner, error)) (_ Scanner, err error) {
	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}

	tx, err := s.dbProvider.BeginTx(opts)
	if err != nil {
		return nil, fmt.Errorf("beginning tx: %w", err)
	}

	txID := atomic.AddInt64(&s.sequence, 1)
	ctx := newContext(opts.Ctx, txID)

	s.dbs.Store(txID, tx.GetDB())
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}

		if err != nil {
			if err2 := tx.Rollback(opts.Ctx); err2 != nil {
				err = fmt.Errorf("rolling back the error='%s': %w", err, err2)
			}
		}

		s.dbs.Delete(txID)
	}()

	scanner, err := f(ctx)
	if err != nil {
		return scanner, err
	}

	if err = tx.Commit(opts.Ctx); err != nil {
		return nil, fmt.Errorf("committing tx: %w", err)
	}

	return scanner, nil
}

func (s *manager[T]) GetDB(ctx Context) (T, error) {
	txID, ok := ctx.getTxID()
	if !ok {
		tx, err := s.dbProvider.BeginTx(Opts{
			Ctx:      ctx,
			useRawDB: true,
		})
		if err != nil {
			return empty[T](), fmt.Errorf("beginning raw db: %w", err)
		}

		return tx.GetDB(), nil
	}

	v, ok := s.dbs.Load(txID)
	if !ok {
		return empty[T](), fmt.Errorf("db not found with txID='%d'", txID)
	}

	return v.(T), nil
}

func empty[T any]() T {
	var t T
	return t
}
