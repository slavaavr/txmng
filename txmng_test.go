package txmng

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var someErr = errors.New("some error")

func TestManager_RunTx(t *testing.T) {
	opts := TxOpts{
		Ctx:       context.Background(),
		Isolation: LevelDefault,
		ReadOnly:  false,
		Ext:       nil,
	}

	cases := []struct {
		name              string
		prepareDBProvider func(m *MockDBProvider[PGX], tx *MockTx[PGX])
		opts              TxOpts
		job               func(ctx Context) (Scanner, error)
		expected          Scanner
		expectedErr       error
	}{
		{
			name: "valid example",
			prepareDBProvider: func(m *MockDBProvider[PGX], tx *MockTx[PGX]) {
				tx.EXPECT().
					GetDB().
					Return(nil)

				tx.EXPECT().
					Commit(mock.Anything).
					Return(nil)

				m.EXPECT().
					BeginTx(opts).
					Return(tx, nil)
			},
			opts: opts,
			job: func(ctx Context) (Scanner, error) {
				return Values("41", 42, 4.3), nil
			},
			expected:    Values("41", 42, 4.3),
			expectedErr: nil,
		},
		{
			name: "begin tx error",
			prepareDBProvider: func(m *MockDBProvider[PGX], tx *MockTx[PGX]) {
				m.EXPECT().
					BeginTx(opts).
					Return(nil, someErr)
			},
			opts: opts,
			job: func(ctx Context) (Scanner, error) {
				return Values(42), nil
			},
			expected:    nil,
			expectedErr: fmt.Errorf("beginning tx: %w", someErr),
		},
		{
			name: "commit tx error",
			prepareDBProvider: func(m *MockDBProvider[PGX], tx *MockTx[PGX]) {
				tx.EXPECT().
					GetDB().
					Return(nil)

				tx.EXPECT().
					Commit(mock.Anything).
					Return(someErr)

				tx.EXPECT().
					Rollback(mock.Anything).
					Return(nil)

				m.EXPECT().
					BeginTx(opts).
					Return(tx, nil)
			},
			opts: opts,
			job: func(ctx Context) (Scanner, error) {
				return Values(42), nil
			},
			expected:    nil,
			expectedErr: fmt.Errorf("committing tx: %w", someErr),
		},
		{
			name: "rollback tx error",
			prepareDBProvider: func(m *MockDBProvider[PGX], tx *MockTx[PGX]) {
				tx.EXPECT().
					GetDB().
					Return(nil)

				tx.EXPECT().
					Rollback(mock.Anything).
					Return(errors.New("rollback error"))

				m.EXPECT().
					BeginTx(opts).
					Return(tx, nil)
			},
			opts: opts,
			job: func(ctx Context) (Scanner, error) {
				return nil, someErr
			},
			expected: nil,
			expectedErr: fmt.Errorf(
				"rolling back the error='%s': %w",
				someErr,
				errors.New("rollback error"),
			),
		},
		{
			name: "panic error",
			prepareDBProvider: func(m *MockDBProvider[PGX], tx *MockTx[PGX]) {
				tx.EXPECT().
					GetDB().
					Return(nil)

				tx.EXPECT().
					Rollback(mock.Anything).
					Return(nil)

				m.EXPECT().
					BeginTx(opts).
					Return(tx, nil)
			},
			opts: opts,
			job: func(ctx Context) (Scanner, error) {
				panic(someErr)
			},
			expected:    nil,
			expectedErr: fmt.Errorf("panic: %v", someErr),
		},
		{
			name: "background context on empty value",
			prepareDBProvider: func(m *MockDBProvider[PGX], tx *MockTx[PGX]) {
				tx.EXPECT().
					GetDB().
					Return(nil)

				tx.EXPECT().
					Commit(mock.Anything).
					Return(nil)

				m.EXPECT().
					BeginTx(mock.Anything).
					RunAndReturn(func(opts TxOpts) (Tx[PGX], error) {
						require.Equal(t, context.Background(), opts.Ctx)
						return tx, nil
					})
			},
			opts: TxOpts{
				Ctx:       nil,
				Isolation: 0,
				ReadOnly:  false,
				Ext:       nil,
			},
			job: func(ctx Context) (Scanner, error) {
				return Values("41", 42, 4.3), nil
			},
			expected:    Values("41", 42, 4.3),
			expectedErr: nil,
		},
	}

	for _, c := range cases {
		c := c

		dbp := NewMockDBProvider[PGX](t)
		txMock := NewMockTx[PGX](t)
		c.prepareDBProvider(dbp, txMock)

		txm, _ := New[PGX](dbp)

		t.Run(c.name, func(t *testing.T) {
			scanner, err := txm.RunTx(c.opts, c.job)

			require.Equal(t, c.expectedErr, err)
			assert.Equal(t, c.expected, scanner)
		})
	}
}

