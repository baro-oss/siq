package siq

import "sync"

type SimpleWorkerOptFunc func(*SimpleWorker)

type SimpleWorker struct {
	isIdle      bool
	execFunc    WFunc
	mu          sync.Mutex
	retryChan   chan any
	shouldRetry func(error) bool
}

// NewSimpleWorker creates a new instance of SimpleWorker with the provided function and optional options.
//
// Parameters:
// - wFunc: The function to be executed by the worker. It should accept any type of message and return an error.
// - opts: A variadic parameter that accepts zero or more SimpleWorkerOptFunc functions. These functions are used to configure the worker.
//
// Returns:
// - A pointer to the newly created SimpleWorker instance.
func NewSimpleWorker(wFunc WFunc, opts ...SimpleWorkerOptFunc) *SimpleWorker {
	worker := &SimpleWorker{
		isIdle:   true,
		execFunc: wFunc,
		mu:       sync.Mutex{},
	}
	for _, opt := range opts {
		opt(worker)
	}
	return worker
}

// Exec executes the worker's function with the provided message.
// If the worker is not idle, it will enqueue the message for retry if a retry channel is provided.
// If the worker's function returns an error and a retry function is provided, it will enqueue the message for retry.
//
// Parameters:
// - msg: The message to be processed by the worker's function. It can be of any type.
func (w *SimpleWorker) Exec(msg any) {
	w.mu.Lock()
	if !w.isIdle {
		if w.retryChan != nil {
			w.retryChan <- msg
		}
		return
	}

	w.isIdle = false
	w.mu.Unlock()

	err := w.execFunc(msg)
	if err != nil && w.shouldRetry != nil && w.shouldRetry(err) {
		if w.retryChan != nil {
			w.retryChan <- msg
		}
	}

	w.mu.Lock()
	w.isIdle = true
	w.mu.Unlock()
}

// Clone creates a new instance of SimpleWorker with the same properties as the current instance.
// It returns a pointer to the cloned SimpleWorker instance.
//
// The cloned SimpleWorker will have the same values for the following fields:
// - isIdle: Indicates whether the worker is currently idle.
// - execFunc: The function to be executed by the worker.
// - mu: A mutex for synchronization.
// - shouldRetry: A function that determines whether a message should be retried.
// - retryChan: A channel for retrying messages.
//
// The cloned SimpleWorker will have a new sync.Mutex instance, but it will share the same execFunc, shouldRetry, and retryChan as the original instance.
func (w *SimpleWorker) Clone() Worker {
	return &SimpleWorker{
		isIdle:      w.isIdle,
		execFunc:    w.execFunc,
		mu:          sync.Mutex{},
		shouldRetry: w.shouldRetry,
		retryChan:   w.retryChan,
	}
}

// SWWithRetryChan is an option function for creating a SimpleWorker instance.
// It sets the retry channel for the worker, allowing it to enqueue messages for retry.
//
// Parameters:
// - retryChan: A channel of type any where messages will be enqueued for retry.
//
// Returns:
// - A SimpleWorkerOptFunc that sets the retry channel for the worker.
func SWWithRetryChan(retryChan chan any) SimpleWorkerOptFunc {
	return func(worker *SimpleWorker) {
		worker.mu.Lock()
		worker.retryChan = retryChan
		worker.mu.Unlock()
	}
}

// SWWithShouldRetry is an option function for creating a SimpleWorker instance.
// It sets the function to determine whether a message should be retried based on the error returned by the worker's function.
//
// Parameters:
//   - f: A function that accepts an error and returns a boolean value.
//     If the function returns true, the message will be enqueued for retry.
//     If the function returns false, the message will not be retried.
//
// Returns:
// - A SimpleWorkerOptFunc that sets the shouldRetry function for the worker.
func SWWithShouldRetry(f func(err error) bool) SimpleWorkerOptFunc {
	return func(worker *SimpleWorker) {
		worker.mu.Lock()
		worker.shouldRetry = f
		worker.mu.Unlock()
	}
}
