package txmng

import (
	"context"
	"fmt"
	"time"
)

type txManagerWithReties struct {
	m       TxManager
	retries retries
}

func newTxManagerWithReties(m TxManager, r retries) TxManager {
	return &txManagerWithReties{
		m:       m,
		retries: r,
	}
}

func (s *txManagerWithReties) Tx(opts Opts, f func(ctx context.Context) (Scanner, error)) (Scanner, error) {
	count := s.retries.count

RETRY:
	scanner, err := s.m.Tx(opts, f)
	if err != nil {
		count--
		if count > 0 && s.retryNeeded(err) {
			time.Sleep(s.retries.interval)
			goto RETRY
		}

		return nil, fmt.Errorf("executing tx with retries: %w", err)
	}

	return scanner, nil
}

func (s *txManagerWithReties) retryNeeded(err error) bool {
	return s.retries.need != nil && s.retries.need(err)
}