func TestManager_RunNoTx(t *testing.T) {
	opts := NoTxOpts{
		Ctx: context.Background(),
		Ext: nil,
	}

	cases := []struct {
		name              string
		prepareDBProvider func(m *MockDBProvider[PGX])
		opts              NoTxOpts
		job               func(ctx Context) (Scanner, error)
		expected          Scanner
		expectedErr       error
	}{
		{
			name: "valid example",
			prepareDBProvider: func(m *MockDBProvider[PGX]) {
				m.EXPECT().
					GetDB(opts).
					Return(nil)
			},
			opts: opts,
			job: func(ctx Context) (Scanner, error) {
				return Values("41", 42, 4.3), nil
			},
			expected:    Values("41", 42, 4.3),
			expectedErr: nil,
		},
		{
			name: "exec error",
			prepareDBProvider: func(m *MockDBProvider[PGX]) {
				m.EXPECT().
					GetDB(opts).
					Return(nil)
			},
			opts: opts,
			job: func(ctx Context) (Scanner, error) {
				return nil, someErr
			},
			expected:    nil,
			expectedErr: someErr,
		},
		{
			name: "panic error",
			prepareDBProvider: func(m *MockDBProvider[PGX]) {
				m.EXPECT().
					GetDB(opts).
					Return(nil)
			},
			opts: opts,
			job: func(ctx Context) (Scanner, error) {
				panic(someErr)
			},
			expected:    nil,
			expectedErr: fmt.Errorf("panic: %v", someErr),
		},
		{
			name: "background context on empty value",
			prepareDBProvider: func(m *MockDBProvider[PGX]) {
				m.EXPECT().
					GetDB(opts).
					RunAndReturn(func(opts NoTxOpts) PGX {
						require.Equal(t, context.Background(), opts.Ctx)
						return nil
					})
			},
			opts: NoTxOpts{
				Ctx: nil,
				Ext: nil,
			},
			job: func(ctx Context) (Scanner, error) {
				return Values("41", 42, 4.3), nil
			},
			expected:    Values("41", 42, 4.3),
			expectedErr: nil,
		},
	}

	for _, c := range cases {
		c := c

		dbp := NewMockDBProvider[PGX](t)
		c.prepareDBProvider(dbp)

		txm, _ := New[PGX](dbp)

		t.Run(c.name, func(t *testing.T) {
			scanner, err := txm.RunNoTx(c.opts, c.job)

			require.Equal(t, c.expectedErr, err)
			assert.Equal(t, c.expected, scanner)
		})
	}
}

