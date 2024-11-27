# SimpleQueue - A Golang Simple Internal Queue with SimpleWorker for Execution

SimpleQueue is a simple implementation of an internal queue in Golang, designed to handle tasks using a worker interface. It utilizes the SimpleWorker to process messages and provides options to customize its behavior.

## Installation

To install SimpleQueue, use the following command:

```sh
go get github.com/baro-oss/siq@latest
```

## Usage

Here's an example of how to use SimpleQueue with SimpleWorker:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/baro-oss/siq"
)

func main() {
	// Create a new SimpleWorker with the processMessage function
	worker := siq.NewSimpleWorker(
		func(value any) error {
			fmt.Println(value)
			return nil
		},
		// Set the function to determine whether a message should be retried
		siq.SWWithShouldRetry(func(err error) bool {
			return os.IsTimeout(err)
		}),
	)

	// New simple queue
	queue := siq.NewSimpleQueue(
		worker,
		siq.SQWithMaxWorker(2),
		siq.SQWithQueueLength(128))

	topic := siq.NewTopic("x-print-stdout")

	// should use group for easy to manage queues by topics
	queueGroup := siq.NewQueueGroup()
	err := queueGroup.AddQueue(topic, queue)
	if err != nil {
		panic(err)
	}

	// Send some messages to the worker
	for i := 0; i < 10; i++ {
		queueGroup.Publish(topic, fmt.Sprintf("message %d", i))
	}

	// Close all queue
	queueGroup.CloseAll()

	// Test publishing message to closed queues
	err = queueGroup.Publish(topic, fmt.Sprintf("message %d",11))
	if errors.Is(err,siq.ErrQueueClosed) {
		println("ok")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	queueGroup.Shutdown(ctx)
	
	println("queue shutdown successful")
}
```

## Features

- SimpleQueue provides a flexible and customizable way to handle tasks using a worker interface.
- It allows you to define your own function to process messages.
- It supports retrying failed tasks with a configurable retry channel and function.
- It provides an error handling mechanism to handle errors during processing.

## Documentation

For detailed documentation, check out the [GoDoc](https://pkg.go.dev/github.com/baro-oss/siq) for SimpleQueue and SimpleWorker.

## Contributing

Contributions are welcome! If you find any bugs or have feature requests, feel free to open an issue or submit a pull request.

## License

SimpleQueue is licensed under the MIT License. See the [LICENSE](https://github.com/baro-oss/siq/blob/main/LICENSE) file for more details.