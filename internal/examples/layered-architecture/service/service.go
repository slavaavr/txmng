package service

import (
	"context"
	"fmt"

	"github.com/slavaavr/txmng"
	"github.com/slavaavr/txmng/internal/examples/layered-architecture/repo"
)

type SomeService interface {
	Do(ctx context.Context) (int, error)
}

type someService struct {
	txm  txmng.TxManager
	repo repo.SomeRepo
}

func NewSomeService(
	txm txmng.TxManager,
	repo repo.SomeRepo,
) SomeService {
	return &someService{
		txm:  txm,
		repo: repo,
	}
}

func (s *someService) Do(ctx context.Context) (int, error) {
	txOpts := txmng.TxOpts{
		Ctx:       ctx,
		Isolation: txmng.LevelDefault,
		ReadOnly:  false,
		Ext:       nil,
	}

	scanner, err := s.txm.RunTx(txOpts, func(ctx txmng.Context) (txmng.Scanner, error) {
		if err := s.repo.Do1(ctx); err != nil {
			return nil, fmt.Errorf("executing 1 query: %w", err)
		}

		if err := s.repo.Do2(ctx); err != nil {
			return nil, fmt.Errorf("executing 2 query: %w", err)
		}

		return txmng.Values(42), nil
	})
	if err != nil {
		return 0, fmt.Errorf("executing tx: %w", err)
	}

	var res int
	if err := scanner.Scan(&res); err != nil {
		return 0, fmt.Errorf("scanning result: %w", err)
	}

	// -----------------------------

	noTxOpts := txmng.NoTxOpts{
		Ctx: ctx,
		Ext: nil,
	}

	_, err = s.txm.RunNoTx(noTxOpts, func(ctx txmng.Context) (txmng.Scanner, error) {
		if err := s.repo.Do3(ctx); err != nil {
			return nil, fmt.Errorf("executing 3 query: %w", err)
		}

		return nil, nil
	})
	if err != nil {
		return 0, fmt.Errorf("executing notx: %w", err)
	}

	return res, nil
}