func TestManager_GetDB(t *testing.T) {
	txOpts := TxOpts{
		Ctx:       context.Background(),
		Isolation: LevelDefault,
		ReadOnly:  false,
		Ext:       nil,
	}

	noTxOpts := NoTxOpts{
		Ctx: context.Background(),
		Ext: nil,
	}

	cases := []struct {
		name              string
		prepareDBProvider func(m *MockDBProvider[PGX], tx *MockTx[PGX])
		runJob            func(txm TxManager, dbm DBManager[PGX]) (Scanner, error)
		expected          Scanner
		expectedErr       error
	}{
		{
			name: "valid example, GetDB inside transaction",
			prepareDBProvider: func(m *MockDBProvider[PGX], tx *MockTx[PGX]) {
				tx.EXPECT().
					GetDB().
					Return(NewMockPGX(t))

				tx.EXPECT().
					Commit(mock.Anything).
					Return(nil)

				m.EXPECT().
					BeginTx(txOpts).
					Return(tx, nil)
			},
			runJob: func(txm TxManager, dbm DBManager[PGX]) (Scanner, error) {
				return txm.RunTx(txOpts, func(ctx Context) (Scanner, error) {
					_, err := dbm.GetDB(ctx)
					if err != nil {
						return nil, err
					}

					return Values(42), nil
				})
			},
			expected:    Values(42),
			expectedErr: nil,
		},
		{
			name: "valid example, GetDB without transaction",
			prepareDBProvider: func(m *MockDBProvider[PGX], tx *MockTx[PGX]) {
				m.EXPECT().
					GetDB(noTxOpts).
					Return(NewMockPGX(t))
			},
			runJob: func(txm TxManager, dbm DBManager[PGX]) (Scanner, error) {
				return txm.RunNoTx(noTxOpts, func(ctx Context) (Scanner, error) {
					_, err := dbm.GetDB(ctx)
					if err != nil {
						return nil, err
					}

					return Values(42), nil
				})
			},
			expected:    Values(42),
			expectedErr: nil,
		},
		{
			name: "db not found error",
			prepareDBProvider: func(m *MockDBProvider[PGX], tx *MockTx[PGX]) {
				tx.EXPECT().
					GetDB().
					Return(nil)

				tx.EXPECT().
					Commit(mock.Anything).
					Return(nil)

				m.EXPECT().
					BeginTx(txOpts).
					Return(tx, nil)
			},
			runJob: func(txm TxManager, dbm DBManager[PGX]) (Scanner, error) {
				var privateCtx Context

				_, _ = txm.RunTx(txOpts, func(ctx Context) (Scanner, error) {
					privateCtx = ctx
					return nil, nil
				})

				_, err := dbm.GetDB(privateCtx)
				if err != nil {
					return nil, err
				}

				return Values(42), nil
			},
			expected:    nil,
			expectedErr: errors.New("db not found"),
		},
		{
			name: "db cast error",
			prepareDBProvider: func(m *MockDBProvider[PGX], tx *MockTx[PGX]) {
				m.EXPECT().
					GetDB(noTxOpts).
					Return(nil)
			},
			runJob: func(txm TxManager, dbm DBManager[PGX]) (Scanner, error) {
				return txm.RunNoTx(noTxOpts, func(ctx Context) (Scanner, error) {
					_, err := dbm.GetDB(ctx)
					if err != nil {
						return nil, err
					}

					return Values(42), nil
				})
			},
			expected:    nil,
			expectedErr: fmt.Errorf("panic: %v", errors.New("interface conversion: interface is nil, not txmng.PGX")),
		},
	}

	for _, c := range cases {
		c := c

		dbp := NewMockDBProvider[PGX](t)
		tx := NewMockTx[PGX](t)
		c.prepareDBProvider(dbp, tx)

		txm, dbm := New[PGX](dbp)

		t.Run(c.name, func(t *testing.T) {
			scanner, err := c.runJob(txm, dbm)

			require.Equal(t, c.expectedErr, err)
			assert.Equal(t, c.expected, scanner)
		})
	}
}

