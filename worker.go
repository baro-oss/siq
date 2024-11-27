package siq

type WFunc func(any) error

type Worker interface {
	Exec(any)
	SetRetryChan(chan any)
	Clone() Worker
}
