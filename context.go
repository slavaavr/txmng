package txmng

import "context"

type Context interface {
	context.Context
	getTxID() (int64, bool)
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

func (ctx contextImpl) getTxID() (_ int64, ok bool) {
	v, ok := ctx.Value(txKey{}).(int64)
	return v, ok
}
