package txmng

import "context"

type Context interface {
	context.Context
	getTxID() int64
}

type txKey struct{}

type contextImpl struct {
	context.Context
}

func newContext(ctx context.Context, txID int64) Context {
	return contextImpl{
		Context: context.WithValue(ctx, txKey{}, txID),
	}
}

func (ctx contextImpl) getTxID() int64 {
	return ctx.Value(txKey{}).(int64)
}
