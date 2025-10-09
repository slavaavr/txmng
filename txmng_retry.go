package txmng

import (
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type Retrier interface {
	Do(func() error) error
}

type defaultRetrier struct {
	retryDelays []time.Duration
	jitter      float64
}

func newDefaultRetrier(
	delays []time.Duration,
	jitter float64, // [0; 1]
) Retrier {
	if jitter < 0 {
		jitter = 0
	} else if jitter > 1 {
		jitter = 1
	}

	return &defaultRetrier{
		retryDelays: append(delays, 0),
		jitter:      jitter,
	}
}

func (s *defaultRetrier) Do(fn func() error) error {
	var err error

	for i := 0; i < len(s.retryDelays); i++ {
		if err = fn(); err != nil && s.needRetry(err) {
			time.Sleep(s.calcDelay(s.retryDelays[i]))
			continue
		}

		break
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
	return s.isNetworkError(err) || s.isSerializationError(err)
}

func (s *defaultRetrier) isNetworkError(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "08000" || pgErr.Code == "08003" || pgErr.Code == "08006"
	}

	return false
}

func (s *defaultRetrier) isSerializationError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "40001" || pgErr.Code == "40P01"
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

func (s *txManagerWithRetrier) RunTx(opts TxOpts, fn func(ctx Context) (Scanner, error)) (Scanner, error) {
	return s.run(func() (Scanner, error) { return s.txm.RunTx(opts, fn) })
}

func (s *txManagerWithRetrier) RunNoTx(opts NoTxOpts, fn func(ctx Context) (Scanner, error)) (Scanner, error) {
	return s.run(func() (Scanner, error) { return s.txm.RunNoTx(opts, fn) })
}

func (s *txManagerWithRetrier) run(fn func() (Scanner, error)) (Scanner, error) {
	var (
		scanner Scanner
		err     error
	)

	err = s.retrier.Do(func() error {
		scanner, err = fn()
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("running tx with retrier: %w", err)
	}

	return scanner, nil
}
