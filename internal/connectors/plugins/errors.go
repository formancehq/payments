package plugins

import (
	"errors"
)

var (
	ErrNotImplemented       = errors.New("not implemented")
	ErrNotYetInstalled      = errors.New("not yet installed")
	ErrInvalidClientRequest = errors.New("invalid client request")
	ErrCurrencyNotSupported = errors.New("currency not supported")
)
