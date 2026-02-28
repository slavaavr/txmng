package txmng

import (
	"context"
)

type Context interface {
	commonContext
	contextMarker()
}

type TxContext interface {
	Context
	NoTxContext() NoTxContext
	txContextMarker()
}

type NoTxContext interface {
	Context
	context.Context
	noTxContextMarker()
}

var _ Context = (*contextImpl)(nil)
var _ TxContext = (*contextImpl)(nil)
var _ NoTxContext = (*contextImpl)(nil)

type contextImpl struct {
	*commonContextImpl
	fallbackDB any
}

func newTxContext(ctx context.Context, primaryDB, fallbackDB any) *contextImpl {
	return &contextImpl{
		commonContextImpl: newCommonContext(ctx, primaryDB),
		fallbackDB:        fallbackDB,
	}
}

func newNoTxContext(ctx context.Context, primaryDB any) *contextImpl {
	// Do not use fallbackDB: NoTxContext() is defined only on TxContext
	return &contextImpl{
		commonContextImpl: newCommonContext(ctx, primaryDB),
		fallbackDB:        nil,
	}
}

func (s *contextImpl) NoTxContext() NoTxContext {
	return &contextImpl{
		commonContextImpl: s.commonContextImpl.withDB(s.fallbackDB),
		fallbackDB:        s.fallbackDB,
	}
}

func (s *contextImpl) txContextMarker()   {}
func (s *contextImpl) noTxContextMarker() {}
func (s *contextImpl) contextMarker()     {}
