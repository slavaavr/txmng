package txmng

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type Retrier interface {
	Do(ctx context.Context, fn func() error) error
}

type defaultRetrier struct {
	jitter      float64
	retryDelays []time.Duration
}

func newDefaultRetrier(
	jitter float64, // [0; 1]
	delays []time.Duration,
) Retrier {
	if jitter < 0 {
		jitter = 0
	} else if jitter > 1 {
		jitter = 1
	}

	return &defaultRetrier{
		jitter:      jitter,
		retryDelays: append(delays, 0),
	}
}

func (s *defaultRetrier) Do(ctx context.Context, fn func() error) error {
	var err error

	for i := 0; i < len(s.retryDelays); i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err = fn(); err == nil || !s.needRetry(err) {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-time.After(s.calcDelay(s.retryDelays[i])):
			// continue to next retry
		}
	}

	return err
}

func (s *defaultRetrier) calcDelay(base time.Duration) time.Duration {
	baseNS := base.Nanoseconds()
	offset := int64(float64(baseNS) * s.jitter)

	return time.Duration(s.randInt(baseNS-offset, baseNS+offset))
}

func (s *defaultRetrier) randInt(a, b int64) int64 { return rand.Int64N(b-a+1) + a }

func (s *defaultRetrier) needRetry(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "08000", "08003", "08006":
			// network errors
			return true

		case "40001", "40P01", "55P03":
			// serialization errors
			return true
		}
	}

	return false
}

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

func (s *txManagerWithRetrier) RunTx(opts TxOpts, fn func(ctx Context) (Result, error)) (Result, error) {
	return s.run(opts.Ctx, func() (Result, error) { return s.txm.RunTx(opts, fn) })
}

func (s *txManagerWithRetrier) RunNoTx(opts NoTxOpts, fn func(ctx Context) (Result, error)) (Result, error) {
	return s.run(opts.Ctx, func() (Result, error) { return s.txm.RunNoTx(opts, fn) })
}

func (s *txManagerWithRetrier) run(ctx context.Context, fn func() (Result, error)) (Result, error) {
	var (
		res Result
		err error
	)

	if ctx == nil {
		ctx = context.Background()
	}

	err = s.retrier.Do(ctx, func() error {
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