func TestManager_MustGetDB(t *testing.T) {
	noTxOpts := NoTxOpts{
		Ctx: context.Background(),
		Ext: nil,
	}

	cases := []struct {
		name              string
		prepareDBProvider func(m *MockDBProvider[PGX])
		runJob            func(txm TxManager, dbm DBManager[PGX]) (Scanner, error)
		expected          Scanner
		expectedErr       error
	}{
		{
			name: "valid example",
			prepareDBProvider: func(m *MockDBProvider[PGX]) {
				m.EXPECT().
					GetDB(noTxOpts).
					Return(NewMockPGX(t))
			},
			runJob: func(txm TxManager, dbm DBManager[PGX]) (Scanner, error) {
				return txm.RunNoTx(noTxOpts, func(ctx Context) (Scanner, error) {
					_ = dbm.MustGetDB(ctx)
					return Values(42), nil
				})
			},
			expected:    Values(42),
			expectedErr: nil,
		},
		{
			name: "panic error",
			prepareDBProvider: func(m *MockDBProvider[PGX]) {
				m.EXPECT().
					GetDB(noTxOpts).
					Return(nil).
					Times(2)
			},
			runJob: func(txm TxManager, dbm DBManager[PGX]) (Scanner, error) {
				var privateCtx Context
				_, _ = txm.RunNoTx(noTxOpts, func(ctx Context) (Scanner, error) {
					privateCtx = ctx
					return nil, nil
				})

				return txm.RunNoTx(noTxOpts, func(ctx Context) (Scanner, error) {
					_ = dbm.MustGetDB(privateCtx)
					return Values(42), nil
				})
			},
			expected:    nil,
			expectedErr: fmt.Errorf("panic: %v", errors.New("db not found")),
		},
	}

	for _, c := range cases {
		c := c

		dbp := NewMockDBProvider[PGX](t)
		c.prepareDBProvider(dbp)

		txm, dbm := New[PGX](dbp)

		t.Run(c.name, func(t *testing.T) {
			scanner, err := c.runJob(txm, dbm)

			require.Equal(t, c.expectedErr, err)
			assert.Equal(t, c.expected, scanner)
		})
	}
}

func TestManager_RunParallel(t *testing.T) {
	const countOfParallelQueries = 50_000

	txOpts := TxOpts{
		Ctx:       context.Background(),
		Isolation: LevelDefault,
		ReadOnly:  false,
		Ext:       nil,
	}

	noTxOpts := NoTxOpts{
		Ctx: context.Background(),
		Ext: nil,
	}

	cases := []struct {
		name              string
		prepareDBProvider func(m *MockDBProvider[PGX], tx *MockTx[PGX])
		runJob            func(txm TxManager, dbm DBManager[PGX]) (Scanner, error)
		expected          Scanner
		expectedErr       error
	}{
		{
			name: "valid example, run inside transaction",
			prepareDBProvider: func(m *MockDBProvider[PGX], tx *MockTx[PGX]) {
				tx.EXPECT().
					GetDB().
					Return(NewMockPGX(t)).
					Times(countOfParallelQueries)

				tx.EXPECT().
					Commit(mock.Anything).
					Return(nil).
					Times(countOfParallelQueries)

				m.EXPECT().
					BeginTx(txOpts).
					Return(tx, nil).
					Times(countOfParallelQueries)
			},
			runJob: func(txm TxManager, dbm DBManager[PGX]) (Scanner, error) {
				return txm.RunTx(txOpts, func(ctx Context) (Scanner, error) {
					db, err := dbm.GetDB(ctx)
					require.NoError(t, err)
					require.NotNil(t, db)

					return Values(41), nil
				})
			},
			expected:    Values(41),
			expectedErr: nil,
		},
		{
			name: "valid example, run without transaction",
			prepareDBProvider: func(m *MockDBProvider[PGX], tx *MockTx[PGX]) {
				m.EXPECT().
					GetDB(noTxOpts).
					Return(NewMockPGX(t)).
					Times(countOfParallelQueries)

			},
			runJob: func(txm TxManager, dbm DBManager[PGX]) (Scanner, error) {
				return txm.RunNoTx(noTxOpts, func(ctx Context) (Scanner, error) {
					db, err := dbm.GetDB(ctx)
					require.NoError(t, err)
					require.NotNil(t, db)

					return Values(42), nil
				})
			},
			expected:    Values(42),
			expectedErr: nil,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			dbp := NewMockDBProvider[PGX](t)
			tx := NewMockTx[PGX](t)
			c.prepareDBProvider(dbp, tx)

			txm, dbm := New[PGX](dbp)
			wg := &sync.WaitGroup{}

			for i := 0; i < countOfParallelQueries; i++ {
				wgGo(wg, func() {
					scanner, err := c.runJob(txm, dbm)
					require.Equal(t, c.expectedErr, err)
					assert.Equal(t, c.expected, scanner)
				})
			}

			wg.Wait()
		})
	}
}

func wgGo(wg *sync.WaitGroup, f func()) {
	wg.Add(1)

	go func() {
		defer wg.Done()
		f()
	}()
}
