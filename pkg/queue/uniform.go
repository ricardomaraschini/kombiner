package queue

import (
	"context"
	"kombiner/pkg/apis/kombiner/v1alpha1"
	"math/rand"
	"slices"
)

// UniformReader is a queue reader that reads messages from multiple queues in
// a uniform manner based on the weights provided in the QueueConfig. It uses a
// weighted random selection algorithm to determine the next queue to process.
// The weights are used to determine the probability of selecting a queue,
// allowing for a more balanced distribution of messages.
type UniformReader struct {
	configs QueueConfigs
}

// Read is needed in order to select the next message out of a list of queues.
// This function returns a *v1alpha1.PlacementRequest or nil if there were none
// to be read (all queues are empty).
func (u *UniformReader) Read(_ context.Context) *v1alpha1.PlacementRequest {
	configs := QueueConfigs{}
	configs = append(configs, u.configs...)

	nrconfigs := len(configs)
	for range nrconfigs {
		qidx := u.next(configs)
		if pr := configs[qidx].QueueRef.Pop(); pr != nil {
			return pr
		}

		// the previous queue returned no message so we can remove it
		// from the list of queues and try again with the remaining
		// ones. we keep doing this until no more queues are left.
		configs = slices.Delete(configs, qidx, qidx+1)
	}
	return nil
}

// next return the next queue from each we should read a message from. This
// function is always expected to find a next queue or panic if it doesn't as
// this should never happen.
func (u *UniformReader) next(configs QueueConfigs) int {
	var total int
	for _, config := range configs {
		total += int(config.Weight)
	}

	// select a random number between 1 and total + 1. this random number
	// will fit somewhere in the sum of all individual weights.
	selected := rand.Intn(total) + 1

	// we keep summing the weights until we find the sum to be greater than
	// or equal to the selected number. this means that the queue at that
	// index is the one we should process next.
	var sum int
	for i, config := range configs {
		if sum += int(config.Weight); selected <= sum {
			return i
		}
	}

	// this should never happen as we always select a number between 1 and
	// total + 1, which is the sum of all weights.
	panic("no queue selected, this should never happen")
}

// NewUniformReader creates a new UniformReader with the provided QueueConfig
// objects. This function is used to initialize the reader with the queues
// that it should read messages from.
func NewUniformReader(configs QueueConfigs) Reader {
	return &UniformReader{
		configs: configs,
	}
}
