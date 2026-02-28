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

type someSource struct{}

func TestScopedManager_RunTx(t *testing.T) {
	opts := TxOpts{
		Ctx:       context.Background(),
		Isolation: LevelDefault,
		ReadOnly:  false,
		Ext:       nil,
	}

	cases := []struct {
		name              string
		prepareDBProvider func(m *MockDBProvider[PGXDB], tx *MockTx[PGXDB])
		opts              TxOpts
		job               func(ctx STxContext[someSource]) (Result, error)
		expected          Result
		expectedErr       error
	}{
		{
			name: "valid example",
			prepareDBProvider: func(m *MockDBProvider[PGXDB], tx *MockTx[PGXDB]) {
				tx.EXPECT().
					GetDB().
					Return(nil)

				tx.EXPECT().
					Commit(mock.Anything).
					Return(nil)

				m.EXPECT().
					BeginTx(opts).
					Return(tx, nil)

				m.EXPECT().
					GetDB(NoTxOpts{}).
					Return(nil)
			},
			opts: opts,
			job: func(ctx STxContext[someSource]) (Result, error) {
				return NewResult("41", 42, 4.3), nil
			},
			expected:    NewResult("41", 42, 4.3),
			expectedErr: nil,
		},
		{
			name: "begin tx error",
			prepareDBProvider: func(m *MockDBProvider[PGXDB], tx *MockTx[PGXDB]) {
				m.EXPECT().
					BeginTx(opts).
					Return(nil, someErr)

				m.EXPECT().
					GetDB(NoTxOpts{}).
					Return(nil)
			},
			opts: opts,
			job: func(ctx STxContext[someSource]) (Result, error) {
				return NewResult(42), nil
			},
			expected:    nil,
			expectedErr: fmt.Errorf("begin tx: %w", someErr),
		},
		{
			name: "commit tx error",
			prepareDBProvider: func(m *MockDBProvider[PGXDB], tx *MockTx[PGXDB]) {
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

				m.EXPECT().
					GetDB(NoTxOpts{}).
					Return(nil)
			},
			opts: opts,
			job: func(ctx STxContext[someSource]) (Result, error) {
				return NewResult(42), nil
			},
			expected:    nil,
			expectedErr: fmt.Errorf("commit tx: %w", someErr),
		},
		{
			name: "rollback tx error",
			prepareDBProvider: func(m *MockDBProvider[PGXDB], tx *MockTx[PGXDB]) {
				tx.EXPECT().
					GetDB().
					Return(nil)

				tx.EXPECT().
					Rollback(mock.Anything).
					Return(errors.New("rollback error"))

				m.EXPECT().
					BeginTx(opts).
					Return(tx, nil)

				m.EXPECT().
					GetDB(NoTxOpts{}).
					Return(nil)
			},
			opts: opts,
			job: func(ctx STxContext[someSource]) (Result, error) {
				return nil, someErr
			},
			expected: nil,
			expectedErr: fmt.Errorf(
				"rollback tx (%s): %w",
				errors.New("rollback error"),
				someErr,
			),
		},
		{
			name: "panic error",
			prepareDBProvider: func(m *MockDBProvider[PGXDB], tx *MockTx[PGXDB]) {
				tx.EXPECT().
					GetDB().
					Return(nil)

				tx.EXPECT().
					Rollback(mock.Anything).
					Return(nil)

				m.EXPECT().
					BeginTx(opts).
					Return(tx, nil)

				m.EXPECT().
					GetDB(NoTxOpts{}).
					Return(nil)
			},
			opts: opts,
			job: func(ctx STxContext[someSource]) (Result, error) {
				panic(someErr)
			},
			expected:    nil,
			expectedErr: fmt.Errorf("panic: %v", someErr),
		},
		{
			name: "background context on empty value",
			prepareDBProvider: func(m *MockDBProvider[PGXDB], tx *MockTx[PGXDB]) {
				tx.EXPECT().
					GetDB().
					Return(nil)

				tx.EXPECT().
					Commit(mock.Anything).
					Return(nil)

				m.EXPECT().
					BeginTx(mock.Anything).
					RunAndReturn(func(opts TxOpts) (Tx[PGXDB], error) {
						require.Equal(t, context.Background(), opts.Ctx)
						return tx, nil
					})

				m.EXPECT().
					GetDB(NoTxOpts{}).
					Return(nil)
			},
			opts: TxOpts{
				Ctx:       nil,
				Isolation: 0,
				ReadOnly:  false,
				Ext:       nil,
			},
			job: func(ctx STxContext[someSource]) (Result, error) {
				return NewResult("41", 42, 4.3), nil
			},
			expected:    NewResult("41", 42, 4.3),
			expectedErr: nil,
		},
	}

	for _, c := range cases {
		c := c

		dbp := NewMockDBProvider[PGXDB](t)
		txMock := NewMockTx[PGXDB](t)
		c.prepareDBProvider(dbp, txMock)

		txm, _ := NewScoped[someSource, PGXDB](dbp)

		t.Run(c.name, func(t *testing.T) {
			actual, err := txm.RunTx(c.opts, c.job)

			require.Equal(t, c.expectedErr, err)
			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestScopedManager_RunNoTx(t *testing.T) {
	opts := NoTxOpts{
		Ctx: context.Background(),
		Ext: nil,
	}

	cases := []struct {
		name              string
		prepareDBProvider func(m *MockDBProvider[PGXDB])
		opts              NoTxOpts
		job               func(ctx SNoTxContext[someSource]) (Result, error)
		expected          Result
		expectedErr       error
	}{
		{
			name: "valid example",
			prepareDBProvider: func(m *MockDBProvider[PGXDB]) {
				m.EXPECT().
					GetDB(opts).
					Return(nil)

				m.EXPECT().
					GetDB(NoTxOpts{}).
					Return(nil)
			},
			opts: opts,
			job: func(ctx SNoTxContext[someSource]) (Result, error) {
				return NewResult("41", 42, 4.3), nil
			},
			expected:    NewResult("41", 42, 4.3),
			expectedErr: nil,
		},
		{
			name: "exec error",
			prepareDBProvider: func(m *MockDBProvider[PGXDB]) {
				m.EXPECT().
					GetDB(opts).
					Return(nil)

				m.EXPECT().
					GetDB(NoTxOpts{}).
					Return(nil)
			},
			opts: opts,
			job: func(ctx SNoTxContext[someSource]) (Result, error) {
				return nil, someErr
			},
			expected:    nil,
			expectedErr: someErr,
		},
		{
			name: "panic error",
			prepareDBProvider: func(m *MockDBProvider[PGXDB]) {
				m.EXPECT().
					GetDB(opts).
					Return(nil)

				m.EXPECT().
					GetDB(NoTxOpts{}).
					Return(nil)
			},
			opts: opts,
			job: func(ctx SNoTxContext[someSource]) (Result, error) {
				panic(someErr)
			},
			expected:    nil,
			expectedErr: fmt.Errorf("panic: %v", someErr),
		},
		{
			name: "background context on empty value",
			prepareDBProvider: func(m *MockDBProvider[PGXDB]) {
				m.EXPECT().
					GetDB(opts).
					RunAndReturn(func(opts NoTxOpts) PGXDB {
						require.Equal(t, context.Background(), opts.Ctx)
						return nil
					})

				m.EXPECT().
					GetDB(NoTxOpts{}).
					Return(nil)
			},
			opts: NoTxOpts{
				Ctx: nil,
				Ext: nil,
			},
			job: func(ctx SNoTxContext[someSource]) (Result, error) {
				return NewResult("41", 42, 4.3), nil
			},
			expected:    NewResult("41", 42, 4.3),
			expectedErr: nil,
		},
	}

	for _, c := range cases {
		c := c

		dbp := NewMockDBProvider[PGXDB](t)
		c.prepareDBProvider(dbp)

		txm, _ := NewScoped[someSource, PGXDB](dbp)

		t.Run(c.name, func(t *testing.T) {
			actual, err := txm.RunNoTx(c.opts, c.job)

			require.Equal(t, c.expectedErr, err)
			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestScopedManager_GetDB(t *testing.T) {
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
		prepareDBProvider func(m *MockDBProvider[PGXDB], tx *MockTx[PGXDB])
		runJob            func(txm STxManager[someSource], dbm SDBManager[someSource, PGXDB]) (Result, error)
		expected          Result
		expectedErr       error
	}{
		{
			name: "valid example, GetDB inside transaction",
			prepareDBProvider: func(m *MockDBProvider[PGXDB], tx *MockTx[PGXDB]) {
				tx.EXPECT().
					GetDB().
					Return(NewMockPGXDB(t))

				tx.EXPECT().
					Commit(mock.Anything).
					Return(nil)

				m.EXPECT().
					BeginTx(txOpts).
					Return(tx, nil)

				m.EXPECT().
					GetDB(NoTxOpts{}).
					Return(nil)
			},
			runJob: func(txm STxManager[someSource], dbm SDBManager[someSource, PGXDB]) (Result, error) {
				return txm.RunTx(txOpts, func(ctx STxContext[someSource]) (Result, error) {
					_, _ = dbm.GetDB(ctx)
					return NewResult(42), nil
				})
			},
			expected:    NewResult(42),
			expectedErr: nil,
		},
		{
			name: "valid example, GetDB without transaction",
			prepareDBProvider: func(m *MockDBProvider[PGXDB], tx *MockTx[PGXDB]) {
				m.EXPECT().
					GetDB(noTxOpts).
					Return(NewMockPGXDB(t))

				m.EXPECT().
					GetDB(NoTxOpts{}).
					Return(nil)
			},
			runJob: func(txm STxManager[someSource], dbm SDBManager[someSource, PGXDB]) (Result, error) {
				return txm.RunNoTx(noTxOpts, func(ctx SNoTxContext[someSource]) (Result, error) {
					_, _ = dbm.GetDB(ctx)
					return NewResult(42), nil
				})
			},
			expected:    NewResult(42),
			expectedErr: nil,
		},
		{
			name: "use of invalid context",
			prepareDBProvider: func(m *MockDBProvider[PGXDB], tx *MockTx[PGXDB]) {
				tx.EXPECT().
					GetDB().
					Return(nil)

				tx.EXPECT().
					Commit(mock.Anything).
					Return(nil)

				tx.EXPECT().
					Rollback(mock.Anything).
					Return(nil)

				m.EXPECT().
					BeginTx(txOpts).
					Return(tx, nil)

				m.EXPECT().
					GetDB(NoTxOpts{}).
					Return(nil)
			},
			runJob: func(txm STxManager[someSource], dbm SDBManager[someSource, PGXDB]) (Result, error) {
				var outdatedCtx SContext[someSource]

				_, err := txm.RunTx(txOpts, func(ctx STxContext[someSource]) (Result, error) {
					outdatedCtx = ctx
					return nil, nil
				})
				require.NoError(t, err)

				_, err = txm.RunTx(txOpts, func(ctx STxContext[someSource]) (Result, error) {
					_, _ = dbm.GetDB(outdatedCtx)
					return nil, nil
				})

				return nil, err
			},
			expected:    nil,
			expectedErr: fmt.Errorf("panic: %v", errInvalidContext),
		},
		{
			name: "db cast error",
			prepareDBProvider: func(m *MockDBProvider[PGXDB], tx *MockTx[PGXDB]) {
				m.EXPECT().
					GetDB(noTxOpts).
					Return(nil)

				m.EXPECT().
					GetDB(NoTxOpts{}).
					Return(nil)
			},
			runJob: func(txm STxManager[someSource], dbm SDBManager[someSource, PGXDB]) (Result, error) {
				return txm.RunNoTx(noTxOpts, func(ctx SNoTxContext[someSource]) (Result, error) {
					_, _ = dbm.GetDB(ctx)
					return NewResult(42), nil
				})
			},
			expected:    nil,
			expectedErr: fmt.Errorf("panic: %v", errors.New("interface conversion: interface is nil, not txmng.PGXDB")),
		},
	}

	for _, c := range cases {
		c := c

		dbp := NewMockDBProvider[PGXDB](t)
		tx := NewMockTx[PGXDB](t)
		c.prepareDBProvider(dbp, tx)

		txm, dbm := NewScoped[someSource, PGXDB](dbp)

		t.Run(c.name, func(t *testing.T) {
			actual, err := c.runJob(txm, dbm)

			require.Equal(t, c.expectedErr, err)
			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestScopedManager_RunParallel(t *testing.T) {
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
		prepareDBProvider func(m *MockDBProvider[PGXDB], tx *MockTx[PGXDB])
		runJob            func(txm STxManager[someSource], dbm SDBManager[someSource, PGXDB]) (Result, error)
		expected          Result
		expectedErr       error
	}{
		{
			name: "valid example, run inside transaction",
			prepareDBProvider: func(m *MockDBProvider[PGXDB], tx *MockTx[PGXDB]) {
				tx.EXPECT().
					GetDB().
					Return(NewMockPGXDB(t)).
					Times(countOfParallelQueries)

				tx.EXPECT().
					Commit(mock.Anything).
					Return(nil).
					Times(countOfParallelQueries)

				m.EXPECT().
					BeginTx(txOpts).
					Return(tx, nil).
					Times(countOfParallelQueries)

				m.EXPECT().
					GetDB(NoTxOpts{}).
					Return(nil)
			},
			runJob: func(txm STxManager[someSource], dbm SDBManager[someSource, PGXDB]) (Result, error) {
				return txm.RunTx(txOpts, func(ctx STxContext[someSource]) (Result, error) {
					db, rawCtx := dbm.GetDB(ctx)
					require.NotNil(t, db)
					require.NotNil(t, rawCtx)
					return NewResult(41), nil
				})
			},
			expected:    NewResult(41),
			expectedErr: nil,
		},
		{
			name: "valid example, run without transaction",
			prepareDBProvider: func(m *MockDBProvider[PGXDB], tx *MockTx[PGXDB]) {
				m.EXPECT().
					GetDB(noTxOpts).
					Return(NewMockPGXDB(t)).
					Times(countOfParallelQueries)

				m.EXPECT().
					GetDB(NoTxOpts{}).
					Return(nil)
			},
			runJob: func(txm STxManager[someSource], dbm SDBManager[someSource, PGXDB]) (Result, error) {
				return txm.RunNoTx(noTxOpts, func(ctx SNoTxContext[someSource]) (Result, error) {
					db, rawCtx := dbm.GetDB(ctx)
					require.NotNil(t, db)
					require.NotNil(t, rawCtx)

					return NewResult(42), nil
				})
			},
			expected:    NewResult(42),
			expectedErr: nil,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			dbp := NewMockDBProvider[PGXDB](t)
			tx := NewMockTx[PGXDB](t)
			c.prepareDBProvider(dbp, tx)

			txm, dbm := NewScoped[someSource, PGXDB](dbp)
			wg := &sync.WaitGroup{}

			for i := 0; i < countOfParallelQueries; i++ {
				wgGo(wg, func() {
					actual, err := c.runJob(txm, dbm)
					require.Equal(t, c.expectedErr, err)
					assert.Equal(t, c.expected, actual)
				})
			}

			wg.Wait()
		})
	}
}
