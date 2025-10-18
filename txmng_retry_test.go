package txmng

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDefaultRetrier_isNetworkError(t *testing.T) {
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
			name:     "net.Error error",
			body:     fmt.Errorf("some error: %w", net.ErrClosed),
			expected: true,
		},
		{
			name:     "net.Error error 2",
			body:     fmt.Errorf("some error: %w", net.UnknownNetworkError("net error")),
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
	}

	s := defaultRetrier{}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			actual := s.isNetworkError(c.body)
			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestDefaultRetrier_isSerializationError(t *testing.T) {
	cases := []struct {
		name     string
		body     error
		expected bool
	}{
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
			actual := s.isSerializationError(c.body)
			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestDefaultRetrier_Do(t *testing.T) {
	cases := []struct {
		name               string
		newRetrier         func() Retrier
		createJob          func(retryCounter *int) func() error
		expectedErr        error
		expectedRetries    int
		expectedMinLatency time.Duration
		expectedMaxLatency time.Duration
	}{
		{
			name: "1 retry, not retriable error",
			newRetrier: func() Retrier {
				return newDefaultRetrier(
					[]time.Duration{
						1 * time.Millisecond,
						2 * time.Millisecond,
						3 * time.Millisecond,
					},
					0,
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
			name: "3 delays, 4 retries, network error",
			newRetrier: func() Retrier {
				return newDefaultRetrier(
					[]time.Duration{
						1 * time.Millisecond,
						2 * time.Millisecond,
						3 * time.Millisecond,
					},
					0,
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
			expectedMaxLatency: 7 * time.Millisecond,
		},
		{
			name: "3 delays, 4 retries, pgx serialization error",
			newRetrier: func() Retrier {
				return newDefaultRetrier(
					[]time.Duration{
						1 * time.Millisecond,
						2 * time.Millisecond,
						3 * time.Millisecond,
					},
					0,
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
			expectedMaxLatency: 7 * time.Millisecond,
		},
		{
			name: "3 delays, 4 retries, network error, left jitter border",
			newRetrier: func() Retrier {
				return newDefaultRetrier(
					[]time.Duration{
						1 * time.Millisecond,
						2 * time.Millisecond,
						3 * time.Millisecond,
					},
					-1,
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
			expectedMaxLatency: 7 * time.Millisecond,
		},
		{
			name: "3 delays, 4 retries, network error, right jitter border",
			newRetrier: func() Retrier {
				return newDefaultRetrier(
					[]time.Duration{
						1 * time.Millisecond,
						2 * time.Millisecond,
						3 * time.Millisecond,
					},
					2,
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
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			retrier := c.newRetrier()
			actualRetries := 0

			now := time.Now()
			actualErr := retrier.Do(c.createJob(&actualRetries))
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
		prepareDBProvider func(m *MockDBProvider[PGX])
		runJob            func(txm TxManager) (Scanner, error)
		expected          Scanner
		expectedErr       error
	}{
		{
			name: "retry error for tx",
			prepareRetrier: func(m *MockRetrier) {
				m.EXPECT().
					Do(mock.Anything).
					Return(someErr)
			},
			prepareDBProvider: func(m *MockDBProvider[PGX]) {},
			runJob: func(txm TxManager) (Scanner, error) {
				return txm.RunTx(txOpts, func(ctx Context) (Scanner, error) { return nil, nil })
			},
			expected:    nil,
			expectedErr: fmt.Errorf("running tx with retrier: %w", someErr),
		},
		{
			name: "job error for tx",
			prepareRetrier: func(m *MockRetrier) {
				m.EXPECT().
					Do(mock.Anything).
					RunAndReturn(func(fn func() error) error {
						return fn()
					})
			},
			prepareDBProvider: func(m *MockDBProvider[PGX]) {
				tx := NewMockTx[PGX](t)
				tx.EXPECT().
					Rollback(mock.Anything).
					Return(nil)
				tx.EXPECT().
					GetDB().
					Return(NewMockPGX(t))

				m.EXPECT().BeginTx(mock.Anything).Return(tx, nil)
			},
			runJob: func(txm TxManager) (Scanner, error) {
				return txm.RunTx(txOpts, func(ctx Context) (Scanner, error) { return nil, someErr })
			},
			expected:    nil,
			expectedErr: fmt.Errorf("running tx with retrier: %w", someErr),
		},
		{
			name: "valid example for tx",
			prepareRetrier: func(m *MockRetrier) {
				m.EXPECT().
					Do(mock.Anything).
					RunAndReturn(func(fn func() error) error {
						return fn()
					})
			},
			prepareDBProvider: func(m *MockDBProvider[PGX]) {
				tx := NewMockTx[PGX](t)
				tx.EXPECT().
					Commit(mock.Anything).
					Return(nil)
				tx.EXPECT().
					GetDB().
					Return(NewMockPGX(t))

				m.EXPECT().BeginTx(mock.Anything).Return(tx, nil)
			},
			runJob: func(txm TxManager) (Scanner, error) {
				return txm.RunTx(txOpts, func(ctx Context) (Scanner, error) { return Values(41), nil })
			},
			expected:    Values(41),
			expectedErr: nil,
		},
		{
			name: "retry error for noTx",
			prepareRetrier: func(m *MockRetrier) {
				m.EXPECT().
					Do(mock.Anything).
					Return(someErr)
			},
			prepareDBProvider: func(m *MockDBProvider[PGX]) {},
			runJob: func(txm TxManager) (Scanner, error) {
				return txm.RunNoTx(noTxOpts, func(ctx Context) (Scanner, error) { return nil, nil })
			},
			expected:    nil,
			expectedErr: fmt.Errorf("running tx with retrier: %w", someErr),
		},
		{
			name: "job error for noTx",
			prepareRetrier: func(m *MockRetrier) {
				m.EXPECT().
					Do(mock.Anything).
					RunAndReturn(func(fn func() error) error {
						return fn()
					})
			},
			prepareDBProvider: func(m *MockDBProvider[PGX]) {
				m.EXPECT().
					GetDB(mock.Anything).
					Return(NewMockPGX(t))
			},
			runJob: func(txm TxManager) (Scanner, error) {
				return txm.RunNoTx(noTxOpts, func(ctx Context) (Scanner, error) { return nil, someErr })
			},
			expected:    nil,
			expectedErr: fmt.Errorf("running tx with retrier: %w", someErr),
		},
		{
			name: "valid example for noTx",
			prepareRetrier: func(m *MockRetrier) {
				m.EXPECT().
					Do(mock.Anything).
					RunAndReturn(func(fn func() error) error {
						return fn()
					})
			},
			prepareDBProvider: func(m *MockDBProvider[PGX]) {
				m.EXPECT().
					GetDB(mock.Anything).
					Return(NewMockPGX(t))
			},
			runJob: func(txm TxManager) (Scanner, error) {
				return txm.RunNoTx(noTxOpts, func(ctx Context) (Scanner, error) { return Values(42), nil })
			},
			expected:    Values(42),
			expectedErr: nil,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			retrier := NewMockRetrier(t)
			c.prepareRetrier(retrier)

			dbp := NewMockDBProvider[PGX](t)
			c.prepareDBProvider(dbp)

			txm, _ := New[PGX](dbp, WithRetrier(retrier))

			scanner, err := c.runJob(txm)
			require.Equal(t, c.expectedErr, err)
			assert.Equal(t, c.expected, scanner)
		})
	}
}
