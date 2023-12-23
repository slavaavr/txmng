package txmng

import (
	"context"
	"fmt"
	"time"
)

type txManagerWithRetries struct {
	m       TxManager
	retries retries
}

func newTxManagerWithRetries(m TxManager, r retries) TxManager {
	return &txManagerWithRetries{
		m:       m,
		retries: r,
	}
}

func (s *txManagerWithRetries) Tx(opts Opts, f func(ctx context.Context) (Scanner, error)) (Scanner, error) {
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

func (s *txManagerWithRetries) retryNeeded(err error) bool {
	return s.retries.need != nil && s.retries.need(err)
}
