package txmng

import "context"

type Context interface {
	context.Context
	getDB() any
	close()
}

type contextImpl struct {
	context.Context
	db     any
	closed bool
}

func newContext(ctx context.Context, db any) Context {
	return &contextImpl{
		Context: ctx,
		db:      db,
		closed:  false,
	}
}

func (s *contextImpl) getDB() any {
	if s.closed {
		panic(errInvalidContext)
	}

	return s.db
}

func (s *contextImpl) close() { s.closed = true }
