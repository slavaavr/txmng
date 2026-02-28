package txmng

import (
	"context"
)

type commonContext interface {
	rawCtx() context.Context
	getDB() any
	close()
}

type commonContextImpl struct {
	context.Context
	db      any
	closeCh chan struct{}
}

func newCommonContext(ctx context.Context, db any) *commonContextImpl {
	return &commonContextImpl{
		Context: ctx,
		db:      db,
		closeCh: make(chan struct{}),
	}
}

var _ commonContext = (*commonContextImpl)(nil)

func (s *commonContextImpl) withDB(db any) *commonContextImpl {
	return &commonContextImpl{
		Context: s.Context,
		db:      db,
		closeCh: s.closeCh,
	}
}

func (s *commonContextImpl) getDB() any {
	select {
	case <-s.closeCh:
		panic(errInvalidContext)

	default:
	}

	return s.db
}

func (s *commonContextImpl) rawCtx() context.Context { return s.Context }
func (s *commonContextImpl) close()                  { close(s.closeCh) }
