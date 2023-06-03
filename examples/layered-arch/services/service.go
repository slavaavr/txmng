package services

import (
	"context"
	"fmt"

	"github.com/slavaavr/txmng"
	"github.com/slavaavr/txmng/examples/layered-arch/repos"
)

type SomeService interface {
	Do(ctx context.Context) (int, error)
}

type someService struct {
	txm  txmng.TxManager
	repo repos.SomeRepo
}

func NewSomeService(
	txm txmng.TxManager,
	repo repos.SomeRepo,
) SomeService {
	return &someService{
		txm:  txm,
		repo: repo,
	}
}

func (s *someService) Do(ctx context.Context) (int, error) {
	opts := txmng.Opts{
		Ctx:       ctx,
		Isolation: txmng.LevelDefault,
		ReadOnly:  false,
	}

	scanner, err := s.txm.Tx(opts, func(ctx context.Context) (txmng.Scanner, error) {
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

	return res, nil
}
