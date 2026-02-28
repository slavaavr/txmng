package txmng

import (
	"context"
	"fmt"
)

type TxManager interface {
	RunTx(opts TxOpts, fn func(ctx TxContext) (Result, error)) (Result, error)
	RunNoTx(opts NoTxOpts, fn func(ctx NoTxContext) (Result, error)) (Result, error)
}

type DBManager[T any] interface {
	GetDB(ctx Context) (db T, rawCtx context.Context)
}

type manager[T any] struct {
	dbProvider        DBProvider[T]
	defaultFallbackDB T
	dynamicFallbackDB bool
}

func New[T any](p DBProvider[T], opts ...Option) (txm TxManager, dbm DBManager[T]) {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	m := &manager[T]{
		dbProvider:        p,
		defaultFallbackDB: p.GetDB(NoTxOpts{}),
		dynamicFallbackDB: cfg.dynamicFallbackDB,
	}

	txm, dbm = m, m
	if cfg.retrier != nil {
		txm = newTxManagerWithRetrier(txm, cfg.retrier)
	}

	return txm, dbm
}

func (s *manager[T]) RunTx(opts TxOpts, fn func(ctx TxContext) (Result, error)) (_ Result, err error) {
	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}

	tx, err := s.dbProvider.BeginTx(opts)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	fallbackDB := s.defaultFallbackDB
	if s.dynamicFallbackDB {
		fallbackDB = s.dbProvider.GetDB(newNoTxOpts(opts.Ctx, opts.Ext))
	}

	newCtx := newTxContext(opts.Ctx, tx.GetDB(), fallbackDB)

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}

		if err != nil {
			if err2 := tx.Rollback(opts.Ctx); err2 != nil {
				err = fmt.Errorf("rollback tx (%s): %w", err2, err)
			}
		}

		newCtx.close()
	}()

	res, err := fn(newCtx)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(opts.Ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return res, nil
}

func (s *manager[T]) RunNoTx(opts NoTxOpts, fn func(ctx NoTxContext) (Result, error)) (_ Result, err error) {
	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}

	newCtx := newNoTxContext(opts.Ctx, s.dbProvider.GetDB(opts))

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}

		newCtx.close()
	}()

	return fn(newCtx)
}

func (s *manager[T]) GetDB(ctx Context) (db T, rawCtx context.Context) {
	return ctx.getDB().(T), ctx.rawCtx()
}
