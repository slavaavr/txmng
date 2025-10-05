package txmng

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate mockgen -source=./db_provider_pgx.go -destination=./db_provider_pgx_mock.go -package txmng

type PGX interface {
	pgx.Tx
}

type pgxProvider struct {
	db    *pgxpool.Pool
	rawDB Tx[PGX]

	isoLevelsMap       map[IsolationLevel]pgx.TxIsoLevel
	deferrableModesMap map[bool]pgx.TxDeferrableMode
	accessModesMap     map[bool]pgx.TxAccessMode
}

type rawPGXAdapter struct {
	*pgxpool.Pool
}

func (s *rawPGXAdapter) Commit(ctx context.Context) error   { return ErrCommitNotSupported }
func (s *rawPGXAdapter) Rollback(ctx context.Context) error { return ErrRollbackNotSupported }
func (s *rawPGXAdapter) LargeObjects() pgx.LargeObjects     { return pgx.LargeObjects{} }
func (s *rawPGXAdapter) Conn() *pgx.Conn                    { return nil }
func (s *rawPGXAdapter) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("prepare statement is not supported")
}

func NewPGXProvider(db *pgxpool.Pool) DBProvider[PGX] {
	return &pgxProvider{
		db: db,
		rawDB: newTx[PGX](
			func() PGX { return &rawPGXAdapter{db} },
			func(ctx context.Context) error { return ErrCommitNotSupported },
			func(ctx context.Context) error { return ErrRollbackNotSupported },
		),
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

func (s *pgxProvider) BeginTx(opts Opts) (Tx[PGX], error) {
	if opts.UseRawDB() {
		return s.rawDB, nil
	}

	o := pgx.TxOptions{
		IsoLevel:       s.isoLevelsMap[opts.Isolation],
		AccessMode:     s.accessModesMap[opts.ReadOnly],
		DeferrableMode: "",
		BeginQuery:     "",
		CommitQuery:    "",
	}

	if opts.Ext != nil {
		if ext, ok := opts.Ext.(DefaultOptsExt); ok {
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
