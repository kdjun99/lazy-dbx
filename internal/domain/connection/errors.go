package connection

import "errors"

// Sentinel errors for connection operations.
var (
	ErrConnectionFailed    = errors.New("connection failed")
	ErrConnectionNotInPool = errors.New("connection not in pool")
	ErrUnsupportedDriver   = errors.New("unsupported database driver")
)
