package txmng

import (
	"context"
)

type SContext[S any] interface {
	commonContext
	scopedContextMarker(S)
}

type STxContext[S any] interface {
	SContext[S]
	NoTxContext() SNoTxContext[S]
	scopedTxContextMarker()
}

type SNoTxContext[S any] interface {
	SContext[S]
	context.Context
	scopedNoTxContextMarker()
}

var _ SContext[int] = (*sContextImpl[int])(nil)
var _ STxContext[int] = (*sContextImpl[int])(nil)
var _ SNoTxContext[int] = (*sContextImpl[int])(nil)

type sContextImpl[S any] struct {
	*contextImpl
}

func newSTxContext[S any](ctx TxContext) *sContextImpl[S] {
	return &sContextImpl[S]{ctx.(*contextImpl)}
}

func newSNoTxContext[S any](ctx NoTxContext) *sContextImpl[S] {
	return &sContextImpl[S]{ctx.(*contextImpl)}
}

func (s *sContextImpl[S]) NoTxContext() SNoTxContext[S] {
	return &sContextImpl[S]{s.contextImpl.NoTxContext().(*contextImpl)}
}

// nolint:unused
func (s *sContextImpl[S]) scopedTxContextMarker()   {}
func (s *sContextImpl[S]) scopedNoTxContextMarker() {}
func (s *sContextImpl[S]) scopedContextMarker(S)    {}
