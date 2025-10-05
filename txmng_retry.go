package txmng

import (
	"context"
	"errors"
	"fmt"
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

func (s *defaultRetrier) Do(f func() error) error {
	var err error

	for i := 0; i < len(s.retryDelays); i++ {
		if err = f(); err != nil && s.isSerializationError(err) {
			time.Sleep(s.retryDelays[i])
			continue
		}

		break
	}

	return err
}

func (s *defaultRetrier) isSerializationError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "40001" || pgErr.Code == "40P01"
	}

	return false
}

type managerWithRetries struct {
	mng     TxManager
	retrier Retrier
}

func newManagerWithRetries(m TxManager, r Retrier) TxManager {
	return &managerWithRetries{
		mng:     m,
		retrier: r,
	}
}

func (s *managerWithRetries) RunTx(opts Opts, f func(ctx Context) (Scanner, error)) (Scanner, error) {
	return s.run(func() (Scanner, error) { return s.mng.RunTx(opts, f) })
}

func (s *managerWithRetries) RunNoTx(ctx context.Context, f func(ctx Context) (Scanner, error)) (Scanner, error) {
	return s.run(func() (Scanner, error) { return s.mng.RunNoTx(ctx, f) })
}

func (s *managerWithRetries) run(f func() (Scanner, error)) (Scanner, error) {
	var (
		scanner Scanner
		err     error
	)

	err = s.retrier.Do(func() error {
		scanner, err = f()
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("running tx with retries: %w", err)
	}

	return scanner, nil
}
