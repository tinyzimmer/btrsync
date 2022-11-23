package receivers

import "errors"

var (
	ErrNotSupported = errors.New("operation not supported by receiver")
)
