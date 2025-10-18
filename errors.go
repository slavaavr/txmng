package txmng

import "errors"

var (
	errDBNotFound               = errors.New("db not found")
	errCommitNotSupported       = errors.New("commit not supported")
	errRollbackNotSupported     = errors.New("rollback not supported")
	errLargeObjectsNotSupported = errors.New("largeObjects not supported")
	errConnNotSupported         = errors.New("conn not supported")
	errPrepareNotSupported      = errors.New("prepare not supported")
)
