package siq

import (
	"errors"
)

var (
	ErrInvalidTopic = errors.New("invalid topic")
	ErrQueueClosed  = errors.New("push message to a closed queue")
	ErrQueueExists  = errors.New("queue already exists for the given topic")
)

type Queue interface {
	Push(msg ...any)
	Len() int
	Close()
	IsClosed() bool
}
