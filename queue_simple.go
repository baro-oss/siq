package siq

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	MaxQueueSize = 1024
	MaxWorker    = 10
)

type SimpleQueueOtpFunc func(*SimpleQueue)

type SimpleQueue struct {
	queue            chan any
	workerQueue      chan Worker
	numOfWorkers     int64
	mu               sync.Locker
	worker           Worker
	isClosed         atomic.Bool
	initialQueueLen  int
	initialMaxWorker int
}

// NewSimpleQueue creates a new instance of SimpleQueue with the provided worker and optional options.
// The function initializes the queue and worker queue with default values or custom values provided through options.
// It starts running the queue and load balancer in separate goroutines.
//
// Parameters:
//   - worker: The Worker interface implementation that will be used to execute tasks in the queue.
//   - opts: A variadic parameter that accepts optional functions of type SimpleQueueOtpFunc.
//     These functions can be used to customize the queue's behavior by setting custom queue length or maximum worker count.
//
// Returns:
// - A pointer to the newly created SimpleQueue instance.
func NewSimpleQueue(worker Worker, opts ...SimpleQueueOtpFunc) *SimpleQueue {
	cmdQueue := &SimpleQueue{
		mu:     &sync.Mutex{},
		worker: worker,
	}

	for _, opt := range opts {
		opt(cmdQueue)
	}

	if cmdQueue.queue == nil {
		cmdQueue.queue = make(chan any, MaxQueueSize)
		cmdQueue.initialQueueLen = MaxQueueSize
	}

	if cmdQueue.workerQueue == nil {
		cmdQueue.workerQueue = make(chan Worker, MaxWorker)
		cmdQueue.initialMaxWorker = MaxWorker
	}

	cmdQueue.increaseWorker(1)

	go cmdQueue.Run()
	go cmdQueue.StartLoadBalancer()

	return cmdQueue
}

// Run starts processing tasks from the queue in a loop.
// It continuously retrieves a worker from the worker queue, executes a task from the queue using the worker,
// and then returns the worker back to the worker queue.
// This function runs indefinitely until the queue is closed.
func (q *SimpleQueue) Run() {
	for {
		w := <-q.workerQueue
		go func() {
			w.Exec(<-q.queue)
			q.workerQueue <- w
		}()
	}
}

// Push adds one or more tasks to the SimpleQueue.
// It accepts a variadic parameter 'values' of type 'any' (which can be any type) and adds them to the queue.
// If the queue is full, the function will block until space becomes available.
//
// Parameters:
//   - values: A variadic parameter that accepts one or more tasks to be added to the queue.
//     The tasks can be of any type.
func (q *SimpleQueue) Push(values ...any) {
	for i := range values {
		q.queue <- values[i]
	}
}

func (q *SimpleQueue) Len() int {
	return len(q.queue)
}

// increaseWorker increases the number of workers in the SimpleQueue by the specified delta.
// It locks the mutex to ensure thread safety while updating the number of workers.
// The function also adds the specified delta to the atomic counter for the number of workers.
// Finally, it clones a worker from the SimpleQueue's worker and adds it to the worker queue for each delta.
//
// Parameters:
//   - delta: The number of workers to increase. It can be positive or negative.
func (q *SimpleQueue) increaseWorker(delta int64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.numOfWorkers += delta
	atomic.AddInt64(&q.numOfWorkers, delta)

	for i := 0; i < int(delta); i++ {
		q.workerQueue <- q.worker.Clone()
	}
}

// decreaseWorker decreases the number of workers in the SimpleQueue by the specified delta.
// It ensures that at least one worker is always running by limiting the delta to the current number of workers.
// The function locks the mutex to ensure thread safety while updating the number of workers.
// It then adds the negative delta to the atomic counter for the number of workers and removes the specified delta from the worker queue.
//
// Parameters:
//   - delta: The number of workers to decrease. It must be a positive integer.
//     If delta is greater than or equal to the current number of workers, delta is set to the current number of workers minus one.
func (q *SimpleQueue) decreaseWorker(delta int64) {
	// keep least one worker
	if delta >= q.numOfWorkers {
		delta = q.numOfWorkers
		if delta > 1 {
			delta -= 1
		}
	}

	atomic.AddInt64(&q.numOfWorkers, 0-delta)

	for i := 0; i < int(delta); i++ {
		<-q.workerQueue
	}
}

// StartLoadBalancer periodically checks the queue's length and the number of workers to balance the workload.
// It runs in a separate goroutine and uses a ticker to execute every 30 seconds.
//
// The function checks the following conditions:
//   - If the queue's length equals the initial queue length, it increases the number of workers by 1.
//   - If the queue's length is less than 10% of the initial maximum worker count,
//     and the current number of workers is less than the initial maximum worker count,
//     and the current number of workers is greater than 1, it decreases the number of workers by 1.
//
// The function continues to run until the queue is closed.
func (q *SimpleQueue) StartLoadBalancer() {
	ticker := time.NewTicker(30 * time.Second)

	for {
		select {
		case <-ticker.C:
			if q.isClosed.Load() {
				return
			}

			if q.Len() == q.initialQueueLen {
				q.increaseWorker(1)
				continue
			}

			if q.Len() < q.initialMaxWorker/10 && q.numOfWorkers < int64(q.initialMaxWorker) && q.numOfWorkers > 1 {
				q.decreaseWorker(1)
				continue
			}
		}
	}
}

func (q *SimpleQueue) Close() {
	q.isClosed.Store(true)
}

func (q *SimpleQueue) IsClosed() bool {
	return q.isClosed.Load()
}

// SQWithMaxWorker is a function that returns a SimpleQueueOtpFunc with a custom maximum worker count.
// This function is used as an option when creating a new SimpleQueue instance to customize the maximum worker count.
//
// Parameters:
//   - maxWorker: An integer representing the maximum number of workers to be used in the SimpleQueue.
//
// Returns:
//   - A SimpleQueueOtpFunc that sets the maximum worker count in the provided SimpleQueue instance.
func SQWithMaxWorker(maxWorker int) SimpleQueueOtpFunc {
	return func(q *SimpleQueue) {
		q.workerQueue = make(chan Worker, maxWorker)
		q.initialMaxWorker = maxWorker
	}
}

// SQWithQueueLength is a function that returns a SimpleQueueOtpFunc with a custom queue length.
// This function is used as an option when creating a new SimpleQueue instance to customize the queue length.
//
// Parameters:
//   - length: An integer representing the maximum capacity of the queue.
//
// Returns:
//   - A SimpleQueueOtpFunc that sets the queue length in the provided SimpleQueue instance.
//
// The returned SimpleQueueOtpFunc creates a new channel with the specified length for the queue,
// and sets the initialQueueLen field of the provided SimpleQueue instance to the specified length.
func SQWithQueueLength(length int) SimpleQueueOtpFunc {
	return func(q *SimpleQueue) {
		q.queue = make(chan any, length)
		q.initialQueueLen = length
	}
}
