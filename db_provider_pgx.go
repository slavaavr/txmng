package txmng

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PGXDB interface {
	pgx.Tx
}

type pgxDB struct {
	db    *pgxpool.Pool
	rawDB PGXDB

	isoLevelsMap       map[IsolationLevel]pgx.TxIsoLevel
	deferrableModesMap map[bool]pgx.TxDeferrableMode
	accessModesMap     map[bool]pgx.TxAccessMode
}

func NewPGXDB(db *pgxpool.Pool) DBProvider[PGXDB] {
	return &pgxDB{
		db: db,
		rawDB: &pgxRawDB{
			Pool: db,
		},
		isoLevelsMap: map[IsolationLevel]pgx.TxIsoLevel{
			LevelDefault:         "",
			LevelReadUncommitted: pgx.ReadUncommitted,
			LevelReadCommitted:   pgx.ReadCommitted,
			LevelWriteCommitted:  pgx.ReadCommitted,
			LevelRepeatableRead:  pgx.RepeatableRead,
			LevelSnapshot:        pgx.Serializable,
			LevelSerializable:    pgx.Serializable,
			LevelLinearizable:    pgx.Serializable,
		},
		deferrableModesMap: map[bool]pgx.TxDeferrableMode{
			true:  pgx.Deferrable,
			false: "",
		},
		accessModesMap: map[bool]pgx.TxAccessMode{
			true:  pgx.ReadOnly,
			false: pgx.ReadWrite,
		},
	}
}

func (s *pgxDB) BeginTx(opts TxOpts) (Tx[PGXDB], error) {
	o := pgx.TxOptions{
		IsoLevel:       s.isoLevelsMap[opts.Isolation],
		AccessMode:     s.accessModesMap[opts.ReadOnly],
		DeferrableMode: "",
		BeginQuery:     "",
		CommitQuery:    "",
	}

	if opts.Ext != nil {
		if ext, ok := opts.Ext.(TxOptsExt); ok {
			o.DeferrableMode = s.deferrableModesMap[ext.DeferrableMode]
			o.BeginQuery = ext.BeginQuery
			o.CommitQuery = ext.CommitQuery
		}
	}

	tx, err := s.db.BeginTx(opts.Ctx, o)
	if err != nil {
		return nil, err
	}

	return newTx(
		func() PGXDB { return tx },
		func(ctx context.Context) error { return tx.Commit(ctx) },
		func(ctx context.Context) error { return tx.Rollback(ctx) },
	), nil
}

func (s *pgxDB) GetDB(_ NoTxOpts) PGXDB { return s.rawDB }

type pgxRawDB struct {
	*pgxpool.Pool
}

func (s *pgxRawDB) Begin(_ context.Context) (pgx.Tx, error) { panic(errBeginNotSupported) }
func (s *pgxRawDB) Commit(_ context.Context) error          { panic(errCommitNotSupported) }
func (s *pgxRawDB) Rollback(_ context.Context) error        { panic(errRollbackNotSupported) }
func (s *pgxRawDB) LargeObjects() pgx.LargeObjects          { panic(errLargeObjectsNotSupported) }
func (s *pgxRawDB) Conn() *pgx.Conn                         { panic(errConnNotSupported) }
func (s *pgxRawDB) Prepare(_ context.Context, _, _ string) (*pgconn.StatementDescription, error) {
	panic(errPrepareNotSupported)
}
