//go:build integration

package integration

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

func (rs Resources) Close() {
	rs.pgx.Close()

	_ = rs.sqlDB.Close()
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

	getPool := func(dsn string) (*pgxpool.Pool, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		pool, err := pgxpool.New(ctx, dsn)
		if err != nil {
			return nil, fmt.Errorf("unable to connect to database: %v", err)
		}

		return pool, nil
	}

	pool, err := getPool(dsn)
	if err != nil {
		log.Fatal(err)
	}

	db := stdlib.OpenDBFromPool(pool)

	rs = Resources{
		pgx:   pool,
		sqlDB: db,
	}

	code := m.Run()

	rs.Close()
	os.Exit(code)
}

func TestTxManager_PGXDB(t *testing.T) {
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
		runJob      func(txm txmng.TxManager, dbm txmng.DBManager[txmng.PGXDB]) (txmng.Result, error)
		expected    txmng.Result
		expectedErr error
	}{
		{
			name: "valid example with tx",
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 41)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.PGXDB]) (txmng.Result, error) {
				return txm.RunTx(txOpts, func(ctx txmng.TxContext) (txmng.Result, error) {
					db, rawCtx := dbm.GetDB(ctx)

					var counter int
					err := db.
						QueryRow(rawCtx, "SELECT value FROM counter WHERE id = $1", 1).
						Scan(&counter)
					if err != nil {
						return nil, err
					}

					return txmng.NewResult(counter), nil
				})
			},
			expected:    txmng.NewResult(41),
			expectedErr: nil,
		},
		{
			name: "error in tx",
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 41)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.PGXDB]) (txmng.Result, error) {
				return txm.RunTx(txOpts, func(ctx txmng.TxContext) (txmng.Result, error) {
					return nil, errors.New("some error")
				})
			},
			expected:    nil,
			expectedErr: fmt.Errorf("retry: %w", errors.New("some error")),
		},
		{
			name: "valid example without tx",
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 42)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.PGXDB]) (txmng.Result, error) {
				return txm.RunNoTx(noTxOpts, func(ctx txmng.NoTxContext) (txmng.Result, error) {
					db, rawCtx := dbm.GetDB(ctx)

					var counter int
					err := db.
						QueryRow(rawCtx, "SELECT value FROM counter WHERE id = $1", 1).
						Scan(&counter)
					if err != nil {
						return nil, err
					}

					return txmng.NewResult(counter), nil
				})
			},
			expected:    txmng.NewResult(42),
			expectedErr: nil,
		},
		{
			name: "error not in tx",
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 42)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.PGXDB]) (txmng.Result, error) {
				return txm.RunNoTx(noTxOpts, func(ctx txmng.NoTxContext) (txmng.Result, error) {
					return nil, errors.New("some error")
				})
			},
			expected:    nil,
			expectedErr: fmt.Errorf("retry: %w", errors.New("some error")),
		},
	}

	txm, dbm := txmng.New(txmng.NewPGXDB(rs.pgx), txmng.WithDefaultRetrier())

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

