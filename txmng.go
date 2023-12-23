package txmng

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

//go:generate mockgen -source=./txmng.go -destination=./txmng_mock.go -package txmng

// TxManager creates a db connection under the hood, and pass it through a context.
// On the other hand, users of DBManager should use this context to get the DB.
type TxManager interface {
	Tx(opts Opts, f func(ctx context.Context) (Scanner, error)) (Scanner, error)
}

type DBManager interface {
	GetDB(ctx context.Context) (DB, error)
}

type txManager struct {
	dbProvider DBProvider
	cfg        Config

	dbs      sync.Map
	sequence int64
}

func New(p DBProvider, opts ...Option) (txm TxManager, dbm DBManager) {
	cfg := Config{}
	for _, opt := range opts {
		opt(&cfg)
	}

	m := &txManager{
		dbProvider: p,
		cfg:        cfg,
	}

	txm, dbm = m, m
	if m.cfg.retries != nil {
		txm = newTxManagerWithRetries(txm, *m.cfg.retries)
	}

	return txm, dbm
}

type txKey struct{}

func (s *txManager) Tx(opts Opts, f func(ctx context.Context) (Scanner, error)) (Scanner, error) {
	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}

	db, err := s.dbProvider.Tx(opts)
	if err != nil {
		return nil, fmt.Errorf("providing db transaction: %w", err)
	}

	txID := atomic.AddInt64(&s.sequence, 1)
	ctx := context.WithValue(opts.Ctx, txKey{}, txID)

	s.dbs.Store(txID, db)
	defer s.dbs.Delete(txID)

	scanner, err := f(ctx)
	if err != nil {
		if err2 := db.Rollback(); err2 != nil {
			return nil, fmt.Errorf("rollingback the error='%s': %w", err, err2)
		}

		return nil, err
	}

	if err = db.Commit(); err != nil {
		return nil, fmt.Errorf("committing db: %w", err)
	}

	return scanner, nil
}

func (s *txManager) GetDB(ctx context.Context) (DB, error) {
	txID, ok := ctx.Value(txKey{}).(int64)
	if !ok {
		if s.cfg.forbidRawDB {
			return nil, errors.New("raw db is forbidden")
		}

		return s.dbProvider.Raw(), nil
	}

	v, ok := s.dbs.Load(txID)
	if !ok {
		return nil, fmt.Errorf("db was not found by txID='%d'", txID)
	}

	return v.(DB), nil
}
