package siq

import (
	"context"
	"time"
)

type QueueGroup struct {
	queue map[Topic]Queue
}

func NewQueueGroup() *QueueGroup {
	return &QueueGroup{queue: make(map[Topic]Queue)}
}

// AddQueue adds a new queue to the QueueGroup for a specific topic.
//
// Parameters:
//   - topic: The Topic to associate with the queue.
//   - queue: The Queue to be added for the given topic.
//
// Returns:
//   - error: Returns nil if the queue is successfully added.
//     Returns ErrQueueExists if a queue already exists for the given topic.
func (qg *QueueGroup) AddQueue(topic Topic, queue Queue) error {
	_, ok := qg.queue[topic]
	if ok {
		return ErrQueueExists
	}

	qg.queue[topic] = queue
	return nil
}

// Publish publishes one or more messages to a specific topic in the QueueGroup.
//
// Parameters:
//   - topic: The Topic to which the messages should be published.
//   - msg: A variadic parameter of type 'any' representing one or more messages to be published.
//
// Returns:
//   - error: Returns nil if the messages are successfully published.
//     Returns ErrInvalidTopic if the specified topic does not exist in the QueueGroup.
//     Returns ErrQueueClosed if the queue for the specified topic is closed.
func (qg *QueueGroup) Publish(topic Topic, msg ...any) error {
	if qg.queue == nil {
		qg.queue = make(map[Topic]Queue)
	}

	q, ok := qg.queue[topic]
	if !ok {
		return ErrInvalidTopic
	}

	if q.IsClosed() {
		return ErrQueueClosed
	}

	q.Push(msg...)
	return nil
}

// Shutdown gracefully shuts down the QueueGroup by periodically checking
// if all queues have finished processing their messages. It will continue
// to check until all queues are empty or the context is canceled.
//
// Parameters:
//   - ctx: A context that can be used to cancel the shutdown process.
//     If the context is canceled, the shutdown will terminate immediately.
func (qg *QueueGroup) Shutdown(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			var isConsuming bool

			for _, v := range qg.queue {
				if v.Len() > 0 {
					isConsuming = true
					break
				}
			}

			if isConsuming {
				continue
			}
			return
		case <-ctx.Done():
			return
		}
	}
}

// CloseAll closes all queues in the QueueGroup, preventing any further messages
// from being pushed to them. This is useful for stopping all message processing
// in preparation for shutdown or maintenance.
func (qg *QueueGroup) CloseAll() {
	for _, v := range qg.queue {
		v.Close()
	}
}
