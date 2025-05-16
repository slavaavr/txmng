package txmng

import (
	"context"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestTxmng_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const countOfParallelQueries = 100_000

	db := NewMockDB(ctrl)
	db.EXPECT().
		Commit().
		Times(countOfParallelQueries)

	p := NewMockDBProvider(ctrl)
	p.EXPECT().
		Tx(gomock.Any()).
		Return(db, nil).
		Times(countOfParallelQueries)

	mem := sync.Map{}
	wg := sync.WaitGroup{}

	txm, dbm := New(p)
	opts := Opts{
		Ctx:       context.Background(),
		Isolation: LevelDefault,
		ReadOnly:  false,
	}

	for i := 0; i < countOfParallelQueries; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = txm.Tx(opts, func(ctx Context) (Scanner, error) {
				txID := ctx.Value(txKey{}).(int64)
				assert.NotEmpty(t, txID, "txID not found")

				_, ok := mem.Load(txID)
				assert.Empty(t, ok, "txID already exists within mem -> duplicate of txID")
				mem.Store(txID, struct{}{})

				db, err := dbm.GetDB(ctx)
				assert.NoError(t, err, "no error should be provided")
				assert.NotEmpty(t, db, "db should exist")

				return nil, nil
			})
		}()
	}

	wg.Wait()
}
