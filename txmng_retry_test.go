package txmng

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDefaultRetrier_needRetry(t *testing.T) {
	cases := []struct {
		name     string
		body     error
		expected bool
	}{
		{
			name:     "io.EOF error",
			body:     fmt.Errorf("some error: %w", io.EOF),
			expected: true,
		},
		{
			name:     "io.ErrClosedPipe error",
			body:     fmt.Errorf("some error: %w", io.ErrClosedPipe),
			expected: true,
		},
		{
			name:     "net.Error temp error",
			body:     fmt.Errorf("some error: %w", &timeoutError{}),
			expected: true,
		},
		{
			name:     "pgx network error. 08000 — Connection Exception",
			body:     fmt.Errorf("some error: %w", &pgconn.PgError{Code: "08000"}),
			expected: true,
		},
		{
			name:     "pgx network error. 08003 — Connection Does Not Exist",
			body:     fmt.Errorf("some error: %w", &pgconn.PgError{Code: "08003"}),
			expected: true,
		},
		{
			name:     "pgx network error. 08006 — Connection Failure",
			body:     fmt.Errorf("some error: %w", &pgconn.PgError{Code: "08006"}),
			expected: true,
		},
		{
			name:     "not a network error",
			body:     errors.New("some error"),
			expected: false,
		},
		{
			name:     "pgx serialization error. 40001 — Serialization Failure",
			body:     fmt.Errorf("some error: %w", &pgconn.PgError{Code: "40001"}),
			expected: true,
		},
		{
			name:     "pgx serialization error. 40P01 — Deadlock Detected",
			body:     fmt.Errorf("some error: %w", &pgconn.PgError{Code: "40P01"}),
			expected: true,
		},
		{
			name:     "pgx serialization error. 55P03 — Lock Not Available",
			body:     fmt.Errorf("some error: %w", &pgconn.PgError{Code: "55P03"}),
			expected: true,
		},
		{
			name:     "pgx random error",
			body:     fmt.Errorf("some error: %w", &pgconn.PgError{Code: "test"}),
			expected: false,
		},
		{
			name:     "not a serialization error",
			body:     errors.New("some error"),
			expected: false,
		},
	}

	s := defaultRetrier{}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			actual := s.needRetry(c.body)
			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestDefaultRetrier_Do(t *testing.T) {
	defaultContext := context.Background()

	cases := []struct {
		name               string
		newCtx             func() context.Context
		newRetrier         func() Retrier
		createJob          func(retryCounter *int) func() error
		expectedErr        error
		expectedRetries    int
		expectedMinLatency time.Duration
		expectedMaxLatency time.Duration
	}{
		{
			name:   "1 retry, not retriable error",
			newCtx: func() context.Context { return defaultContext },
			newRetrier: func() Retrier {
				return newDefaultRetrier(
					0,
					[]time.Duration{
						1 * time.Millisecond,
						2 * time.Millisecond,
						3 * time.Millisecond,
					},
				)
			},
			createJob: func(retryCounter *int) func() error {
				return func() error {
					*retryCounter++
					return errors.New("some error")
				}
			},
			expectedErr:        errors.New("some error"),
			expectedRetries:    1,
			expectedMinLatency: 0,
			expectedMaxLatency: 100 * time.Microsecond,
		},
		{
			name:   "3 delays, 4 retries, network error",
			newCtx: func() context.Context { return defaultContext },
			newRetrier: func() Retrier {
				return newDefaultRetrier(
					0,
					[]time.Duration{
						1 * time.Millisecond,
						2 * time.Millisecond,
						3 * time.Millisecond,
					},
				)
			},
			createJob: func(retryCounter *int) func() error {
				return func() error {
					*retryCounter++
					return io.EOF
				}
			},
			expectedErr:        io.EOF,
			expectedRetries:    4,
			expectedMinLatency: 6 * time.Millisecond,
			expectedMaxLatency: 8 * time.Millisecond,
		},
		{
			name:   "3 delays, 4 retries, pgx serialization error",
			newCtx: func() context.Context { return defaultContext },
			newRetrier: func() Retrier {
				return newDefaultRetrier(
					0,
					[]time.Duration{
						1 * time.Millisecond,
						2 * time.Millisecond,
						3 * time.Millisecond,
					},
				)
			},
			createJob: func(retryCounter *int) func() error {
				return func() error {
					*retryCounter++
					return &pgconn.PgError{Code: "40001"}
				}
			},
			expectedErr:        &pgconn.PgError{Code: "40001"},
			expectedRetries:    4,
			expectedMinLatency: 6 * time.Millisecond,
			expectedMaxLatency: 8 * time.Millisecond,
		},
		{
			name:   "3 delays, 4 retries, network error, left jitter border",
			newCtx: func() context.Context { return defaultContext },
			newRetrier: func() Retrier {
				return newDefaultRetrier(
					-1,
					[]time.Duration{
						1 * time.Millisecond,
						2 * time.Millisecond,
						3 * time.Millisecond,
					},
				)
			},
			createJob: func(retryCounter *int) func() error {
				return func() error {
					*retryCounter++
					return io.EOF
				}
			},
			expectedErr:        io.EOF,
			expectedRetries:    4,
			expectedMinLatency: 6 * time.Millisecond,
			expectedMaxLatency: 8 * time.Millisecond,
		},
		{
			name:   "3 delays, 4 retries, network error, right jitter border",
			newCtx: func() context.Context { return defaultContext },
			newRetrier: func() Retrier {
				return newDefaultRetrier(
					2,
					[]time.Duration{
						1 * time.Millisecond,
						2 * time.Millisecond,
						3 * time.Millisecond,
					},
				)
			},
			createJob: func(retryCounter *int) func() error {
				return func() error {
					*retryCounter++
					return io.EOF
				}
			},
			expectedErr:        io.EOF,
			expectedRetries:    4,
			expectedMinLatency: 0,
			expectedMaxLatency: (12 + 1) * time.Millisecond,
		},
		{
			name: "context done error",
			newCtx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			},
			newRetrier: func() Retrier {
				return newDefaultRetrier(
					0,
					[]time.Duration{
						1 * time.Millisecond,
						2 * time.Millisecond,
						3 * time.Millisecond,
					},
				)
			},
			createJob: func(retryCounter *int) func() error {
				return func() error {
					*retryCounter++
					return io.EOF
				}
			},
			expectedErr:        context.Canceled,
			expectedRetries:    0,
			expectedMinLatency: 0,
			expectedMaxLatency: 3 * time.Millisecond,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			retrier := c.newRetrier()
			actualRetries := 0

			now := time.Now()
			actualErr := retrier.Do(c.newCtx(), c.createJob(&actualRetries))
			actualLatency := time.Since(now)

			assert.Equal(t, c.expectedErr, actualErr)
			assert.Equal(t, c.expectedRetries, actualRetries)
			assert.LessOrEqualf(t, c.expectedMinLatency, actualLatency,
				"expected min latency %v, got %v", c.expectedMinLatency, actualLatency)

			assert.LessOrEqualf(t, actualLatency, c.expectedMaxLatency,
				"expected max latency %v, got %v", c.expectedMaxLatency, actualLatency)
		})
	}
}