func TestTxManager_SQLDB(t *testing.T) {
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
		runJob      func(txm txmng.TxManager, dbm txmng.DBManager[txmng.SQLDB]) (txmng.Result, error)
		expected    txmng.Result
		expectedErr error
	}{
		{
			name: "valid example with tx",
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 41)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.SQLDB]) (txmng.Result, error) {
				return txm.RunTx(txOpts, func(ctx txmng.TxContext) (txmng.Result, error) {
					db, rawCtx := dbm.GetDB(ctx)

					var counter int
					err := db.
						QueryRowContext(rawCtx, "SELECT value FROM counter WHERE id = $1", 1).
						Scan(&counter)
					if err != nil {
						return nil, err
					}

					return txmng.NewResult(counter), nil
				})
			},
			expected:    txmng.NewResult(41),
			expectedErr: nil,
		},
		{
			name: "error in tx",
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 41)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.SQLDB]) (txmng.Result, error) {
				return txm.RunTx(txOpts, func(ctx txmng.TxContext) (txmng.Result, error) {
					return nil, errors.New("some error")
				})
			},
			expected:    nil,
			expectedErr: fmt.Errorf("retry: %w", errors.New("some error")),
		},
		{
			name: "valid example without tx",
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 42)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.SQLDB]) (txmng.Result, error) {
				return txm.RunNoTx(noTxOpts, func(ctx txmng.NoTxContext) (txmng.Result, error) {
					db, rawCtx := dbm.GetDB(ctx)

					var counter int
					err := db.
						QueryRowContext(rawCtx, "SELECT value FROM counter WHERE id = $1", 1).
						Scan(&counter)
					if err != nil {
						return nil, err
					}

					return txmng.NewResult(counter), nil
				})
			},
			expected:    txmng.NewResult(42),
			expectedErr: nil,
		},
		{
			name: "error not in tx",
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 42)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.SQLDB]) (txmng.Result, error) {
				return txm.RunNoTx(noTxOpts, func(ctx txmng.NoTxContext) (txmng.Result, error) {
					return nil, errors.New("some error")
				})
			},
			expected:    nil,
			expectedErr: fmt.Errorf("retry: %w", errors.New("some error")),
		},
	}

	txm, dbm := txmng.New(txmng.NewSQLDB(rs.sqlDB), txmng.WithDefaultRetrier())

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

