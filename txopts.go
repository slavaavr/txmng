package txmng

import (
	"context"
	"database/sql"
)

type Opts struct {
	Ctx       context.Context
	Isolation IsolationLevel
	ReadOnly  bool
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
