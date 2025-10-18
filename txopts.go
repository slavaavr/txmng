package txmng

import (
	"context"
	"database/sql"
)

type TxOpts struct {
	Ctx       context.Context
	Isolation IsolationLevel
	ReadOnly  bool
	Ext       any
}

type TxOptsExt struct {
	DeferrableMode bool
	BeginQuery     string
	CommitQuery    string
}

type NoTxOpts struct {
	Ctx context.Context
	Ext any
}

type IsolationLevel int

const (
	LevelDefault IsolationLevel = iota
	LevelReadUncommitted
	LevelReadCommitted
	LevelWriteCommitted
	LevelRepeatableRead
	LevelSnapshot
	LevelSerializable
	LevelLinearizable
)

func (t IsolationLevel) toSQL() sql.IsolationLevel {
	return sql.IsolationLevel(t)
}
