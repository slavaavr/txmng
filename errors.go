package txmng

import "errors"

var (
	errInvalidContext           = errors.New("use of invalid context")
	errBeginNotSupported        = errors.New("begin not supported")
	errCommitNotSupported       = errors.New("commit not supported")
	errRollbackNotSupported     = errors.New("rollback not supported")
	errLargeObjectsNotSupported = errors.New("largeObjects not supported")
	errConnNotSupported         = errors.New("conn not supported")
	errPrepareNotSupported      = errors.New("prepare not supported")
)
