package txmng

import "context"

type STxManager[S any] interface {
	RunTx(opts TxOpts, fn func(ctx STxContext[S]) (Result, error)) (Result, error)
	RunNoTx(opts NoTxOpts, fn func(ctx SNoTxContext[S]) (Result, error)) (Result, error)
}

type SDBManager[S, T any] interface {
	GetDB(ctx SContext[S]) (db T, rawCtx context.Context)
}

type sManager[S, T any] struct {
	txm TxManager
}

func NewScoped[S, T any](p DBProvider[T], opts ...Option) (txm STxManager[S], dbm SDBManager[S, T]) {
	t, _ := New(p, opts...)
	m := &sManager[S, T]{txm: t}

	return m, m
}

func (s *sManager[S, T]) RunTx(opts TxOpts, fn func(ctx STxContext[S]) (Result, error)) (_ Result, err error) {
	return s.txm.RunTx(opts, func(ctx TxContext) (Result, error) { return fn(newSTxContext[S](ctx)) })
}

func (s *sManager[S, T]) RunNoTx(opts NoTxOpts, fn func(ctx SNoTxContext[S]) (Result, error)) (_ Result, err error) {
	return s.txm.RunNoTx(opts, func(ctx NoTxContext) (Result, error) { return fn(newSNoTxContext[S](ctx)) })
}

func (s *sManager[S, T]) GetDB(ctx SContext[S]) (db T, rawCtx context.Context) {
	return ctx.getDB().(T), ctx.rawCtx()
}
