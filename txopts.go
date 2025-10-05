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

type DefaultOptsExt struct {
	DeferrableMode bool
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
