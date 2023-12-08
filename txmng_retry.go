package txmng

import (
	"context"
	"errors"
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
	var (
		err   = errors.New("executing tx with retries")
		count = s.retries.count
	)

RETRY:
	scanner, err2 := s.m.Tx(opts, f)
	if err2 != nil {
		time.Sleep(s.retries.interval)
		err = fmt.Errorf("%s: %w", err, err2)
		count--

		if count > 0 {
			goto RETRY
		}

		return nil, err
	}

	return scanner, nil
}
