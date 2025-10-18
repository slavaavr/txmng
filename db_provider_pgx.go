package txmng

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PGX interface {
	pgx.Tx
}

type pgxProvider struct {
	db    *pgxpool.Pool
	rawDB PGX

	isoLevelsMap       map[IsolationLevel]pgx.TxIsoLevel
	deferrableModesMap map[bool]pgx.TxDeferrableMode
	accessModesMap     map[bool]pgx.TxAccessMode
}

type pgxAdapter struct {
	*pgxpool.Pool
}

func (s *pgxAdapter) Commit(_ context.Context) error   { panic(errCommitNotSupported) }
func (s *pgxAdapter) Rollback(_ context.Context) error { panic(errRollbackNotSupported) }
func (s *pgxAdapter) LargeObjects() pgx.LargeObjects   { panic(errLargeObjectsNotSupported) }
func (s *pgxAdapter) Conn() *pgx.Conn                  { panic(errConnNotSupported) }
func (s *pgxAdapter) Prepare(_ context.Context, _, _ string) (*pgconn.StatementDescription, error) {
	panic(errPrepareNotSupported)
}

func NewPGXProvider(db *pgxpool.Pool) DBProvider[PGX] {
	return &pgxProvider{
		db: db,
		rawDB: &pgxAdapter{
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

func (s *pgxProvider) BeginTx(opts TxOpts) (Tx[PGX], error) {
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
		func() PGX { return tx },
		func(ctx context.Context) error { return tx.Commit(ctx) },
		func(ctx context.Context) error { return tx.Rollback(ctx) },
	), nil
}

func (s *pgxProvider) GetDB(_ NoTxOpts) PGX { return s.rawDB }
