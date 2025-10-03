package txmng

import (
	"context"
	"database/sql"
)

type Opts struct {
	Ctx       context.Context
	Isolation IsolationLevel
	ReadOnly  bool
	Ext       any

	useRawDB bool
}

type DeferrableMode int

const (
	Deferrable DeferrableMode = iota
	NotDeferrable
)

type DefaultOptsExt struct {
	DeferrableMode DeferrableMode
	BeginQuery     string
	CommitQuery    string
}

func (t Opts) UseRawDB() bool {
	return t.useRawDB
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
