package txmng

import "context"

type Context interface {
	context.Context
	getID() int64
}

type idKey struct{}

type contextImpl struct {
	context.Context
}

func newContext(ctx context.Context, id int64) Context {
	return contextImpl{
		Context: context.WithValue(ctx, idKey{}, id),
	}
}

func (ctx contextImpl) getID() int64 {
	return ctx.Value(idKey{}).(int64)
}
