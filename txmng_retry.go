package txmng

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type Retrier interface {
	Do(func() error) error
}

type defaultRetrier struct {
	retryDelays []time.Duration
}

func newDefaultRetrier(retryDelays []time.Duration) Retrier {
	return &defaultRetrier{retryDelays: append(retryDelays, 0)}
}

func (s *defaultRetrier) Do(fn func() error) error {
	var err error

	for i := 0; i < len(s.retryDelays); i++ {
		if err = fn(); err != nil && s.needRetry(err) {
			time.Sleep(s.retryDelays[i])
			continue
		}

		break
	}

	return err
}

func (s *defaultRetrier) needRetry(err error) bool {
	return s.isNetworkError(err) || s.isSerializationError(err)
}

func (s *defaultRetrier) isSerializationError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "40001" || pgErr.Code == "40P01"
	}

	return false
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

type managerWithRetrier struct {
	mng     TxManager
	retrier Retrier
}

func newManagerWithRetrier(m TxManager, r Retrier) TxManager {
	return &managerWithRetrier{
		mng:     m,
		retrier: r,
	}
}

func (s *managerWithRetrier) RunTx(opts TxOpts, fn func(ctx Context) (Scanner, error)) (Scanner, error) {
	return s.run(func() (Scanner, error) { return s.mng.RunTx(opts, fn) })
}

func (s *managerWithRetrier) RunNoTx(opts NoTxOpts, fn func(ctx Context) (Scanner, error)) (Scanner, error) {
	return s.run(func() (Scanner, error) { return s.mng.RunNoTx(opts, fn) })
}

func (s *managerWithRetrier) run(fn func() (Scanner, error)) (Scanner, error) {
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