func TestTxManagerWithRetrier(t *testing.T) {
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
		prepareRetrier    func(m *MockRetrier)
		prepareDBProvider func(m *MockDBProvider[PGXDB])
		runJob            func(txm TxManager) (Result, error)
		expected          Result
		expectedErr       error
	}{
		{
			name: "nil context, expect no panic",
			prepareRetrier: func(m *MockRetrier) {
				m.EXPECT().
					Do(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func() error) error {
						require.NotNil(t, ctx)
						return nil
					})
			},
			prepareDBProvider: func(m *MockDBProvider[PGXDB]) {},
			runJob: func(txm TxManager) (Result, error) {
				txOptsTmp := txOpts
				txOptsTmp.Ctx = nil

				return txm.RunTx(txOptsTmp, func(ctx Context) (Result, error) { return nil, nil })
			},
			expected:    nil,
			expectedErr: nil,
		},
		{
			name: "retry error for tx",
			prepareRetrier: func(m *MockRetrier) {
				m.EXPECT().
					Do(mock.Anything, mock.Anything).
					Return(someErr)
			},
			prepareDBProvider: func(m *MockDBProvider[PGXDB]) {},
			runJob: func(txm TxManager) (Result, error) {
				return txm.RunTx(txOpts, func(ctx Context) (Result, error) { return nil, nil })
			},
			expected:    nil,
			expectedErr: fmt.Errorf("retry: %w", someErr),
		},
		{
			name: "job error for tx",
			prepareRetrier: func(m *MockRetrier) {
				m.EXPECT().
					Do(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func() error) error {
						return fn()
					})
			},
			prepareDBProvider: func(m *MockDBProvider[PGXDB]) {
				tx := NewMockTx[PGXDB](t)
				tx.EXPECT().
					Rollback(mock.Anything).
					Return(nil)
				tx.EXPECT().
					GetDB().
					Return(NewMockPGXDB(t))

				m.EXPECT().BeginTx(mock.Anything).Return(tx, nil)
			},
			runJob: func(txm TxManager) (Result, error) {
				return txm.RunTx(txOpts, func(ctx Context) (Result, error) { return nil, someErr })
			},
			expected:    nil,
			expectedErr: fmt.Errorf("retry: %w", someErr),
		},
		{
			name: "valid example for tx",
			prepareRetrier: func(m *MockRetrier) {
				m.EXPECT().
					Do(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func() error) error {
						return fn()
					})
			},
			prepareDBProvider: func(m *MockDBProvider[PGXDB]) {
				tx := NewMockTx[PGXDB](t)
				tx.EXPECT().
					Commit(mock.Anything).
					Return(nil)
				tx.EXPECT().
					GetDB().
					Return(NewMockPGXDB(t))

				m.EXPECT().BeginTx(mock.Anything).Return(tx, nil)
			},
			runJob: func(txm TxManager) (Result, error) {
				return txm.RunTx(txOpts, func(ctx Context) (Result, error) { return NewResult(41), nil })
			},
			expected:    NewResult(41),
			expectedErr: nil,
		},
		{
			name: "retry error for noTx",
			prepareRetrier: func(m *MockRetrier) {
				m.EXPECT().
					Do(mock.Anything, mock.Anything).
					Return(someErr)
			},
			prepareDBProvider: func(m *MockDBProvider[PGXDB]) {},
			runJob: func(txm TxManager) (Result, error) {
				return txm.RunNoTx(noTxOpts, func(ctx Context) (Result, error) { return nil, nil })
			},
			expected:    nil,
			expectedErr: fmt.Errorf("retry: %w", someErr),
		},
		{
			name: "job error for noTx",
			prepareRetrier: func(m *MockRetrier) {
				m.EXPECT().
					Do(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func() error) error {
						return fn()
					})
			},
			prepareDBProvider: func(m *MockDBProvider[PGXDB]) {
				m.EXPECT().
					GetDB(mock.Anything).
					Return(NewMockPGXDB(t))
			},
			runJob: func(txm TxManager) (Result, error) {
				return txm.RunNoTx(noTxOpts, func(ctx Context) (Result, error) { return nil, someErr })
			},
			expected:    nil,
			expectedErr: fmt.Errorf("retry: %w", someErr),
		},
		{
			name: "valid example for noTx",
			prepareRetrier: func(m *MockRetrier) {
				m.EXPECT().
					Do(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func() error) error {
						return fn()
					})
			},
			prepareDBProvider: func(m *MockDBProvider[PGXDB]) {
				m.EXPECT().
					GetDB(mock.Anything).
					Return(NewMockPGXDB(t))
			},
			runJob: func(txm TxManager) (Result, error) {
				return txm.RunNoTx(noTxOpts, func(ctx Context) (Result, error) { return NewResult(42), nil })
			},
			expected:    NewResult(42),
			expectedErr: nil,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			retrier := NewMockRetrier(t)
			c.prepareRetrier(retrier)

			dbp := NewMockDBProvider[PGXDB](t)
			c.prepareDBProvider(dbp)

			txm, _ := New[PGXDB](dbp, WithRetrier(retrier))

			Result, err := c.runJob(txm)
			require.Equal(t, c.expectedErr, err)
			assert.Equal(t, c.expected, Result)
		})
	}
}

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "i/o timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }
