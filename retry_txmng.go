package txmng

import (
	"context"
	"fmt"
)

type txManagerWithRetrier struct {
	txm     TxManager
	retrier Retrier
}

func newTxManagerWithRetrier(m TxManager, r Retrier) TxManager {
	return &txManagerWithRetrier{
		txm:     m,
		retrier: r,
	}
}

func (s *txManagerWithRetrier) RunTx(opts TxOpts, fn func(ctx TxContext) (Result, error)) (Result, error) {
	return s.run(opts.Ctx, s.retrier, func() (Result, error) { return s.txm.RunTx(opts, fn) })
}

func (s *txManagerWithRetrier) RunNoTx(opts NoTxOpts, fn func(ctx NoTxContext) (Result, error)) (Result, error) {
	return s.run(opts.Ctx, s.retrier, func() (Result, error) { return s.txm.RunNoTx(opts, fn) })
}

func (s *txManagerWithRetrier) run(ctx context.Context, retrier Retrier, fn func() (Result, error)) (Result, error) {
	var (
		res Result
		err error
	)

	if ctx == nil {
		ctx = context.Background()
	}

	err = retrier.Do(ctx, func() error {
		res, err = fn()
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("retry: %w", err)
	}

	return res, nil
}