func TestTxManager_PGXDB_RunParallel(t *testing.T) {
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
		runJob          func(txm txmng.TxManager, dbm txmng.DBManager[txmng.PGXDB])
		getActual       func(txm txmng.TxManager, dbm txmng.DBManager[txmng.PGXDB]) (txmng.Result, error)
		expected        txmng.Result
		expectedErr     error
	}{
		{
			name:            "valid example",
			parallelQueries: 1000,
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 0)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.PGXDB]) {
				_, err := txm.RunTx(txOpts, func(ctx txmng.TxContext) (txmng.Result, error) {
					db, rawCtx := dbm.GetDB(ctx)

					var counter int
					err := db.
						QueryRow(rawCtx, "SELECT value FROM counter WHERE id = $1", 1).
						Scan(&counter)
					if err != nil {
						return nil, err
					}

					_, err = db.Exec(rawCtx, "UPDATE counter SET value = $1 WHERE id=$2", counter+1, 1)
					if err != nil {
						return nil, err
					}

					return nil, nil
				})
				require.NoErrorf(t, err, "unable to run tx")
			},
			getActual: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.PGXDB]) (txmng.Result, error) {
				return txm.RunNoTx(noTxOpts, func(ctx txmng.NoTxContext) (txmng.Result, error) {
					db, rawCtx := dbm.GetDB(ctx)

					var counter int
					err := db.
						QueryRow(rawCtx, "SELECT value FROM counter WHERE id = $1", 1).
						Scan(&counter)
					if err != nil {
						return nil, err
					}

					return txmng.NewResult(counter), nil
				})
			},
			expected: txmng.NewResult(1000),
		},
	}

	txm, dbm := txmng.New(
		txmng.NewPGXDB(rs.pgx),
		txmng.WithRetrierParams(
			0.5,
			100*time.Millisecond,
			500*time.Millisecond,
			1*time.Second,
			2*time.Second,
			5*time.Second,
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

func TestTxManager_SQLDB_RunParallel(t *testing.T) {
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
		runJob          func(txm txmng.TxManager, dbm txmng.DBManager[txmng.SQLDB])
		getActual       func(txm txmng.TxManager, dbm txmng.DBManager[txmng.SQLDB]) (txmng.Result, error)
		expected        txmng.Result
		expectedErr     error
	}{
		{
			name:            "valid example",
			parallelQueries: 1000,
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 0)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.SQLDB]) {
				_, err := txm.RunTx(txOpts, func(ctx txmng.TxContext) (txmng.Result, error) {
					db, rawCtx := dbm.GetDB(ctx)

					var counter int
					err := db.
						QueryRowContext(rawCtx, "SELECT value FROM counter WHERE id = $1", 1).
						Scan(&counter)
					if err != nil {
						return nil, err
					}

					_, err = db.ExecContext(rawCtx, "UPDATE counter SET value = $1 WHERE id=$2", counter+1, 1)
					if err != nil {
						return nil, err
					}

					return nil, nil
				})
				require.NoErrorf(t, err, "unable to run tx")
			},
			getActual: func(txm txmng.TxManager, dbm txmng.DBManager[txmng.SQLDB]) (txmng.Result, error) {
				return txm.RunNoTx(noTxOpts, func(ctx txmng.NoTxContext) (txmng.Result, error) {
					db, rawCtx := dbm.GetDB(ctx)

					var counter int
					err := db.
						QueryRowContext(rawCtx, "SELECT value FROM counter WHERE id = $1", 1).
						Scan(&counter)
					if err != nil {
						return nil, err
					}

					return txmng.NewResult(counter), nil
				})
			},
			expected: txmng.NewResult(1000),
		},
	}

	txm, dbm := txmng.New(
		txmng.NewSQLDB(rs.sqlDB),
		txmng.WithRetrierParams(
			0.5,
			[]time.Duration{
				100 * time.Millisecond,
				500 * time.Millisecond,
				1 * time.Second,
				2 * time.Second,
				5 * time.Second,
			}...,
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

func TestScopedTxManager_PGXDB_RunParallel(t *testing.T) {
	type someSource struct{}

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
		runJob          func(txm txmng.STxManager[someSource], dbm txmng.SDBManager[someSource, txmng.PGXDB])
		getActual       func(txm txmng.STxManager[someSource], dbm txmng.SDBManager[someSource, txmng.PGXDB]) (txmng.Result, error)
		expected        txmng.Result
		expectedErr     error
	}{
		{
			name:            "valid example",
			parallelQueries: 1000,
			setup: func() {
				_, err := rs.pgx.Exec(ctx, "INSERT INTO counter(id, value) VALUES (1, 0)")
				require.NoError(t, err)
			},
			runJob: func(txm txmng.STxManager[someSource], dbm txmng.SDBManager[someSource, txmng.PGXDB]) {
				_, err := txm.RunTx(txOpts, func(ctx txmng.STxContext[someSource]) (txmng.Result, error) {
					db, rawCtx := dbm.GetDB(ctx)

					var counter int
					err := db.
						QueryRow(rawCtx, "SELECT value FROM counter WHERE id = $1", 1).
						Scan(&counter)
					if err != nil {
						return nil, err
					}

					_, err = db.Exec(rawCtx, "UPDATE counter SET value = $1 WHERE id=$2", counter+1, 1)
					if err != nil {
						return nil, err
					}

					return nil, nil
				})
				require.NoErrorf(t, err, "unable to run tx")
			},
			getActual: func(txm txmng.STxManager[someSource], dbm txmng.SDBManager[someSource, txmng.PGXDB]) (txmng.Result, error) {
				return txm.RunNoTx(noTxOpts, func(ctx txmng.SNoTxContext[someSource]) (txmng.Result, error) {
					db, rawCtx := dbm.GetDB(ctx)

					var counter int
					err := db.
						QueryRow(rawCtx, "SELECT value FROM counter WHERE id = $1", 1).
						Scan(&counter)
					if err != nil {
						return nil, err
					}

					return txmng.NewResult(counter), nil
				})
			},
			expected: txmng.NewResult(1000),
		},
	}

	txm, dbm := txmng.NewScoped[someSource](
		txmng.NewPGXDB(rs.pgx),
		txmng.WithRetrierParams(
			0.5,
			100*time.Millisecond,
			500*time.Millisecond,
			1*time.Second,
			2*time.Second,
			5*time.Second,
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
