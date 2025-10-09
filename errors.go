package txmng

import "errors"

var (
	ErrDBNotFound           = errors.New("db not found")
	ErrCommitNotSupported   = errors.New("commit is not supported")
	ErrRollbackNotSupported = errors.New("rollback is not supported")
)
