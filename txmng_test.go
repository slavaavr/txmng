package txmng

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestTxmng_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const countOfParallelQueries = 100_000

	tx := NewMockTx[string](ctrl)
	tx.EXPECT().
		GetDB().
		Return("db").
		Times(countOfParallelQueries)
	tx.EXPECT().
		Commit(gomock.Any()).
		Times(countOfParallelQueries)

	p := NewMockDBProvider[string](ctrl)
	p.EXPECT().
		BeginTx(gomock.Any()).
		Return(tx, nil).
		Times(countOfParallelQueries)

	mem := sync.Map{}
	wg := sync.WaitGroup{}

	txm, dbm := New(p)
	opts := TxOpts{
		Ctx:       context.Background(),
		Isolation: LevelDefault,
		ReadOnly:  false,
	}

	for i := 0; i < countOfParallelQueries; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, _ = txm.RunTx(opts, func(ctx Context) (Scanner, error) {
				txID := ctx.Value(idKey{}).(int64)
				assert.NotEmpty(t, txID, "txID not found")

				_, ok := mem.Load(txID)
				assert.Empty(t, ok, "txID already exists within mem -> duplicate of txID")
				mem.Store(txID, struct{}{})

				db := dbm.GetDB(ctx)
				assert.NotEmpty(t, db, "db should exist")

				return nil, nil
			})
		}()
	}

	wg.Wait()
}

func Test_isSerializationError(t *testing.T) {
	cases := []struct {
		name     string
		body     error
		expected bool
	}{
		{
			name: "pgx serialization error",
			body: &pgconn.PgError{
				Code: "40001",
			},
			expected: true,
		},
		{
			name:     "pgx serialization error, nested",
			body:     fmt.Errorf("some error: %w", &pgconn.PgError{Code: "40001"}),
			expected: true,
		},
		{
			name: "pgx not serialization error",
			body: &pgconn.PgError{
				Code: "test",
			},
			expected: false,
		},
		{
			name:     "pgx not serialization error, nested",
			body:     fmt.Errorf("some error: %w", &pgconn.PgError{Code: "test"}),
			expected: false,
		},
		{
			name:     "not serialization error",
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
