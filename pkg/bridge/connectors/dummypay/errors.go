package dummypay

import "github.com/pkg/errors"

var (
	ErrMissingDirectory = errors.New("missing directory to watch")
	ErrMissingTask      = errors.New("task is not implemented")
)
