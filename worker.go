package siq

type WFunc func(any) error

type Worker interface {
	Exec(any)
	Clone() Worker
}
