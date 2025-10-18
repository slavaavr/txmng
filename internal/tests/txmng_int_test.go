//go:build integration

package tests

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/slavaavr/txmng"
)

type Resources struct {
	pgx   *pgxpool.Pool
	sqlDB *sql.DB
}

var (
	//go:embed env/pg_setup.sql
	pgSetupSQL string
	//go:embed env/pg_teardown.sql
	pgTeardownSQL string

	rs Resources
)

func beforeTest(t *testing.T) {
	t.Helper()

	_, err := rs.pgx.Exec(t.Context(), pgSetupSQL)
	require.NoError(t, err)
}

func afterTest(t *testing.T) {
	t.Helper()

	_, err := rs.pgx.Exec(t.Context(), pgTeardownSQL)
	require.NoError(t, err)
}

func TestMain(m *testing.M) {
	dsn := os.Getenv("PG_DSN")
	if dsn == "" {
		log.Fatal("PG_DSN environment variable is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}

	defer pool.Close()

	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	rs = Resources{
		pgx:   pool,
		sqlDB: db,
	}

	code := m.Run()
	os.Exit(code)
}

func TestTxManager_PGX(t *testing.T) {
	ctx := t.Context()
	txOpts := txmng.TxOpts{
		Ctx:       ctx,
		Isolation: txmng.LevelSerializable,
		ReadOnly:  false,
		Ext:       nil,
	}
	noTxOpts := txmng.NoTxOpts{
		Ctx: ctx,
		Ext: nil,
	}

	cases := []struct {
		name        string
		setup       func()
		runJob      func(txm txmng.TxManager, dbm txmng.DBManager[txmng.PGX]) (txmng.Scanner, error)
		expected    txmng.Scanner
		expectedErr error
	}{
		{
			name: "valid example with tx",
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 41)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.PGX]) (txmng.Scanner, error) {
				return txm.RunTx(txOpts, func(ctx txmng.Context) (txmng.Scanner, error) {
					db, err := dbm.GetDB(ctx)
					if err != nil {
						return nil, err
					}

					var counter int
					err = db.
						QueryRow(ctx, "SELECT value FROM counter WHERE id = $1", 1).
						Scan(&counter)
					if err != nil {
						return nil, err
					}

					return txmng.Values(counter), nil
				})
			},
			expected:    txmng.Values(41),
			expectedErr: nil,
		},
		{
			name: "error in tx",
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 41)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.PGX]) (txmng.Scanner, error) {
				return txm.RunTx(txOpts, func(ctx txmng.Context) (txmng.Scanner, error) {
					return nil, errors.New("some error")
				})
			},
			expected:    nil,
			expectedErr: fmt.Errorf("running tx with retrier: %w", errors.New("some error")),
		},
		{
			name: "valid example without tx",
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 42)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.PGX]) (txmng.Scanner, error) {
				return txm.RunNoTx(noTxOpts, func(ctx txmng.Context) (txmng.Scanner, error) {
					db, err := dbm.GetDB(ctx)
					if err != nil {
						return nil, err
					}

					var counter int
					err = db.
						QueryRow(ctx, "SELECT value FROM counter WHERE id = $1", 1).
						Scan(&counter)
					if err != nil {
						return nil, err
					}

					return txmng.Values(counter), nil
				})
			},
			expected:    txmng.Values(42),
			expectedErr: nil,
		},
		{
			name: "error not in tx",
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 42)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.PGX]) (txmng.Scanner, error) {
				return txm.RunNoTx(noTxOpts, func(ctx txmng.Context) (txmng.Scanner, error) {
					return nil, errors.New("some error")
				})
			},
			expected:    nil,
			expectedErr: fmt.Errorf("running tx with retrier: %w", errors.New("some error")),
		},
	}

	txm, dbm := txmng.New(txmng.NewPGXProvider(rs.pgx), txmng.WithDefaultRetrier())

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			afterTest(t)
			beforeTest(t)
			c.setup()
			actual, err := c.runJob(txm, dbm)
			require.Equal(t, c.expectedErr, err)
			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestTxManager_StdSQL(t *testing.T) {
	ctx := t.Context()
	txOpts := txmng.TxOpts{
		Ctx:       ctx,
		Isolation: txmng.LevelSerializable,
		ReadOnly:  false,
		Ext:       nil,
	}
	noTxOpts := txmng.NoTxOpts{
		Ctx: ctx,
		Ext: nil,
	}

	cases := []struct {
		name        string
		setup       func()
		runJob      func(txm txmng.TxManager, dbm txmng.DBManager[txmng.StdSQL]) (txmng.Scanner, error)
		expected    txmng.Scanner
		expectedErr error
	}{
		{
			name: "valid example with tx",
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 41)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.StdSQL]) (txmng.Scanner, error) {
				return txm.RunTx(txOpts, func(ctx txmng.Context) (txmng.Scanner, error) {
					db, err := dbm.GetDB(ctx)
					if err != nil {
						return nil, err
					}

					var counter int
					err = db.
						QueryRowContext(ctx, "SELECT value FROM counter WHERE id = $1", 1).
						Scan(&counter)
					if err != nil {
						return nil, err
					}

					return txmng.Values(counter), nil
				})
			},
			expected:    txmng.Values(41),
			expectedErr: nil,
		},
		{
			name: "error in tx",
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 41)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.StdSQL]) (txmng.Scanner, error) {
				return txm.RunTx(txOpts, func(ctx txmng.Context) (txmng.Scanner, error) {
					return nil, errors.New("some error")
				})
			},
			expected:    nil,
			expectedErr: fmt.Errorf("running tx with retrier: %w", errors.New("some error")),
		},
		{
			name: "valid example without tx",
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 42)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.StdSQL]) (txmng.Scanner, error) {
				return txm.RunNoTx(noTxOpts, func(ctx txmng.Context) (txmng.Scanner, error) {
					db, err := dbm.GetDB(ctx)
					if err != nil {
						return nil, err
					}

					var counter int
					err = db.
						QueryRowContext(ctx, "SELECT value FROM counter WHERE id = $1", 1).
						Scan(&counter)
					if err != nil {
						return nil, err
					}

					return txmng.Values(counter), nil
				})
			},
			expected:    txmng.Values(42),
			expectedErr: nil,
		},
		{
			name: "error not in tx",
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 42)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.StdSQL]) (txmng.Scanner, error) {
				return txm.RunNoTx(noTxOpts, func(ctx txmng.Context) (txmng.Scanner, error) {
					return nil, errors.New("some error")
				})
			},
			expected:    nil,
			expectedErr: fmt.Errorf("running tx with retrier: %w", errors.New("some error")),
		},
	}

	txm, dbm := txmng.New(txmng.NewStdSQLProvider(rs.sqlDB), txmng.WithDefaultRetrier())

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			afterTest(t)
			beforeTest(t)
			c.setup()
			actual, err := c.runJob(txm, dbm)
			require.Equal(t, c.expectedErr, err)
			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestTxManager_PGX_RunParallel(t *testing.T) {
	ctx := t.Context()
	txOpts := txmng.TxOpts{
		Ctx:       ctx,
		Isolation: txmng.LevelSerializable,
		ReadOnly:  false,
		Ext:       nil,
	}
	noTxOpts := txmng.NoTxOpts{
		Ctx: ctx,
		Ext: nil,
	}

	cases := []struct {
		name            string
		parallelQueries int
		setup           func()
		runJob          func(txm txmng.TxManager, dbm txmng.DBManager[txmng.PGX])
		getActual       func(txm txmng.TxManager, dbm txmng.DBManager[txmng.PGX]) (txmng.Scanner, error)
		expected        txmng.Scanner
		expectedErr     error
	}{
		{
			name:            "valid example",
			parallelQueries: 1000,
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 0)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.PGX]) {
				_, err := txm.RunTx(txOpts, func(ctx txmng.Context) (txmng.Scanner, error) {
					db, err := dbm.GetDB(ctx)
					require.NoErrorf(t, err, "unable to get DB")

					var counter int
					err = db.
						QueryRow(ctx, "SELECT value FROM counter WHERE id = $1", 1).
						Scan(&counter)
					if err != nil {
						return nil, err
					}

					_, err = db.Exec(ctx, "UPDATE counter SET value = $1 WHERE id=$2", counter+1, 1)
					if err != nil {
						return nil, err
					}

					return nil, nil
				})
				require.NoErrorf(t, err, "unable to run tx")
			},
			getActual: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.PGX]) (txmng.Scanner, error) {
				return txm.RunNoTx(noTxOpts, func(ctx txmng.Context) (txmng.Scanner, error) {
					db := dbm.MustGetDB(ctx)
					var counter int
					err := db.
						QueryRow(ctx, "SELECT value FROM counter WHERE id = $1", 1).
						Scan(&counter)
					if err != nil {
						return nil, err
					}

					return txmng.Values(counter), nil
				})
			},
			expected: txmng.Values(1000),
		},
	}

	txm, dbm := txmng.New(
		txmng.NewPGXProvider(rs.pgx),
		txmng.WithRetrierParams([]time.Duration{
			100 * time.Millisecond,
			500 * time.Millisecond,
			1 * time.Second,
			2 * time.Second,
			5 * time.Second,
		},
			0.5,
		),
	)

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			wg := &sync.WaitGroup{}
			afterTest(t)
			beforeTest(t)
			c.setup()

			for i := 0; i < c.parallelQueries; i++ {
				wgGo(wg, func() { c.runJob(txm, dbm) })
			}

			wg.Wait()

			actual, err := c.getActual(txm, dbm)
			require.Equal(t, c.expectedErr, err)
			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestTxManager_StdSQL_RunParallel(t *testing.T) {
	ctx := t.Context()
	txOpts := txmng.TxOpts{
		Ctx:       ctx,
		Isolation: txmng.LevelSerializable,
		ReadOnly:  false,
		Ext:       nil,
	}
	noTxOpts := txmng.NoTxOpts{
		Ctx: ctx,
		Ext: nil,
	}

	cases := []struct {
		name            string
		parallelQueries int
		setup           func()
		runJob          func(txm txmng.TxManager, dbm txmng.DBManager[txmng.StdSQL])
		getActual       func(txm txmng.TxManager, dbm txmng.DBManager[txmng.StdSQL]) (txmng.Scanner, error)
		expected        txmng.Scanner
		expectedErr     error
	}{
		{
			name:            "valid example",
			parallelQueries: 1000,
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 0)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.StdSQL]) {
				_, err := txm.RunTx(txOpts, func(ctx txmng.Context) (txmng.Scanner, error) {
					db, err := dbm.GetDB(ctx)
					require.NoErrorf(t, err, "unable to get DB")

					var counter int
					err = db.
						QueryRowContext(ctx, "SELECT value FROM counter WHERE id = $1", 1).
						Scan(&counter)
					if err != nil {
						return nil, err
					}

					_, err = db.ExecContext(ctx, "UPDATE counter SET value = $1 WHERE id=$2", counter+1, 1)
					if err != nil {
						return nil, err
					}

					return nil, nil
				})
				require.NoErrorf(t, err, "unable to run tx")
			},
			getActual: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.StdSQL]) (txmng.Scanner, error) {
				return txm.RunNoTx(noTxOpts, func(ctx txmng.Context) (txmng.Scanner, error) {
					db := dbm.MustGetDB(ctx)
					var counter int
					err := db.
						QueryRowContext(ctx, "SELECT value FROM counter WHERE id = $1", 1).
						Scan(&counter)
					if err != nil {
						return nil, err
					}

					return txmng.Values(counter), nil
				})
			},
			expected: txmng.Values(1000),
		},
	}

	txm, dbm := txmng.New(
		txmng.NewStdSQLProvider(rs.sqlDB),
		txmng.WithRetrierParams([]time.Duration{
			100 * time.Millisecond,
			500 * time.Millisecond,
			1 * time.Second,
			2 * time.Second,
			5 * time.Second,
		},
			0.5,
		),
	)

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			wg := &sync.WaitGroup{}
			afterTest(t)
			beforeTest(t)
			c.setup()

			for i := 0; i < c.parallelQueries; i++ {
				wgGo(wg, func() { c.runJob(txm, dbm) })
			}

			wg.Wait()

			actual, err := c.getActual(txm, dbm)
			require.Equal(t, c.expectedErr, err)
			assert.Equal(t, c.expected, actual)
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
