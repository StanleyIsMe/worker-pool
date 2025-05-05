package workerpool

import "errors"

var (
	ErrPoolClosed = errors.New("worker pool already closed")
)
