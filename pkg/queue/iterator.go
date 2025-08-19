package queue

import (
	"context"
	"fmt"
	"math/rand"
	"slices"
	"sync"

	"kombiner/pkg/apis/kombiner/v1alpha1"
)

// QueueIterator is an entity that iterates over multiple queues popping
// PlacementRequests from them respecting their Weight.
type QueueIterator struct {
	Next    chan *v1alpha1.PlacementRequest
	mtx     sync.Mutex
	configs QueueConfigs
	resume  chan bool
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

		var cfgs []QueueConfig

		q.mtx.Lock()
		cfgs = append(cfgs, q.configs...)
		q.mtx.Unlock()

		allempty, nrconfigs := true, len(cfgs)
		for range nrconfigs {
			if i, end := q.readOne(ctx, cfgs); !end {
				cfgs = slices.Delete(cfgs, i, i+1)
				continue
			}

			allempty = false
			break
		}

		if allempty {
			select {
			case <-ctx.Done():
			case <-q.resume:
			}
		}
	}
}

// readOne reads a single PlacementRequest from the queues based on their
// weights. It selects the next queue to process and attempts to pop a
// PlacementRequest from it. If a request is found this function returns
// true, otherwise it returns false. The index of the selected queue is
// also returned. If a message is found it is written to the Next channel
// so this function may block.
func (q *QueueIterator) readOne(ctx context.Context, configs []QueueConfig) (int, bool) {
	index := q.selectNextQueue(configs)
	if pr := configs[index].Queue.Pop(); pr != nil {
		select {
		case <-ctx.Done():
		case q.Next <- pr:
		}
		return index, true
	}
	return index, false
}

// selectNextQueue uses the weights to select the next queue to process. It
// returns the index of the selected queue based on the weights provided in
// the QueueConfig. The selection is done in a weighted random manner.
func (q *QueueIterator) selectNextQueue(configs []QueueConfig) int {
	var total uint
	for _, config := range configs {
		total += config.Weight
	}

	// select a random number between 1 and total + 1. this random number
	// will fit somewhere in the sum of all individual weights.
	selected := uint(rand.Intn(int(total)) + 1)

	// we keep summing the weights until we find the sum to be greater than
	// or equal to the selected number. this means that the queue at that
	// index is the one we should process next.
	var sum uint
	for i, config := range configs {
		if sum += config.Weight; selected <= sum {
			return i
		}
	}

	// this should never happen as we always select a number between 1 and
	// total + 1, which is the sum of all weights.
	panic("no queue selected, this should never happen")
}

// NewQueueIterator creates a queue iterator based on the provided QueueConfig
// objects. This function registers a custom push handler for each queue so it
// is capable of resuming reading from queues.
func NewQueueIterator(configs QueueConfigs) (*QueueIterator, error) {
	if err := configs.Validate(); err != nil {
		return nil, fmt.Errorf("invalid queue configuration: %w", err)
	}

	it := &QueueIterator{
		Next:    make(chan *v1alpha1.PlacementRequest),
		resume:  make(chan bool, 2),
		configs: configs,
	}

	configs.AddPushHandler(it.Resume)
	return it, nil
}
