package queue

import (
	"context"
	"fmt"

	"kombiner/pkg/apis/kombiner/v1alpha1"
)

// QueueIterator is an entity that iterates over multiple queues popping
// PlacementRequests from them respecting their Weight.
type QueueIterator struct {
	Next          chan *v1alpha1.PlacementRequest
	readerFactory ReaderFactory
	configs       QueueConfigs
	resume        chan bool
}

// Resume ensures we have a resume signal ready to be intercepted by the Run()
// loop. This signal is used so inform the loop that new elements have been
// added to one of the queues. If the signal channel is full then we can move
// on as we already have a "resume" scheduled.
func (q *QueueIterator) Resume() {
	select {
	case q.resume <- true:
	default:
	}
}

// Run starts the queue iterator. We start with a list with all queues and we
// select one based on the queue weights. If the selected queue delivers us a
// message we send it to the Next channel and restart the loop. If the selected
// queue does not deliver us a message we exclude it from the list of repeat
// the process. We keep doing this until we either find a message in one of the
// queues or we found all queues to be empty. In the latter case we then wait
// for a resume signal to be sent by the PushHandler of one of the queues.
func (q *QueueIterator) Run(ctx context.Context) {
	defer close(q.Next)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		configs := QueueConfigs{}
		configs = append(configs, q.configs...)

		reader := q.readerFactory(configs)
		for p := reader.Read(ctx); p != nil; p = reader.Read(ctx) {
			select {
			case <-ctx.Done():
			case q.Next <- p:
			}
		}

		// nothing found to read in the queues, we now need to wait for
		// the resume signal so we can resume reading.
		select {
		case <-ctx.Done():
		case <-q.resume:
		}
	}
}

// NewQueueIterator creates a queue iterator based on the provided QueueConfig
// objects. This function registers a custom push handler for each queue so it
// is capable of resuming reading from queues.
func NewQueueIterator(configs QueueConfigs, opts ...QueueIteratorOption) (*QueueIterator, error) {
	if err := configs.Validate(); err != nil {
		return nil, fmt.Errorf("invalid queue configuration: %w", err)
	}

	it := &QueueIterator{
		Next:          make(chan *v1alpha1.PlacementRequest),
		readerFactory: NewUniformReader,
		resume:        make(chan bool, 2),
		configs:       configs,
	}

	configs.AddPushHandler(it.Resume)

	for _, opt := range opts {
		opt(it)
	}

	return it, nil
}
